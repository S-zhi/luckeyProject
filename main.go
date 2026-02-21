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
		log.Fatalf("初始化配置失败：%v", err)
	}

	// 2. Initialize database
	if err := config.InitDB(); err != nil {
		log.Fatalf("初始化数据库失败：%v", err)
	}

	// 3. Initialize redis
	if err := config.InitRedis(); err != nil {
		log.Fatalf("初始化Redis失败：%v", err)
	}

	// 4. Setup router
	r := router.SetupRouter()

	// 5. Start server
	port := config.AppConfig.Server.Port
	if port == 0 {
		port = 8080
	}

	fmt.Printf("服务已启动，监听端口 %d...\n", port)
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatalf("服务运行失败：%v", err)
	}
}
