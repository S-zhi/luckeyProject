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
}
