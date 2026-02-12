package service

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeBaiduUploader struct {
	localPath string
	remoteDir string
}

func (f *fakeBaiduUploader) Upload(localPath, remoteDir string) (string, error) {
	f.localPath = localPath
	f.remoteDir = remoteDir
	return filepath.ToSlash(filepath.Join(remoteDir, filepath.Base(localPath))), nil
}

func TestUploadServiceSaveModelFileBaiduFlow(t *testing.T) {
	tmpDir := t.TempDir()
	pathService := &ArtifactPathService{
		BackendWeightsRoot:  filepath.Join(tmpDir, "backend", "weights"),
		BackendDatasetsRoot: filepath.Join(tmpDir, "backend", "datasets"),
		BaiduWeightsRoot:    "/project/luckyProject/weights",
		BaiduDatasetsRoot:   "/project/luckyProject/datasets",
		OtherWeightsRoot:    filepath.Join(tmpDir, "other", "weights"),
		OtherDatasetsRoot:   filepath.Join(tmpDir, "other", "datasets"),
	}
	uploader := &fakeBaiduUploader{}
	svc := &UploadService{
		PathService:   pathService,
		BaiduUploader: uploader,
	}

	srcFilePath := filepath.Join(tmpDir, "yolov7_HRW_4.2k.pt")
	err := os.WriteFile(srcFilePath, []byte("mock model content"), 0o644)
	assert.NoError(t, err)

	fileHeader := mustBuildFileHeader(t, "file", srcFilePath)
	result, err := svc.SaveModelFile(fileHeader, "yolov7_HRW_4.2k", "", "baidu", false)
	assert.NoError(t, err)

	assert.True(t, result.UploadToBaidu)
	assert.True(t, result.BaiduUploaded)
	assert.Equal(t, StorageTargetBaiduNetdisk, result.StorageTarget)
	assert.Equal(t, pathService.BaiduWeightsRoot, uploader.remoteDir)

	assert.Equal(t, "yolov7_HRW_4.2k.pt", result.FileName)
	assert.Equal(t, filepath.ToSlash(filepath.Join(pathService.BaiduWeightsRoot, result.FileName)), result.BaiduPath)
	assert.Equal(t, filepath.ToSlash(filepath.Join(pathService.BackendWeightsRoot, result.FileName)), result.ResolvedPath)
	assert.Equal(t, result.ResolvedPath, filepath.ToSlash(uploader.localPath))

	_, err = os.Stat(result.ResolvedPath)
	assert.NoError(t, err)
}

func TestUploadServiceSaveDatasetFileBackendFlow(t *testing.T) {
	tmpDir := t.TempDir()
	pathService := &ArtifactPathService{
		BackendWeightsRoot:  filepath.Join(tmpDir, "backend", "weights"),
		BackendDatasetsRoot: filepath.Join(tmpDir, "backend", "datasets"),
		BaiduWeightsRoot:    "/project/luckyProject/weights",
		BaiduDatasetsRoot:   "/project/luckyProject/datasets",
		OtherWeightsRoot:    filepath.Join(tmpDir, "other", "weights"),
		OtherDatasetsRoot:   filepath.Join(tmpDir, "other", "datasets"),
	}
	svc := &UploadService{
		PathService: pathService,
	}

	srcFilePath := filepath.Join(tmpDir, "demo_dataset.zip")
	err := os.WriteFile(srcFilePath, []byte("mock dataset content"), 0o644)
	assert.NoError(t, err)

	fileHeader := mustBuildFileHeader(t, "file", srcFilePath)
	result, err := svc.SaveDatasetFile(fileHeader, "my_dataset", StorageTargetBackend, "nas-01", false)
	assert.NoError(t, err)

	assert.False(t, result.UploadToBaidu)
	assert.False(t, result.BaiduUploaded)
	assert.Equal(t, StorageTargetBackend, result.StorageTarget)
	assert.Equal(t, filepath.ToSlash(filepath.Join(pathService.BackendDatasetsRoot, result.FileName)), result.ResolvedPath)
	assert.Equal(t, filepath.ToSlash(filepath.Join(pathService.BaiduDatasetsRoot, result.FileName)), result.Paths.BaiduPath)

	_, err = os.Stat(result.ResolvedPath)
	assert.NoError(t, err)
}

func TestUploadServiceSaveModelFileLegacyUploadToBaiduFlag(t *testing.T) {
	tmpDir := t.TempDir()
	pathService := &ArtifactPathService{
		BackendWeightsRoot:  filepath.Join(tmpDir, "backend", "weights"),
		BackendDatasetsRoot: filepath.Join(tmpDir, "backend", "datasets"),
		BaiduWeightsRoot:    "/project/luckyProject/weights",
		BaiduDatasetsRoot:   "/project/luckyProject/datasets",
		OtherWeightsRoot:    filepath.Join(tmpDir, "other", "weights"),
		OtherDatasetsRoot:   filepath.Join(tmpDir, "other", "datasets"),
	}
	uploader := &fakeBaiduUploader{}
	svc := &UploadService{
		PathService:   pathService,
		BaiduUploader: uploader,
	}

	srcFilePath := filepath.Join(tmpDir, "legacy.pt")
	err := os.WriteFile(srcFilePath, []byte("legacy"), 0o644)
	assert.NoError(t, err)

	fileHeader := mustBuildFileHeader(t, "file", srcFilePath)
	result, err := svc.SaveModelFile(fileHeader, "legacy", "", "backend", true)
	assert.NoError(t, err)
	assert.Equal(t, StorageTargetBaiduNetdisk, result.StorageTarget)
	assert.True(t, result.UploadToBaidu)
	assert.True(t, result.BaiduUploaded)
	assert.Equal(t, pathService.BaiduWeightsRoot, uploader.remoteDir)
}

func mustBuildFileHeader(t *testing.T, fieldName, filePath string) *multipart.FileHeader {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	assert.NoError(t, err)

	src, err := os.Open(filePath)
	assert.NoError(t, err)
	defer src.Close()

	_, err = io.Copy(part, src)
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	err = req.ParseMultipartForm(32 << 20)
	assert.NoError(t, err)

	files := req.MultipartForm.File[fieldName]
	if len(files) == 0 {
		t.Fatalf("multipart form field %s is empty", fieldName)
	}
	return files[0]
}
