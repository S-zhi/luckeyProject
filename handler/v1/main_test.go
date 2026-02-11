package v1_test

import (
	"bytes"
	"io"
	"lucky_project/config"
	"lucky_project/router"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	if err := config.InitDB(); err != nil {
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

func performMultipartRequest(t *testing.T, r http.Handler, method, path, fileField, filePath string, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("open upload file failed: %v", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile(fileField, filepath.Base(filePath))
	if err != nil {
		t.Fatalf("create multipart file failed: %v", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		t.Fatalf("copy multipart file failed: %v", err)
	}

	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write multipart field failed: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	req, _ := http.NewRequest(method, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
