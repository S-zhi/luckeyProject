package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeBaiduDownloader struct {
	remotePath string
	localPath  string
	content    []byte
	err        error
}

func (f *fakeBaiduDownloader) Download(remotePath, localPath string) error {
	f.remotePath = remotePath
	f.localPath = localPath
	if f.err != nil {
		return f.err
	}
	return os.WriteFile(localPath, f.content, 0o644)
}

func TestBaiduDownloadServiceDownloadToLocalWeights(t *testing.T) {
	tmpDir := t.TempDir()
	downloader := &fakeBaiduDownloader{content: []byte("mock")}
	pathService := &ArtifactPathService{
		BackendWeightsRoot:  filepath.Join(tmpDir, "weights"),
		BackendDatasetsRoot: filepath.Join(tmpDir, "datasets"),
		BaiduWeightsRoot:    "/project/luckyProject/weights",
		BaiduDatasetsRoot:   "/project/luckyProject/datasets",
		OtherWeightsRoot:    filepath.Join(tmpDir, "other", "weights"),
		OtherDatasetsRoot:   filepath.Join(tmpDir, "other", "datasets"),
	}
	svc := &BaiduDownloadService{
		PathService: pathService,
		Downloader:  downloader,
	}

	result, err := svc.DownloadToLocal("/project/luckyProject/weights/yolo.pt", "", "")
	assert.NoError(t, err)
	assert.Equal(t, "/project/luckyProject/weights/yolo.pt", result.RemotePath)
	assert.Equal(t, DownloadCategoryWeights, result.Category)
	assert.Equal(t, filepath.ToSlash(filepath.Join(tmpDir, "weights", "yolo.pt")), result.LocalPath)
	assert.Equal(t, "yolo.pt", result.FileName)
	assert.EqualValues(t, 4, result.Size)
}

func TestBaiduDownloadServiceDownloadToLocalDatasetsWithCustomName(t *testing.T) {
	tmpDir := t.TempDir()
	downloader := &fakeBaiduDownloader{content: []byte("dataset")}
	pathService := &ArtifactPathService{
		BackendWeightsRoot:  filepath.Join(tmpDir, "weights"),
		BackendDatasetsRoot: filepath.Join(tmpDir, "datasets"),
		BaiduWeightsRoot:    "/project/luckyProject/weights",
		BaiduDatasetsRoot:   "/project/luckyProject/datasets",
		OtherWeightsRoot:    filepath.Join(tmpDir, "other", "weights"),
		OtherDatasetsRoot:   filepath.Join(tmpDir, "other", "datasets"),
	}
	svc := &BaiduDownloadService{
		PathService: pathService,
		Downloader:  downloader,
	}

	result, err := svc.DownloadToLocal("/project/luckyProject/datasets/raw.zip", "dataset", "my_ds.zip")
	assert.NoError(t, err)
	assert.Equal(t, DownloadCategoryDatasets, result.Category)
	assert.Equal(t, "my_ds.zip", result.FileName)
	assert.Equal(t, filepath.ToSlash(filepath.Join(tmpDir, "datasets", "my_ds.zip")), result.LocalPath)
}

func TestBaiduDownloadServiceDownloadToLocalInvalidCategory(t *testing.T) {
	svc := &BaiduDownloadService{
		PathService: NewArtifactPathService(),
		Downloader:  &fakeBaiduDownloader{content: []byte("x")},
	}

	_, err := svc.DownloadToLocal("/project/luckyProject/weights/a.pt", "unknown", "")
	assert.True(t, errors.Is(err, ErrInvalidDownloadCategory))
}

func TestBaiduDownloadServiceDownloadToLocalDownloaderError(t *testing.T) {
	tmpDir := t.TempDir()
	svc := &BaiduDownloadService{
		PathService: &ArtifactPathService{
			BackendWeightsRoot:  filepath.Join(tmpDir, "weights"),
			BackendDatasetsRoot: filepath.Join(tmpDir, "datasets"),
			BaiduWeightsRoot:    "/project/luckyProject/weights",
			BaiduDatasetsRoot:   "/project/luckyProject/datasets",
			OtherWeightsRoot:    filepath.Join(tmpDir, "other", "weights"),
			OtherDatasetsRoot:   filepath.Join(tmpDir, "other", "datasets"),
		},
		Downloader: &fakeBaiduDownloader{err: errors.New("download failed")},
	}

	_, err := svc.DownloadToLocal("/project/luckyProject/weights/a.pt", "weights", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download failed")
}
