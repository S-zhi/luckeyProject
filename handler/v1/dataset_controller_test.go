package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	entity2 "lucky_project/entity"
	"lucky_project/service"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
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

	t.Run("Update Dataset Metadata", func(t *testing.T) {
		dataset := entity2.Dataset{
			Name:          fmt.Sprintf("PatchDataset_%d", time.Now().UnixNano()),
			StorageServer: "backend",
			TaskType:      "detect",
			DatasetFormat: "yolo",
			DatasetPath:   "/tmp/patch_dataset_origin.zip",
			FileName:      "patch_dataset_origin.zip",
			Version:       "v1.0.0",
			SizeMB:        88.8,
		}
		createBody, _ := json.Marshal(dataset)
		createResp := performRequest(testRouter, "POST", "/v1/datasets", bytes.NewBuffer(createBody))
		assert.Equal(t, http.StatusCreated, createResp.Code)

		var created entity2.Dataset
		err := json.Unmarshal(createResp.Body.Bytes(), &created)
		assert.NoError(t, err)
		assert.NotZero(t, created.ID)

		updateReq := map[string]interface{}{
			"name":           created.Name + "_updated",
			"task_type":      "segment",
			"dataset_format": "coco",
			"dataset_path":   "/tmp/patch_dataset_updated.zip",
			"file_name":      "patch_dataset_updated.zip",
			"description":    "updated dataset description",
			"config_path":    "data_updated.yaml",
			"version":        "v2.0.0",
			"num_classes":    3,
			"class_names": []string{
				"cat",
				"dog",
				"bird",
			},
			"train_count": 100,
			"val_count":   20,
			"test_count":  10,
			"size_mb":     222.333,
			"storage_servers": []string{
				"backend",
				"baidu_netdisk",
			},
		}
		updateBody, _ := json.Marshal(updateReq)
		updateURL := fmt.Sprintf("/v1/datasets/%d", created.ID)
		updateResp := performRequest(testRouter, "PATCH", updateURL, bytes.NewBuffer(updateBody))
		assert.Equal(t, http.StatusOK, updateResp.Code)

		var updated entity2.Dataset
		err = json.Unmarshal(updateResp.Body.Bytes(), &updated)
		assert.NoError(t, err)
		assert.Equal(t, created.ID, updated.ID)
		assert.Equal(t, updateReq["name"], updated.Name)
		assert.Equal(t, updateReq["task_type"], updated.TaskType)
		assert.Equal(t, updateReq["dataset_format"], updated.DatasetFormat)
		assert.Equal(t, updateReq["dataset_path"], updated.DatasetPath)
		assert.Equal(t, updateReq["file_name"], updated.FileName)
		assert.Equal(t, "updated dataset description", derefString(updated.Description))
		assert.Equal(t, "data_updated.yaml", derefString(updated.ConfigPath))
		assert.Equal(t, updateReq["version"], updated.Version)
		assert.Equal(t, uint(3), *updated.NumClasses)
		assert.Equal(t, uint(100), *updated.TrainCount)
		assert.Equal(t, uint(20), *updated.ValCount)
		assert.Equal(t, uint(10), *updated.TestCount)
		assert.InDelta(t, 222.333, updated.SizeMB, 0.0001)
		assert.True(t, storageServerContains(updated.StorageServer, "backend"))
		assert.True(t, storageServerContains(updated.StorageServer, "baidu_netdisk"))

		var classNames []string
		err = json.Unmarshal(updated.ClassNames, &classNames)
		assert.NoError(t, err)
		assert.Equal(t, []string{"cat", "dog", "bird"}, classNames)
	})

	t.Run("Update Dataset Metadata Reject Immutable Field", func(t *testing.T) {
		req := map[string]interface{}{
			"id": 100,
		}
		body, _ := json.Marshal(req)
		w := performRequest(testRouter, "PATCH", "/v1/datasets/1", bytes.NewBuffer(body))
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "id is immutable")
	})

	t.Run("Download Dataset File From Backend", func(t *testing.T) {
		fileName := fmt.Sprintf("download_dataset_%d.zip", time.Now().UnixNano())
		dataset := entity2.Dataset{
			Name:          fmt.Sprintf("DownloadDataset_%d", time.Now().UnixNano()),
			StorageServer: "backend",
			TaskType:      "detect",
			DatasetFormat: "yolo",
			DatasetPath:   "/tmp/" + fileName,
			FileName:      fileName,
			Version:       "v1.0.0",
			SizeMB:        77.7,
		}
		createBody, _ := json.Marshal(dataset)
		createResp := performRequest(testRouter, "POST", "/v1/datasets", bytes.NewBuffer(createBody))
		assert.Equal(t, http.StatusCreated, createResp.Code)

		var created entity2.Dataset
		err := json.Unmarshal(createResp.Body.Bytes(), &created)
		assert.NoError(t, err)
		assert.NotZero(t, created.ID)

		localPath := filepath.Join(service.DefaultBackendDatasetsRoot, fileName)
		err = os.MkdirAll(filepath.Dir(localPath), 0o755)
		assert.NoError(t, err)
		content := []byte("mock dataset archive content")
		err = os.WriteFile(localPath, content, 0o644)
		assert.NoError(t, err)

		t.Cleanup(func() {
			_ = os.Remove(localPath)
		})

		downloadURL := fmt.Sprintf("/v1/datasets/%d/download", created.ID)
		w := performRequest(testRouter, "GET", downloadURL, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, string(content), w.Body.String())
		assert.True(t, strings.Contains(w.Header().Get("Content-Disposition"), fileName))
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
			"artifact_name":  "dataset_upload",
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
		assert.Equal(t, "backend", resp["storage_target"])
		assert.Equal(t, false, resp["upload_to_baidu"])
		assert.Equal(t, false, resp["baidu_uploaded"])
		assert.Equal(t, false, resp["mysql_updated"])
		assert.Equal(t, float64(0), resp["mysql_affected"])
		fileName, _ := resp["file_name"].(string)
		assert.Equal(t, "dataset_upload.zip", fileName)
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
			"artifact_name": "dataset_default",
		})
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "backend", resp["storage_server"])
		assert.Equal(t, "backend", resp["storage_target"])
		assert.Equal(t, false, resp["upload_to_baidu"])
		assert.Equal(t, false, resp["baidu_uploaded"])
		assert.Equal(t, false, resp["mysql_updated"])
		assert.Equal(t, float64(0), resp["mysql_affected"])

		savedPath, ok := resp["saved_path"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, savedPath)
		_, err = os.Stat(savedPath)
		assert.NoError(t, err)

		t.Cleanup(func() {
			_ = os.Remove(savedPath)
		})
	})

	t.Run("Upload Dataset File Sync MySQL Size", func(t *testing.T) {
		fileName := fmt.Sprintf("sync_dataset_%d.zip", time.Now().UnixNano())
		dataset := entity2.Dataset{
			Name:          fmt.Sprintf("SyncSizeDataset_%d", time.Now().UnixNano()),
			StorageServer: "backend",
			TaskType:      "detect",
			DatasetFormat: "yolo",
			DatasetPath:   "/tmp/" + fileName,
			FileName:      fileName,
			Version:       "v1.0.0",
			SizeMB:        0,
		}
		createBody, _ := json.Marshal(dataset)
		createResp := performRequest(testRouter, "POST", "/v1/datasets", bytes.NewBuffer(createBody))
		assert.Equal(t, http.StatusCreated, createResp.Code)

		var created entity2.Dataset
		err := json.Unmarshal(createResp.Body.Bytes(), &created)
		assert.NoError(t, err)
		assert.NotZero(t, created.ID)

		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, fileName)
		content := bytes.Repeat([]byte("a"), 2*1024*1024+500)
		err = os.WriteFile(srcPath, content, 0o644)
		assert.NoError(t, err)

		w := performMultipartRequest(t, testRouter, http.MethodPost, "/v1/datasets/upload", "file", srcPath, map[string]string{})
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, fileName, resp["file_name"])
		assert.Equal(t, true, resp["mysql_updated"])

		datasetService := service.NewDatasetService()
		updated, err := datasetService.GetByID(context.Background(), created.ID)
		assert.NoError(t, err)
		expectedMB := math.Round((float64(len(content))/(1024*1024))*1000) / 1000
		assert.InDelta(t, expectedMB, updated.SizeMB, 0.0001)

		savedPath, _ := resp["saved_path"].(string)
		t.Cleanup(func() {
			if strings.TrimSpace(savedPath) != "" {
				_ = os.Remove(savedPath)
			}
		})
	})

	t.Run("Delete Dataset By FileName", func(t *testing.T) {
		fileName := fmt.Sprintf("delete_dataset_%d.zip", time.Now().UnixNano())
		dataset := entity2.Dataset{
			Name:          fmt.Sprintf("DeleteDataset_%d", time.Now().UnixNano()),
			StorageServer: "backend",
			TaskType:      "detect",
			DatasetFormat: "yolo",
			DatasetPath:   "/tmp/" + fileName,
			FileName:      fileName,
			Version:       "v1.0.0",
			SizeMB:        1.23,
		}
		createBody, _ := json.Marshal(dataset)
		createResp := performRequest(testRouter, "POST", "/v1/datasets", bytes.NewBuffer(createBody))
		assert.Equal(t, http.StatusCreated, createResp.Code)

		var created entity2.Dataset
		err := json.Unmarshal(createResp.Body.Bytes(), &created)
		assert.NoError(t, err)
		assert.NotZero(t, created.ID)

		localPath := filepath.Join(service.DefaultBackendDatasetsRoot, fileName)
		err = os.MkdirAll(filepath.Dir(localPath), 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(localPath, []byte("to-delete"), 0o644)
		assert.NoError(t, err)

		deleteURL := "/v1/datasets/by-filename?file_name=" + url.QueryEscape(fileName)
		w := performRequest(testRouter, http.MethodDelete, deleteURL, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, fileName, resp["file_name"])
		assert.True(t, resp["deleted_records"].(float64) >= 1)

		_, statErr := os.Stat(localPath)
		assert.True(t, os.IsNotExist(statErr))

		datasetService := service.NewDatasetService()
		_, err = datasetService.GetByID(context.Background(), created.ID)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	})
}
