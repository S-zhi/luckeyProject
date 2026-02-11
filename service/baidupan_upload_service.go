package service

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"

	baidupanplus "github.com/S-zhi/baidupansdk/baidupanplus"

	"lucky_project/config"
)

const (
	DefaultBaiduPanLogPath = "logs/baiduPanSDK.log"
	BaiduWeightsRemoteDir  = "/project/luckyProject/weights"
	BaiduDatasetsRemoteDir = "/project/luckyProject/datasets"
)

var (
	ErrBaiduPanAccessTokenRequired = errors.New("baidu pan access_token is required")
	ErrInvalidBaiduRemoteDir       = errors.New("invalid baidu remote dir")
)

// BaiduPanUploader wraps baidupansdk upload flow and serializes calls for the SDK's global config state.
type BaiduPanUploader struct {
	AccessToken string
	IsSVIP      bool
	LogPath     string
	mu          sync.Mutex
}

func NewBaiduPanUploaderFromConfig() *BaiduPanUploader {
	var cfg config.BaiduPanConfig
	if config.AppConfig != nil {
		cfg = config.AppConfig.BaiduPan
	}

	logPath := strings.TrimSpace(cfg.LogPath)
	if logPath == "" {
		logPath = DefaultBaiduPanLogPath
	}

	return &BaiduPanUploader{
		AccessToken: strings.TrimSpace(cfg.AccessToken),
		IsSVIP:      cfg.IsSVIP,
		LogPath:     logPath,
	}
}

func (u *BaiduPanUploader) Upload(localPath, remoteDir string) (string, error) {
	if strings.TrimSpace(u.AccessToken) == "" {
		return "", ErrBaiduPanAccessTokenRequired
	}

	normalizedDir, err := normalizeBaiduRemoteDir(remoteDir)
	if err != nil {
		return "", err
	}

	baseName := filepath.Base(localPath)
	if strings.TrimSpace(baseName) == "" || baseName == "." || baseName == string(filepath.Separator) {
		return "", ErrInvalidUploadFile
	}

	remotePath := path.Join(normalizedDir, baseName)

	u.mu.Lock()
	defer u.mu.Unlock()

	baidupanplus.NewBasicConfig(u.AccessToken, u.IsSVIP, u.LogPath)
	uploadConfig := baidupanplus.NewUploadFileConfig(localPath, remotePath)
	if err := baidupanplus.UploadFileWithConfig(uploadConfig); err != nil {
		return "", fmt.Errorf("upload file to baidu pan failed: %w", err)
	}

	return remotePath, nil
}

func normalizeBaiduRemoteDir(remoteDir string) (string, error) {
	value := strings.TrimSpace(remoteDir)
	if value == "" {
		return "", ErrInvalidBaiduRemoteDir
	}

	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = path.Clean(value)
	if value == "." || value == "/" {
		return "", ErrInvalidBaiduRemoteDir
	}
	return value, nil
}
