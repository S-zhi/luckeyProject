package v1

import (
	entity2 "lucky_project/entity"
	"lucky_project/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TrainingResultController struct {
	trainingService *service.TrainingResultService
}

func NewTrainingResultController() *TrainingResultController {
	return &TrainingResultController{
		trainingService: service.NewTrainingResultService(),
	}
}

// CreateTrainingResult handles POST /v1/training-results
func (c *TrainingResultController) CreateTrainingResult(ctx *gin.Context) {
	var result entity2.ModelTrainingResult
	if err := ctx.ShouldBindJSON(&result); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.trainingService.CreateTrainingResult(ctx.Request.Context(), &result); err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, result)
}

// GetAllResults handles GET /v1/training-results
func (c *TrainingResultController) GetAllResults(ctx *gin.Context) {
	var params entity2.QueryParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.trainingService.GetAllResults(ctx.Request.Context(), params)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, result)
}
