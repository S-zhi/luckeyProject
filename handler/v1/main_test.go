package v1_test

import (
	"io"
	"lucky_project/config"
	"lucky_project/infrastructure/db"
	"lucky_project/router"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

var testRouter *gin.Engine

func TestMain(m *testing.M) {
	// 切换到项目根目录读取配置
	os.Chdir("../..")

	// 初始化配置
	if err := config.InitConfig(); err != nil {
		panic(err)
	}

	// 初始化数据库
	if err := db.InitDB(); err != nil {
		panic(err)
	}

	// 设置 Gin 为测试模式
	gin.SetMode(gin.TestMode)
	testRouter = router.SetupRouter()

	// 运行测试
	code := m.Run()
	os.Exit(code)
}

// performRequest 执行请求的辅助函数
func performRequest(r http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
