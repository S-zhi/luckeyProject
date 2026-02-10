package v1

import (
	entity2 "lucky_project/entity"
	"lucky_project/service"
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
