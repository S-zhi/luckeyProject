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

func TestDatasetAPI(t *testing.T) {
	// 1. 测试创建数据集
	t.Run("Create Dataset", func(t *testing.T) {
		dataset := entity2.Dataset{
			Name:          fmt.Sprintf("TestDataset_%d", time.Now().UnixNano()),
			StorageServer: "nas-01",
			TaskType:      "detect",
			DatasetFormat: "yolo",
			DatasetPath:   "/tmp/test_dataset",
			Version:       "v1.0.0",
			SizeMB:        123.456,
		}
		body, _ := json.Marshal(dataset)
		w := performRequest(testRouter, "POST", "/v1/datasets", bytes.NewBuffer(body))

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp entity2.Dataset
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, dataset.Name, resp.Name)
		assert.True(t, resp.ID > 0)
	})

	// 2. 测试组合过滤查询
	t.Run("Filter Datasets", func(t *testing.T) {
		w := performRequest(testRouter, "GET", "/v1/datasets?task_type=detect&dataset_format=yolo", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var result entity2.PageResult
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.True(t, result.Total >= 1)
	})

	// 3. 测试上传数据集文件
	t.Run("Upload Dataset File", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "dataset.zip")
		err := os.WriteFile(filePath, []byte("mock dataset zip content"), 0o644)
		assert.NoError(t, err)

		w := performMultipartRequest(t, testRouter, http.MethodPost, "/v1/datasets/upload", "file", filePath, map[string]string{
			"subdir":         "ut",
			"storage_server": "nas-02",
		})
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)

		savedPath, ok := resp["saved_path"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, savedPath)
		assert.Equal(t, "nas-02", resp["storage_server"])
		assert.Equal(t, false, resp["upload_to_baidu"])
		assert.Equal(t, false, resp["baidu_uploaded"])
		_, err = os.Stat(savedPath)
		assert.NoError(t, err)

		t.Cleanup(func() {
			_ = os.Remove(savedPath)
		})
	})

	t.Run("Upload Dataset File Default Storage Server", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "dataset_default.zip")
		err := os.WriteFile(filePath, []byte("mock dataset zip content"), 0o644)
		assert.NoError(t, err)

		w := performMultipartRequest(t, testRouter, http.MethodPost, "/v1/datasets/upload", "file", filePath, map[string]string{
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
}
