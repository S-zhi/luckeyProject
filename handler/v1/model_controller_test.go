package v1_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	entity2 "lucky_project/entity"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestModelAPI(t *testing.T) {
	// 1. 测试创建模型
	t.Run("Create Model", func(t *testing.T) {
		algorithmID := "yolo_ultralytics"
		model := entity2.Model{
			Name:          fmt.Sprintf("TestModel_%d", time.Now().UnixNano()),
			Version:       1.00,
			BaseModelID:   0,
			AlgorithmID:   &algorithmID,
			WeightName:    "test_weight.pt",
			StorageServer: "nas-01",
			WeightSizeMB:  95.5,
			TaskType:      "detect",
		}
		body, _ := json.Marshal(model)
		w := performRequest(testRouter, "POST", "/v1/models", bytes.NewBuffer(body))

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp entity2.Model
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, model.Name, resp.Name)
		assert.True(t, resp.ID > 0)
	})

	t.Run("Create Model Duplicate Name Should Upsert", func(t *testing.T) {
		algorithmID := "yolo_ultralytics"
		model1 := entity2.Model{
			Name:          fmt.Sprintf("DupModel_%d", time.Now().UnixNano()),
			Version:       1.00,
			BaseModelID:   0,
			AlgorithmID:   &algorithmID,
			WeightName:    "test_weight_dup_1.pt",
			StorageServer: "nas-01",
			WeightSizeMB:  95.5,
			TaskType:      "detect",
		}
		model2 := model1
		model2.StorageServer = "nas-02"
		model2.WeightName = "test_weight_dup_2.pt"
		model2.WeightSizeMB = 96.7

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
		assert.True(t, storageServerContains(resp2.StorageServer, model2.StorageServer))
		assert.Equal(t, model2.WeightName, resp2.WeightName)
	})

	t.Run("Update Model Metadata", func(t *testing.T) {
		algorithmID := "algo_before_update"
		model := entity2.Model{
			Name:          fmt.Sprintf("PatchModel_%d", time.Now().UnixNano()),
			Version:       1.00,
			BaseModelID:   0,
			AlgorithmID:   &algorithmID,
			WeightName:    "patch_model_origin.pt",
			StorageServer: "backend",
			WeightSizeMB:  88.8,
			TaskType:      "detect",
		}
		createBody, _ := json.Marshal(model)
		createResp := performRequest(testRouter, "POST", "/v1/models", bytes.NewBuffer(createBody))
		assert.Equal(t, http.StatusCreated, createResp.Code)

		var created entity2.Model
		err := json.Unmarshal(createResp.Body.Bytes(), &created)
		assert.NoError(t, err)
		assert.NotZero(t, created.ID)

		updateReq := map[string]interface{}{
			"name":           created.Name + "_updated",
			"version":        1.10,
			"base_model_id":  0,
			"algorithm_id":   "algo_after_update",
			"task_type":      "detect",
			"description":    "updated description",
			"framework":      "pytorch",
			"weight_size_mb": 123.456,
			"paper":          "https://example.com/paper",
			"params_url":     "https://example.com/params",
			"storage_servers": []string{
				"backend",
				"baidu_netdisk",
			},
			"weight_name": "patch_model_updated.pt",
		}
		updateBody, _ := json.Marshal(updateReq)
		updateURL := fmt.Sprintf("/v1/models/%d", created.ID)
		updateResp := performRequest(testRouter, "PATCH", updateURL, bytes.NewBuffer(updateBody))
		assert.Equal(t, http.StatusOK, updateResp.Code)

		var updated entity2.Model
		err = json.Unmarshal(updateResp.Body.Bytes(), &updated)
		assert.NoError(t, err)
		assert.Equal(t, created.ID, updated.ID)
		assert.Equal(t, updateReq["name"], updated.Name)
		assert.InDelta(t, 1.10, updated.Version, 0.0001)
		assert.Equal(t, "algo_after_update", derefString(updated.AlgorithmID))
		assert.Equal(t, "updated description", derefString(updated.Description))
		assert.Equal(t, "pytorch", derefString(updated.Framework))
		assert.InDelta(t, 123.456, updated.WeightSizeMB, 0.0001)
		assert.Equal(t, "https://example.com/paper", derefString(updated.Paper))
		assert.Equal(t, "https://example.com/params", derefString(updated.ParamsURL))
		assert.Equal(t, "patch_model_updated.pt", updated.WeightName)
		assert.True(t, storageServerContains(updated.StorageServer, "backend"))
		assert.True(t, storageServerContains(updated.StorageServer, "baidu_netdisk"))
	})

	t.Run("Update Model Metadata Reject Immutable Field", func(t *testing.T) {
		req := map[string]interface{}{
			"id": 100,
		}
		body, _ := json.Marshal(req)
		w := performRequest(testRouter, "PATCH", "/v1/models/1", bytes.NewBuffer(body))
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "id is immutable")
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
		w := performRequest(testRouter, "GET", "/v1/models?algorithm_id=yolo_ultralytics&task_type=detect", nil)

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
			"artifact_name":  "test_model",
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
		assert.Equal(t, "backend", resp["storage_target"])
		assert.Equal(t, false, resp["upload_to_baidu"])
		assert.Equal(t, false, resp["baidu_uploaded"])
		fileName, _ := resp["file_name"].(string)
		assert.True(t, regexp.MustCompile(`^test_model_[a-f0-9]{12}\.pt$`).MatchString(fileName), fileName)
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
			"artifact_name": "default_model",
		})
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "backend", resp["storage_server"])
		assert.Equal(t, "backend", resp["storage_target"])
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

		w1 := performMultipartRequest(t, testRouter, http.MethodPost, "/v1/models/upload", "file", filePath, map[string]string{
			"artifact_name": "duplicate_model",
		})
		assert.Equal(t, http.StatusCreated, w1.Code)

		w2 := performMultipartRequest(t, testRouter, http.MethodPost, "/v1/models/upload", "file", filePath, map[string]string{
			"artifact_name": "duplicate_model",
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
		fileName1, _ := resp1["file_name"].(string)
		fileName2, _ := resp2["file_name"].(string)
		assert.True(t, regexp.MustCompile(`^duplicate_model_[a-f0-9]{12}\.pt$`).MatchString(fileName1), fileName1)
		assert.True(t, regexp.MustCompile(`^duplicate_model_[a-f0-9]{12}\.pt$`).MatchString(fileName2), fileName2)
		assert.NotEqual(t, fileName1, fileName2)

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
			"artifact_name":   "invalid_bool",
			"upload_to_baidu": "not-bool",
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "upload_to_baidu must be a boolean")
	})
}

func storageServerContains(value, expected string) bool {
	if value == expected {
		return true
	}

	var list []string
	if err := json.Unmarshal([]byte(value), &list); err != nil {
		return false
	}

	for _, item := range list {
		if item == expected {
			return true
		}
	}
	return false
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
