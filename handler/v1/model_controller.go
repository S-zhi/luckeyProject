package v1

import (
	entity2 "lucky_project/entity"
	"lucky_project/service"
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
