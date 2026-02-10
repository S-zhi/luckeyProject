package router

import (
	v2 "lucky_project/handler/v1"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	modelController := v2.NewModelController()
	datasetController := v2.NewDatasetController()
	trainingController := v2.NewTrainingResultController()

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	_ = r.SetTrustedProxies(nil)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	})

	v1Group := r.Group("/v1")
	{
		// Model routes
		models := v1Group.Group("/models")
		{
			models.POST("", modelController.CreateModel)
			models.GET("", modelController.GetAllModels)
		}

		// Dataset routes
		datasets := v1Group.Group("/datasets")
		{
			datasets.POST("", datasetController.CreateDataset)
			datasets.GET("", datasetController.GetAllDatasets)
		}

		// Training Result routes
		trainings := v1Group.Group("/training-results")
		{
			trainings.POST("", trainingController.CreateTrainingResult)
			trainings.GET("", trainingController.GetAllResults)
		}
	}

	return r
}
