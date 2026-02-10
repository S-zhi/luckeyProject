package v1

import (
	"lucky_project/internal/entity"
	"lucky_project/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DatasetController struct {
	datasetService *service.DatasetService
}

func NewDatasetController() *DatasetController {
	return &DatasetController{
		datasetService: service.NewDatasetService(),
	}
}

// CreateDataset handles POST /v1/datasets
func (c *DatasetController) CreateDataset(ctx *gin.Context) {
	var dataset entity.Dataset
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
	var params entity.QueryParams
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

// UploadDataset handles POST /v1/datasets/:id/upload
func (c *DatasetController) UploadDataset(ctx *gin.Context) {
	id, err := parseIDParam(ctx, "id")
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	if err := c.datasetService.UploadDataset(ctx.Request.Context(), id); err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "upload success"})
}

// ListRemoteFiles handles GET /v1/datasets/remote-files
func (c *DatasetController) ListRemoteFiles(ctx *gin.Context) {
	dir := ctx.DefaultQuery("dir", "/")
	result, err := c.datasetService.ListRemoteFiles(dir)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, result)
}
