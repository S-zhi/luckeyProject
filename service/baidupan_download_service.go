package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	baidupanplus "github.com/S-zhi/baidupansdk/baidupanplus"
)

var (
	ErrBaiduDownloaderNil          = errors.New("baidu downloader is nil")
	ErrInvalidDownloadCategory     = errors.New("invalid download category")
	ErrInvalidBaiduDownloadPath    = errors.New("invalid baidu remote path")
	ErrInvalidBaiduDownloadFile    = errors.New("invalid baidu remote file name")
	ErrInvalidLocalDownloadFile    = errors.New("invalid local download file")
	ErrBaiduDownloadTargetRequired = errors.New("download target is required")
)

const (
	DownloadCategoryWeights  = ArtifactCategoryWeights
	DownloadCategoryDatasets = ArtifactCategoryDatasets
)

type BaiduDownloader interface {
	Download(remotePath, localPath string) error
}

type BaiduDownloadResult struct {
	RemotePath string `json:"remote_path"`
	LocalPath  string `json:"local_path"`
	FileName   string `json:"file_name"`
	Category   string `json:"category"`
	Size       int64  `json:"size"`
}

type BaiduDownloadService struct {
	PathService *ArtifactPathService
	Downloader  BaiduDownloader
}

type BaiduPanDownloader struct {
	AccessToken string
	IsSVIP      bool
	LogPath     string
	mu          sync.Mutex
}

func NewBaiduPanDownloaderFromConfig() *BaiduPanDownloader {
	uploader := NewBaiduPanUploaderFromConfig()
	return &BaiduPanDownloader{
		AccessToken: uploader.AccessToken,
		IsSVIP:      uploader.IsSVIP,
		LogPath:     uploader.LogPath,
	}
}

func (d *BaiduPanDownloader) Download(remotePath, localPath string) error {
	if strings.TrimSpace(d.AccessToken) == "" {
		return ErrBaiduPanAccessTokenRequired
	}

	normalizedRemote, err := normalizeBaiduRemotePath(remotePath)
	if err != nil {
		return err
	}

	normalizedLocal := strings.TrimSpace(localPath)
	if normalizedLocal == "" {
		return ErrInvalidLocalDownloadFile
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	baidupanplus.NewBasicConfig(d.AccessToken, d.IsSVIP, d.LogPath)
	downloadConfig := baidupanplus.NewDownloadFileConfig(normalizedLocal, normalizedRemote)
	if err := baidupanplus.DownloadFileWithConfig(downloadConfig); err != nil {
		return fmt.Errorf("download file from baidu pan failed: %w", err)
	}
	return nil
}

func NewBaiduDownloadService() *BaiduDownloadService {
	return &BaiduDownloadService{
		PathService: NewArtifactPathService(),
		Downloader:  NewBaiduPanDownloaderFromConfig(),
	}
}

func (s *BaiduDownloadService) DownloadToLocal(remotePath, category, fileName string) (BaiduDownloadResult, error) {
	if s.Downloader == nil {
		return BaiduDownloadResult{}, ErrBaiduDownloaderNil
	}
	if s.PathService == nil {
		return BaiduDownloadResult{}, ErrArtifactPathServiceNil
	}

	normalizedRemote, err := normalizeBaiduRemotePath(remotePath)
	if err != nil {
		return BaiduDownloadResult{}, err
	}

	normalizedCategory, err := s.PathService.NormalizeCategory(category)
	if err != nil {
		if errors.Is(err, ErrInvalidArtifactCategory) {
			return BaiduDownloadResult{}, ErrInvalidDownloadCategory
		}
		return BaiduDownloadResult{}, err
	}

	targetFile, err := buildTargetFileName(normalizedRemote, fileName)
	if err != nil {
		return BaiduDownloadResult{}, err
	}

	localPath, err := s.PathService.BuildPath(normalizedCategory, StorageTargetBackend, targetFile)
	if err != nil {
		return BaiduDownloadResult{}, err
	}
	if strings.TrimSpace(localPath) == "" {
		return BaiduDownloadResult{}, ErrBaiduDownloadTargetRequired
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return BaiduDownloadResult{}, fmt.Errorf("create local download dir failed: %w", err)
	}

	if err := s.Downloader.Download(normalizedRemote, localPath); err != nil {
		return BaiduDownloadResult{}, err
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return BaiduDownloadResult{}, fmt.Errorf("stat local downloaded file failed: %w", err)
	}

	return BaiduDownloadResult{
		RemotePath: normalizedRemote,
		LocalPath:  filepath.ToSlash(localPath),
		FileName:   targetFile,
		Category:   normalizedCategory,
		Size:       info.Size(),
	}, nil
}

func normalizeBaiduRemotePath(remotePath string) (string, error) {
	value := strings.TrimSpace(remotePath)
	if value == "" {
		return "", ErrInvalidBaiduDownloadPath
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = filepath.ToSlash(filepath.Clean(value))
	if value == "/" || value == "." {
		return "", ErrInvalidBaiduDownloadPath
	}
	return value, nil
}

func buildTargetFileName(remotePath, fileName string) (string, error) {
	custom := strings.TrimSpace(fileName)
	if custom != "" {
		name, err := normalizeArtifactFileName(custom)
		if err != nil {
			return "", ErrInvalidLocalDownloadFile
		}
		return name, nil
	}

	derived := strings.TrimSpace(filepath.Base(remotePath))
	if derived == "" || derived == "." || derived == string(filepath.Separator) {
		return "", ErrInvalidBaiduDownloadFile
	}
	return derived, nil
}
