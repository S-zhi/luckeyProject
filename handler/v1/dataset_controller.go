package v1

import (
	"errors"
	"lucky_project/config"
	entity2 "lucky_project/entity"
	"lucky_project/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DatasetController struct {
	datasetService *service.DatasetService
	uploadService  *service.UploadService
}

func NewDatasetController() *DatasetController {
	return &DatasetController{
		datasetService: service.NewDatasetService(),
		uploadService:  service.NewUploadService(),
	}
}

// CreateDataset handles POST /v1/datasets
func (c *DatasetController) CreateDataset(ctx *gin.Context) {
	config.AppLogger.Info("CreateDataset started")

	var dataset entity2.Dataset
	if err := ctx.ShouldBindJSON(&dataset); err != nil {
		config.AppLogger.Error("Failed to bind JSON: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.AppLogger.Info("Received dataset: %+v", dataset)

	if err := c.datasetService.CreateDataset(ctx.Request.Context(), &dataset); err != nil {
		config.AppLogger.Error("Failed to create dataset: %v", err)
		writeHTTPError(ctx, err)
		return
	}

	config.AppLogger.Info("Dataset created successfully")
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

// UploadDatasetFile handles POST /v1/datasets/upload
func (c *DatasetController) UploadDatasetFile(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	subdir := ctx.PostForm("subdir")
	storageServer := ctx.PostForm("storage_server")
	uploadToBaidu, err := parseOptionalBoolForm(ctx, "upload_to_baidu", false)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.uploadService.SaveDatasetFile(file, subdir, storageServer, uploadToBaidu)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidUploadFile), errors.Is(err, service.ErrInvalidUploadSubdir):
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message":         "upload success",
		"file_name":       result.FileName,
		"saved_path":      result.SavedPath,
		"size":            result.Size,
		"storage_server":  result.StorageServer,
		"upload_to_baidu": result.UploadToBaidu,
		"baidu_uploaded":  result.BaiduUploaded,
		"baidu_path":      result.BaiduPath,
	})
}
