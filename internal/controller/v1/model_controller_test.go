package v1_test

import (
	"bytes"
	"encoding/json"
	"lucky_project/internal/entity"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelAPI(t *testing.T) {
	// 1. 测试创建模型
	t.Run("Create Model", func(t *testing.T) {
		model := entity.Model{
			ModelName:    "TestModel_v1",
			ModelType:    1,
			ModelVersion: 1.0,
			Algorithm:    "YOLOv8",
			Framework:    "PyTorch",
			WeightPath:   "/tmp/test_weight.pt",
		}
		body, _ := json.Marshal(model)
		w := performRequest(testRouter, "POST", "/v1/models", bytes.NewBuffer(body))

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp entity.Model
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, model.ModelName, resp.ModelName)
		assert.True(t, resp.ID > 0)
	})

	// 2. 测试分页查询
	t.Run("List Models", func(t *testing.T) {
		w := performRequest(testRouter, "GET", "/v1/models?page=1&page_size=10", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var result entity.PageResult
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.True(t, result.Total >= 1)
	})

	// 3. 测试组合过滤
	t.Run("Filter Models", func(t *testing.T) {
		w := performRequest(testRouter, "GET", "/v1/models?algorithm=YOLOv8&framework=PyTorch", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var result entity.PageResult
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.True(t, result.Total >= 1)
	})

	// 4. 测试排序
	t.Run("Sort Models", func(t *testing.T) {
		w := performRequest(testRouter, "GET", "/v1/models?weight_sort=desc", nil)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
