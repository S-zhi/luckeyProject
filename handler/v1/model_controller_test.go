package v1_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	entity2 "lucky_project/entity"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestModelAPI(t *testing.T) {
	// 1. 测试创建模型
	t.Run("Create Model", func(t *testing.T) {
		model := entity2.Model{
			Name:          fmt.Sprintf("TestModel_%d", time.Now().UnixNano()),
			StorageServer: "nas-01",
			ModelPath:     "/tmp/test_weight.pt",
			ImplType:      "yolo_ultralytics",
			DatasetID:     1,
			SizeMB:        95.5,
			Version:       "v1.0.0",
			TaskType:      "detect",
		}
		body, _ := json.Marshal(model)
		w := performRequest(testRouter, "POST", "/v1/models", bytes.NewBuffer(body))

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp entity2.Model
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, fmt.Sprintf("%s_%s", model.Name, model.Version), resp.Name)
		assert.True(t, resp.ID > 0)
	})

	t.Run("Create Model Duplicate Name Should Upsert", func(t *testing.T) {
		model1 := entity2.Model{
			Name:          fmt.Sprintf("DupModel_%d", time.Now().UnixNano()),
			StorageServer: "nas-01",
			ModelPath:     "/tmp/test_weight_dup_1.pt",
			ImplType:      "yolo_ultralytics",
			DatasetID:     1,
			SizeMB:        95.5,
			Version:       "v1.0.0",
			TaskType:      "detect",
		}
		model2 := model1
		model2.StorageServer = "nas-02"
		model2.ModelPath = "/tmp/test_weight_dup_2.pt"
		model2.SizeMB = 96.7

		body1, _ := json.Marshal(model1)
		body2, _ := json.Marshal(model2)

		w1 := performRequest(testRouter, "POST", "/v1/models", bytes.NewBuffer(body1))
		assert.Equal(t, http.StatusCreated, w1.Code)

		var resp1 entity2.Model
		err := json.Unmarshal(w1.Body.Bytes(), &resp1)
		assert.NoError(t, err)
		assert.NotZero(t, resp1.ID)

		w2 := performRequest(testRouter, "POST", "/v1/models", bytes.NewBuffer(body2))
		assert.Equal(t, http.StatusCreated, w2.Code)

		var resp2 entity2.Model
		err = json.Unmarshal(w2.Body.Bytes(), &resp2)
		assert.NoError(t, err)
		assert.Equal(t, resp1.ID, resp2.ID)
		assert.Equal(t, model2.StorageServer, resp2.StorageServer)
		assert.Equal(t, model2.ModelPath, resp2.ModelPath)
	})

	// 2. 测试分页查询
	t.Run("List Models", func(t *testing.T) {
		w := performRequest(testRouter, "GET", "/v1/models?page=1&page_size=10", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var result entity2.PageResult
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.True(t, result.Total >= 1)
	})

	// 3. 测试组合过滤
	t.Run("Filter Models", func(t *testing.T) {
		w := performRequest(testRouter, "GET", "/v1/models?impl_type=yolo_ultralytics&task_type=detect", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var result entity2.PageResult
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.True(t, result.Total >= 1)
	})

	// 4. 测试排序
	t.Run("Sort Models", func(t *testing.T) {
		w := performRequest(testRouter, "GET", "/v1/models?size_sort=desc", nil)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 5. 测试上传模型文件
	t.Run("Upload Model File", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test_model.pt")
		err := os.WriteFile(filePath, []byte("mock model content"), 0o644)
		assert.NoError(t, err)

		w := performMultipartRequest(t, testRouter, http.MethodPost, "/v1/models/upload", "file", filePath, map[string]string{
			"subdir":         "ut",
			"storage_server": "nas-01",
		})
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)

		savedPath, ok := resp["saved_path"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, savedPath)
		assert.Equal(t, "nas-01", resp["storage_server"])
		assert.Equal(t, false, resp["upload_to_baidu"])
		assert.Equal(t, false, resp["baidu_uploaded"])
		_, err = os.Stat(savedPath)
		assert.NoError(t, err)

		t.Cleanup(func() {
			_ = os.Remove(savedPath)
		})
	})

	t.Run("Upload Model File Default Storage Server", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test_model_default.pt")
		err := os.WriteFile(filePath, []byte("mock model content"), 0o644)
		assert.NoError(t, err)

		w := performMultipartRequest(t, testRouter, http.MethodPost, "/v1/models/upload", "file", filePath, map[string]string{
			"subdir": "ut/default",
		})
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "backend", resp["storage_server"])
		assert.Equal(t, false, resp["upload_to_baidu"])
		assert.Equal(t, false, resp["baidu_uploaded"])

		savedPath, ok := resp["saved_path"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, savedPath)
		_, err = os.Stat(savedPath)
		assert.NoError(t, err)

		t.Cleanup(func() {
			_ = os.Remove(savedPath)
		})
	})

	t.Run("Upload Model File Duplicate Name", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "duplicate_model.pt")
		err := os.WriteFile(filePath, []byte("mock model content"), 0o644)
		assert.NoError(t, err)

		subdir := fmt.Sprintf("ut/duplicate_%d", time.Now().UnixNano())
		w1 := performMultipartRequest(t, testRouter, http.MethodPost, "/v1/models/upload", "file", filePath, map[string]string{
			"subdir": subdir,
		})
		assert.Equal(t, http.StatusCreated, w1.Code)

		w2 := performMultipartRequest(t, testRouter, http.MethodPost, "/v1/models/upload", "file", filePath, map[string]string{
			"subdir": subdir,
		})
		assert.Equal(t, http.StatusCreated, w2.Code)

		var resp1 map[string]interface{}
		var resp2 map[string]interface{}
		err = json.Unmarshal(w1.Body.Bytes(), &resp1)
		assert.NoError(t, err)
		err = json.Unmarshal(w2.Body.Bytes(), &resp2)
		assert.NoError(t, err)

		savedPath1, ok1 := resp1["saved_path"].(string)
		savedPath2, ok2 := resp2["saved_path"].(string)
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, "duplicate_model.pt", filepath.Base(savedPath1))
		assert.Equal(t, "duplicate_model_1.pt", filepath.Base(savedPath2))

		t.Cleanup(func() {
			_ = os.Remove(savedPath1)
			_ = os.Remove(savedPath2)
		})
	})

	t.Run("Upload Model File Invalid upload_to_baidu", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test_model_invalid_bool.pt")
		err := os.WriteFile(filePath, []byte("mock model content"), 0o644)
		assert.NoError(t, err)

		w := performMultipartRequest(t, testRouter, http.MethodPost, "/v1/models/upload", "file", filePath, map[string]string{
			"subdir":          "ut/invalid-bool",
			"upload_to_baidu": "not-bool",
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "upload_to_baidu must be a boolean")
	})
}
