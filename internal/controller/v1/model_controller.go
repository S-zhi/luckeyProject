package v1

import (
	"lucky_project/internal/entity"
	"lucky_project/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ModelController struct {
	modelService *service.ModelService
}

func NewModelController() *ModelController {
	return &ModelController{
		modelService: service.NewModelService(),
	}
}

// CreateModel handles POST /v1/models
func (c *ModelController) CreateModel(ctx *gin.Context) {
	var model entity.Model
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
	var params entity.QueryParams
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

// UploadModel handles POST /v1/models/:id/upload
func (c *ModelController) UploadModel(ctx *gin.Context) {
	id, err := parseIDParam(ctx, "id")
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	if err := c.modelService.UploadModel(ctx.Request.Context(), id); err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "upload success"})
}

// ListRemoteFiles handles GET /v1/models/remote-files
func (c *ModelController) ListRemoteFiles(ctx *gin.Context) {
	dir := ctx.DefaultQuery("dir", "/")
	result, err := c.modelService.ListRemoteFiles(dir)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, result)
}
