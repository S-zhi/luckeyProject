package v1_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	entity2 "lucky_project/entity"
	"net/http"
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
		assert.Equal(t, model.Name, resp.Name)
		assert.True(t, resp.ID > 0)
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
}
