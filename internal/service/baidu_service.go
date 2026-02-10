package service

import (
	"fmt"
	"lucky_project/baidusdk/user-api"
	"lucky_project/config"
	"lucky_project/pkg/utils"
	"os"
	"path/filepath"
)

type BaiduService struct {
	accessToken string
	shardSize   int64
}

func NewBaiduService() *BaiduService {
	return &BaiduService{
		accessToken: config.AppConfig.Baidu.AccessToken,
		shardSize:   config.AppConfig.Baidu.ShardSize,
	}
}

// UploadModel 上传模型文件
func (s *BaiduService) UploadModel(localPath, remotePath string) error {
	return user_api.UploadFile(s.accessToken, remotePath, localPath, s.shardSize)
}

// UploadDataset 上传数据集文件夹（先压缩）
func (s *BaiduService) UploadDataset(localPath, remotePath string) error {
	// 1. 创建临时压缩文件
	tmpZip := filepath.Join(os.TempDir(), fmt.Sprintf("dataset_%s.zip", filepath.Base(localPath)))
	err := utils.ZipFolder(localPath, tmpZip)
	if err != nil {
		return fmt.Errorf("zip folder failed: %v", err)
	}
	defer os.Remove(tmpZip)

	// 2. 上传压缩后的文件
	return user_api.UploadFile(s.accessToken, remotePath, tmpZip, s.shardSize)
}

// GetFiles 获取文件列表
func (s *BaiduService) GetFiles(dir string) (*user_api.FileListResponse, error) {
	return user_api.GetFileList(s.accessToken, dir, "0", 100)
}
