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
	// 初始化依赖
	config.InitDeploy()
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
