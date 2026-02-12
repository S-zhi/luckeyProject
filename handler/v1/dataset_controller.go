package v1

import (
	"errors"
	"lucky_project/dao"
	entity2 "lucky_project/entity"
	"lucky_project/service"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DatasetController struct {
	datasetService  *service.DatasetService
	uploadService   *service.UploadService
	downloadService *service.BaiduDownloadService
}

func NewDatasetController() *DatasetController {
	return &DatasetController{
		datasetService:  service.NewDatasetService(),
		uploadService:   service.NewUploadService(),
		downloadService: service.NewBaiduDownloadService(),
	}
}

// CreateDataset handles POST /v1/datasets
func (c *DatasetController) CreateDataset(ctx *gin.Context) {
	var dataset entity2.Dataset
	if err := ctx.ShouldBindJSON(&dataset); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.datasetService.CreateDataset(ctx.Request.Context(), &dataset); err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, dataset)
}

// GetAllDatasets handles GET /v1/datasets
func (c *DatasetController) GetAllDatasets(ctx *gin.Context) {
	var params entity2.QueryParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.datasetService.GetAllDatasets(ctx.Request.Context(), params)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// GetDatasetStorageServers handles GET /v1/datasets/:id/storage-server
func (c *DatasetController) GetDatasetStorageServers(ctx *gin.Context) {
	id, err := parseUintPathParam(ctx, "id")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	servers, err := c.datasetService.GetStorageServersByID(ctx.Request.Context(), id)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, buildStorageServerResponse(id, servers))
}

// UpdateDatasetStorageServers handles PATCH /v1/datasets/:id/storage-server
func (c *DatasetController) UpdateDatasetStorageServers(ctx *gin.Context) {
	id, err := parseUintPathParam(ctx, "id")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var payload storageServerUpdatePayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action, servers := normalizeStorageServerPayload(payload)
	updated, err := c.datasetService.UpdateStorageServersByID(ctx.Request.Context(), id, action, servers)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, buildStorageServerResponse(id, updated))
}

// UploadDatasetFile handles POST /v1/datasets/upload
func (c *DatasetController) UploadDatasetFile(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	subdir := ctx.PostForm("subdir")
	_ = subdir // deprecated: fixed root strategy no longer uses subdir for path resolution.
	artifactName := ctx.PostForm("artifact_name")
	storageTarget := ctx.PostForm("storage_target")
	storageServer := ctx.PostForm("storage_server")
	uploadToBaidu, err := parseOptionalBoolForm(ctx, "upload_to_baidu", false)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.uploadService.SaveDatasetFile(file, artifactName, storageTarget, storageServer, uploadToBaidu)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidUploadFile),
			errors.Is(err, service.ErrInvalidUploadSubdir),
			errors.Is(err, service.ErrInvalidStorageTarget):
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	affectedRows, sizeMB, err := c.datasetService.SyncSizeByFileName(ctx.Request.Context(), result.FileName, result.Size)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message":         "upload success",
		"file_name":       result.FileName,
		"resolved_path":   result.ResolvedPath,
		"saved_path":      result.SavedPath,
		"paths":           result.Paths,
		"size":            result.Size,
		"size_mb":         sizeMB,
		"mysql_updated":   affectedRows > 0,
		"mysql_affected":  affectedRows,
		"storage_server":  result.StorageServer,
		"storage_target":  result.StorageTarget,
		"upload_to_baidu": result.UploadToBaidu,
		"baidu_uploaded":  result.BaiduUploaded,
		"baidu_path":      result.BaiduPath,
	})
}

// DownloadDatasetFile handles GET /v1/datasets/:id/download.
func (c *DatasetController) DownloadDatasetFile(ctx *gin.Context) {
	id, err := parseUintPathParam(ctx, "id")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dataset, err := c.datasetService.GetByID(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
			return
		}
		writeHTTPError(ctx, err)
		return
	}

	fileName := strings.TrimSpace(dataset.FileName)
	if fileName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "dataset file_name is empty"})
		return
	}
	fileName = filepath.Base(fileName)

	localPath, err := c.datasetService.ResolveFilePathByID(ctx.Request.Context(), id, service.StorageTargetBackend)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	servers, err := c.datasetService.GetStorageServersByID(ctx.Request.Context(), id)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	if fileExists(localPath) {
		ctx.FileAttachment(localPath, fileName)
		return
	}

	if !containsBaiduStorage(servers) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "dataset file not found in backend and baidu_netdisk is not configured"})
		return
	}

	if c.downloadService == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "download service is nil"})
		return
	}

	remotePath, err := c.datasetService.ResolveFilePathByID(ctx.Request.Context(), id, service.StorageTargetBaiduNetdisk)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	result, err := c.downloadService.DownloadToLocal(remotePath, service.ArtifactCategoryDatasets, fileName)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidDownloadCategory),
			errors.Is(err, service.ErrInvalidBaiduDownloadPath),
			errors.Is(err, service.ErrInvalidBaiduDownloadFile),
			errors.Is(err, service.ErrInvalidLocalDownloadFile),
			errors.Is(err, service.ErrBaiduDownloadTargetRequired),
			errors.Is(err, service.ErrBaiduPanAccessTokenRequired),
			errors.Is(err, service.ErrInvalidStorageTarget):
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	localPath = result.LocalPath
	if !fileExists(localPath) {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "downloaded file not found in backend storage"})
		return
	}

	if _, err := c.datasetService.UpdateStorageServersByID(
		ctx.Request.Context(),
		id,
		dao.StorageActionAdd,
		[]string{service.StorageTargetBackend},
	); err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.FileAttachment(localPath, fileName)
}

// UpdateDatasetMetadata handles PATCH /v1/datasets/:id.
func (c *DatasetController) UpdateDatasetMetadata(ctx *gin.Context) {
	id, err := parseUintPathParam(ctx, "id")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var payload map[string]interface{}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates, err := parseDatasetMetadataUpdates(payload)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dataset, err := c.datasetService.UpdateDatasetMetadata(ctx.Request.Context(), id, updates)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dataset)
}

// DeleteDatasetByFileName handles DELETE /v1/datasets/by-filename?file_name=xxx.
func (c *DatasetController) DeleteDatasetByFileName(ctx *gin.Context) {
	fileName := strings.TrimSpace(ctx.Query("file_name"))
	if fileName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file_name is required"})
		return
	}

	result, err := c.datasetService.DeleteByFileName(ctx.Request.Context(), fileName)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":            "delete success",
		"file_name":          result.FileName,
		"deleted_records":    result.DeletedRecords,
		"local_file_deleted": result.LocalFileDeleted,
	})
}
