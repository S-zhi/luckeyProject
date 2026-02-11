package v1

import (
	"errors"
	entity2 "lucky_project/entity"
	"lucky_project/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ModelController struct {
	modelService  *service.ModelService
	uploadService *service.UploadService
}

func NewModelController() *ModelController {
	return &ModelController{
		modelService:  service.NewModelService(),
		uploadService: service.NewUploadService(),
	}
}

// CreateModel handles POST /v1/models
func (c *ModelController) CreateModel(ctx *gin.Context) {
	var model entity2.Model
	if err := ctx.ShouldBindJSON(&model); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.modelService.CreateModel(ctx.Request.Context(), &model); err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, model)
}

// GetAllModels handles GET /v1/models
func (c *ModelController) GetAllModels(ctx *gin.Context) {
	var params entity2.QueryParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.modelService.GetAllModels(ctx.Request.Context(), params)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// UploadModelFile handles POST /v1/models/upload
func (c *ModelController) UploadModelFile(ctx *gin.Context) {
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

	result, err := c.uploadService.SaveModelFile(file, subdir, storageServer, uploadToBaidu)
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
