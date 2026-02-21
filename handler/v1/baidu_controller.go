package v1

import (
	"errors"
	"fmt"
	"lucky_project/dao"
	"lucky_project/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type BaiduController struct {
	downloadService *service.BaiduDownloadService
	modelService    *service.ModelService
	datasetService  *service.DatasetService
	pathService     *service.ArtifactPathService
}

type BaiduDownloadRequest struct {
	RemotePath         string `json:"remote_path"`
	Category           string `json:"category"`
	Subdir             string `json:"subdir"`
	FileName           string `json:"file_name"`
	StorageTarget      string `json:"storage_target"`
	ModelID            *uint  `json:"model_id"`
	ModelName          string `json:"model_name"`
	DatasetID          *uint  `json:"dataset_id"`
	DatasetName        string `json:"dataset_name"`
	LocalStorageServer string `json:"local_storage_server"`
}

func NewBaiduController() *BaiduController {
	return &BaiduController{
		downloadService: service.NewBaiduDownloadService(),
		modelService:    service.NewModelService(),
		datasetService:  service.NewDatasetService(),
		pathService:     service.NewArtifactPathService(),
	}
}

// DownloadFileToLocal handles POST /v1/baidu/download.
func (c *BaiduController) DownloadFileToLocal(ctx *gin.Context) {
	var req BaiduDownloadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = req.Subdir // deprecated with fixed path strategy

	syncTarget, err := c.resolveSyncTarget(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	remotePath, category, fileName, err := c.resolveDownloadInput(ctx, req, syncTarget)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.downloadService.DownloadToLocal(remotePath, category, fileName)
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

	resp := gin.H{
		"message":     "下载成功",
		"remote_path": result.RemotePath,
		"local_path":  result.LocalPath,
		"file_name":   result.FileName,
		"category":    result.Category,
		"size":        result.Size,
	}

	if syncTarget.kind != "" {
		localStorageServer := strings.TrimSpace(req.LocalStorageServer)
		if localStorageServer == "" {
			localStorageServer = service.StorageTargetBackend
		}

		servers, err := c.syncRecordStorageServer(ctx, syncTarget, localStorageServer)
		if err != nil {
			writeHTTPError(ctx, err)
			return
		}

		resp["record_synced"] = true
		resp["record_type"] = syncTarget.kind
		resp["record_id"] = syncTarget.id
		resp["storage_servers"] = servers
		if len(servers) > 0 {
			resp["storage_server"] = servers[0]
		} else {
			resp["storage_server"] = ""
		}
	} else {
		resp["record_synced"] = false
	}

	ctx.JSON(http.StatusOK, resp)
}

type storageSyncTarget struct {
	kind string
	id   uint
}

func (c *BaiduController) resolveSyncTarget(ctx *gin.Context, req BaiduDownloadRequest) (storageSyncTarget, error) {
	modelTarget := req.ModelID != nil || strings.TrimSpace(req.ModelName) != ""
	datasetTarget := req.DatasetID != nil || strings.TrimSpace(req.DatasetName) != ""

	if modelTarget && datasetTarget {
		return storageSyncTarget{}, fmt.Errorf("模型目标和数据集目标只能提供一个")
	}

	if modelTarget {
		if req.ModelID != nil {
			return storageSyncTarget{kind: "model", id: *req.ModelID}, nil
		}
		model, err := c.modelService.FindByName(ctx.Request.Context(), strings.TrimSpace(req.ModelName))
		if err != nil {
			return storageSyncTarget{}, err
		}
		return storageSyncTarget{kind: "model", id: model.ID}, nil
	}

	if datasetTarget {
		if req.DatasetID != nil {
			return storageSyncTarget{kind: "dataset", id: *req.DatasetID}, nil
		}
		dataset, err := c.datasetService.FindByName(ctx.Request.Context(), strings.TrimSpace(req.DatasetName))
		if err != nil {
			return storageSyncTarget{}, err
		}
		return storageSyncTarget{kind: "dataset", id: dataset.ID}, nil
	}

	return storageSyncTarget{}, nil
}

func (c *BaiduController) resolveDownloadInput(ctx *gin.Context, req BaiduDownloadRequest, target storageSyncTarget) (string, string, string, error) {
	remotePath := strings.TrimSpace(req.RemotePath)
	category := strings.TrimSpace(req.Category)
	fileName := strings.TrimSpace(req.FileName)

	if remotePath != "" {
		if category == "" {
			if target.kind == "dataset" {
				category = service.ArtifactCategoryDatasets
			} else {
				category = service.ArtifactCategoryWeights
			}
		}
		return remotePath, category, fileName, nil
	}

	if target.kind == "" {
		return "", "", "", fmt.Errorf("未提供模型/数据集目标时必须提供 remote_path")
	}
	if strings.TrimSpace(req.StorageTarget) == "" {
		return "", "", "", fmt.Errorf("remote_path 为空时必须提供 storage_target")
	}
	if c.pathService == nil {
		return "", "", "", service.ErrArtifactPathServiceNil
	}

	normalizedTarget, err := c.pathService.NormalizeStorageTarget(req.StorageTarget)
	if err != nil {
		return "", "", "", err
	}
	if normalizedTarget != service.StorageTargetBaiduNetdisk {
		return "", "", "", fmt.Errorf("按记录驱动下载时必须设置 storage_target=%s", service.StorageTargetBaiduNetdisk)
	}

	switch target.kind {
	case "model":
		remotePath, err = c.modelService.ResolveFilePathByID(ctx.Request.Context(), target.id, normalizedTarget)
		if err != nil {
			return "", "", "", err
		}
		if fileName == "" {
			fileName, err = c.modelService.GetWeightNameByID(ctx.Request.Context(), target.id)
			if err != nil {
				return "", "", "", err
			}
		}
		category = service.ArtifactCategoryWeights
	case "dataset":
		remotePath, err = c.datasetService.ResolveFilePathByID(ctx.Request.Context(), target.id, normalizedTarget)
		if err != nil {
			return "", "", "", err
		}
		if fileName == "" {
			fileName, err = c.datasetService.GetFileNameByID(ctx.Request.Context(), target.id)
			if err != nil {
				return "", "", "", err
			}
		}
		category = service.ArtifactCategoryDatasets
	default:
		return "", "", "", fmt.Errorf("无效的同步目标")
	}

	return remotePath, category, fileName, nil
}

func (c *BaiduController) syncRecordStorageServer(ctx *gin.Context, target storageSyncTarget, localStorageServer string) ([]string, error) {
	switch target.kind {
	case "model":
		return c.modelService.UpdateStorageServersByID(
			ctx.Request.Context(),
			target.id,
			dao.StorageActionAdd,
			[]string{localStorageServer},
		)
	case "dataset":
		return c.datasetService.UpdateStorageServersByID(
			ctx.Request.Context(),
			target.id,
			dao.StorageActionAdd,
			[]string{localStorageServer},
		)
	default:
		return nil, nil
	}
}
