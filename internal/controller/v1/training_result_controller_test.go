package v1_test

import (
	"bytes"
	"encoding/json"
	"lucky_project/internal/entity"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrainingResultAPI(t *testing.T) {
	// 1. 测试创建训练结果
	t.Run("Create Training Result", func(t *testing.T) {
		result := entity.ModelTrainingResult{
			ModelID:        1,
			DatasetID:      1,
			DatasetVersion: 1.0,
			TrainingStatus: 2,
			MetricDetail:   json.RawMessage(`{"mAP50": 0.95, "recall": 0.92}`),
			WeightPath:     "/tmp/best.pt",
			CometLogURL:    "https://comet.ml/user/exp1",
		}
		body, _ := json.Marshal(result)
		w := performRequest(testRouter, "POST", "/v1/training-results", bytes.NewBuffer(body))

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp entity.ModelTrainingResult
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, result.ModelID, resp.ModelID)
		assert.True(t, resp.ID > 0)
	})

	// 2. 测试分页与过滤查询
	t.Run("Filter Training Results", func(t *testing.T) {
		w := performRequest(testRouter, "GET", "/v1/training-results?training_status=2", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var result entity.PageResult
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.True(t, result.Total >= 1)
	})
}
