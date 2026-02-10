package router

import (
	v1 "lucky_project/internal/controller/v1"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	modelController := v1.NewModelController()
	datasetController := v1.NewDatasetController()
	trainingController := v1.NewTrainingResultController()

	r := gin.Default()
	r.Use(gin.Recovery())

	v1Group := r.Group("/v1")
	{
		// Model routes
		models := v1Group.Group("/models")
		{
			models.POST("", modelController.CreateModel)
			models.GET("", modelController.GetAllModels)
			models.POST("/:id/upload", modelController.UploadModel)
			models.GET("/remote-files", modelController.ListRemoteFiles)
		}

		// Dataset routes
		datasets := v1Group.Group("/datasets")
		{
			datasets.POST("", datasetController.CreateDataset)
			datasets.GET("", datasetController.GetAllDatasets)
			datasets.POST("/:id/upload", datasetController.UploadDataset)
			datasets.GET("/remote-files", datasetController.ListRemoteFiles)
		}

		// Training Result routes
		trainings := v1Group.Group("/training-results")
		{
			trainings.POST("", trainingController.CreateTrainingResult)
			trainings.GET("", trainingController.GetAllResults)
			trainings.POST("/:id/upload", trainingController.UploadWeight)
		}
	}

	return r
}
