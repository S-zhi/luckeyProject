package main

import (
	"fmt"
	"log"
	"lucky_project/config"
	"lucky_project/router"

	"github.com/gin-gonic/gin"
)

func main() {
	// 默认使用 release，避免线上以 debug 模式启动
	if gin.Mode() == gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	// 1. Initialize configuration
	if err := config.InitConfig(); err != nil {
		log.Fatalf("Init config failed: %v", err)
	}

	// 2. Initialize database
	if err := config.InitDB(); err != nil {
		log.Fatalf("Init database failed: %v", err)
	}

	// 3. Setup router
	r := router.SetupRouter()

	// 4. Start server
	port := config.AppConfig.Server.Port
	if port == 0 {
		port = 8080
	}

	fmt.Printf("Server is running on port %d...\n", port)
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatalf("Server run failed: %v", err)
	}
}
