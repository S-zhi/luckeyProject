package router

import (
	v2 "lucky_project/handler/v1"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	modelController := v2.NewModelController()
	datasetController := v2.NewDatasetController()
	trainingController := v2.NewTrainingResultController()

	r := gin.New()
	r.MaxMultipartMemory = 256 << 20 // 256MB
	r.Use(gin.Logger(), gin.Recovery())
	_ = r.SetTrustedProxies(nil)
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// 允许本地开发任意端口来源，例如 Vite(5173)、Live Server(5501) 等
			return strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "http://127.0.0.1:") ||
				strings.HasPrefix(origin, "https://localhost:") ||
				strings.HasPrefix(origin, "https://127.0.0.1:")
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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
			models.POST("/upload", modelController.UploadModelFile)
		}

		// Dataset routes
		datasets := v1Group.Group("/datasets")
		{
			datasets.POST("", datasetController.CreateDataset)
			datasets.GET("", datasetController.GetAllDatasets)
			datasets.POST("/upload", datasetController.UploadDatasetFile)
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
