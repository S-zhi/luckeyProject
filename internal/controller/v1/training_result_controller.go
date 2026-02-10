package v1

import (
	"lucky_project/internal/entity"
	"lucky_project/internal/service"
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
	var result entity.ModelTrainingResult
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
	var params entity.QueryParams
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

// UploadWeight handles POST /v1/training-results/:id/upload
func (c *TrainingResultController) UploadWeight(ctx *gin.Context) {
	id, err := parseIDParam(ctx, "id")
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	var req struct {
		RemotePath string `json:"remote_path" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.trainingService.UploadWeight(ctx.Request.Context(), id, req.RemotePath); err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "upload success"})
}
