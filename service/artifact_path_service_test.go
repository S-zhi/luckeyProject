package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArtifactPathServiceBuildPath(t *testing.T) {
	svc := NewArtifactPathService()

	backendPath, err := svc.BuildPath(ArtifactCategoryWeights, StorageTargetBackend, "demo.pt")
	assert.NoError(t, err)
	assert.Equal(t, "/Users/wenzhengfeng/code/go/lucky_project/weights/demo.pt", backendPath)

	baiduPath, err := svc.BuildPath(ArtifactCategoryDatasets, StorageTargetBaiduNetdisk, "demo.zip")
	assert.NoError(t, err)
	assert.Equal(t, "/project/luckyProject/datasets/demo.zip", baiduPath)

	otherPath, err := svc.BuildPath(ArtifactCategoryWeights, StorageTargetOtherLocal, "w.pt")
	assert.NoError(t, err)
	assert.Equal(t, "/project/luckyProject/weights/w.pt", otherPath)
}

func TestArtifactPathServiceGenerateStoredFileName(t *testing.T) {
	svc := NewArtifactPathService()

	name, err := svc.GenerateStoredFileName("yolov7_HRW_4.2k", "origin.pt")
	assert.NoError(t, err)
	assert.Equal(t, "yolov7_HRW_4.2k.pt", name)

	name2, err := svc.GenerateStoredFileName("", "yolo26n_v6.0.onnx")
	assert.NoError(t, err)
	assert.Equal(t, "yolo26n_v6.0.onnx", name2)
}
