package v1_test

import (
	"bytes"
	"encoding/json"
	"lucky_project/internal/entity"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatasetAPI(t *testing.T) {
	// 1. 测试创建数据集
	t.Run("Create Dataset", func(t *testing.T) {
		dataset := entity.Dataset{
			DatasetName:    "TestDataset_v1",
			DatasetType:    1,
			DatasetVersion: 1.0,
			SampleCount:    100,
			StorageType:    1,
			DatasetPath:    "/tmp/test_dataset",
			AnnotationType: 1,
		}
		body, _ := json.Marshal(dataset)
		w := performRequest(testRouter, "POST", "/v1/datasets", bytes.NewBuffer(body))

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp entity.Dataset
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, dataset.DatasetName, resp.DatasetName)
		assert.True(t, resp.ID > 0)
	})

	// 2. 测试组合过滤查询
	t.Run("Filter Datasets", func(t *testing.T) {
		w := performRequest(testRouter, "GET", "/v1/datasets?dataset_type=1&storage_type=1", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var result entity.PageResult
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.True(t, result.Total >= 1)
	})
}
