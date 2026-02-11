package service

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var (
	ErrInvalidUploadFile      = errors.New("invalid upload file")
	ErrInvalidUploadSubdir    = errors.New("invalid upload subdir")
	ErrBaiduUploaderNil       = errors.New("baidu uploader is nil")
	ErrArtifactPathServiceNil = errors.New("artifact path service is nil")
)

const (
	DefaultStorageServer = StorageTargetBackend

	// Exported aliases kept for compatibility with existing tests/imports.
	DefaultWeightsSyncDir  = DefaultBackendWeightsRoot
	DefaultDatasetsSyncDir = DefaultBackendDatasetsRoot

	uploadCategoryModels   = ArtifactCategoryWeights
	uploadCategoryDatasets = ArtifactCategoryDatasets
)

type UploadResult struct {
	FileName      string        `json:"file_name"`
	SavedPath     string        `json:"saved_path"`
	ResolvedPath  string        `json:"resolved_path"`
	Paths         ArtifactPaths `json:"paths"`
	Size          int64         `json:"size"`
	StorageServer string        `json:"storage_server"`
	StorageTarget string        `json:"storage_target"`
	UploadToBaidu bool          `json:"upload_to_baidu"`
	BaiduUploaded bool          `json:"baidu_uploaded"`
	BaiduPath     string        `json:"baidu_path,omitempty"`
}

type BaiduUploader interface {
	Upload(localPath, remoteDir string) (string, error)
}

type UploadService struct {
	PathService   *ArtifactPathService
	BaiduUploader BaiduUploader
}

func NewUploadService() *UploadService {
	return &UploadService{
		PathService:   NewArtifactPathService(),
		BaiduUploader: NewBaiduPanUploaderFromConfig(),
	}
}

func (s *UploadService) SaveModelFile(file *multipart.FileHeader, artifactName, storageTarget, storageServer string, uploadToBaidu bool) (UploadResult, error) {
	return s.save(file, uploadCategoryModels, artifactName, storageTarget, storageServer, uploadToBaidu)
}

func (s *UploadService) SaveDatasetFile(file *multipart.FileHeader, artifactName, storageTarget, storageServer string, uploadToBaidu bool) (UploadResult, error) {
	return s.save(file, uploadCategoryDatasets, artifactName, storageTarget, storageServer, uploadToBaidu)
}

func (s *UploadService) save(file *multipart.FileHeader, category, artifactName, storageTarget, storageServer string, uploadToBaidu bool) (UploadResult, error) {
	if file == nil || strings.TrimSpace(file.Filename) == "" {
		return UploadResult{}, ErrInvalidUploadFile
	}
	if s.PathService == nil {
		return UploadResult{}, ErrArtifactPathServiceNil
	}

	normalizedStorageServer := normalizeStorageServer(storageServer)
	normalizedStorageTarget, actualUploadToBaidu, err := s.resolveUploadTarget(storageTarget, normalizedStorageServer, uploadToBaidu)
	if err != nil {
		return UploadResult{}, err
	}

	storedFileName, err := s.PathService.GenerateStoredFileName(artifactName, file.Filename)
	if err != nil {
		return UploadResult{}, err
	}
	paths, err := s.PathService.BuildAllPaths(category, storedFileName)
	if err != nil {
		return UploadResult{}, err
	}

	writeTarget := normalizedStorageTarget
	if normalizedStorageTarget == StorageTargetBaiduNetdisk {
		// Baidu upload flow: persist to backend first, then upload to Baidu.
		writeTarget = StorageTargetBackend
	}
	resolvedPath, err := s.PathService.BuildPath(category, writeTarget, storedFileName)
	if err != nil {
		return UploadResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return UploadResult{}, fmt.Errorf("create upload dir failed: %w", err)
	}

	src, err := file.Open()
	if err != nil {
		return UploadResult{}, fmt.Errorf("open upload file failed: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(resolvedPath)
	if err != nil {
		return UploadResult{}, fmt.Errorf("create target file failed: %w", err)
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		return UploadResult{}, fmt.Errorf("save upload file failed: %w", err)
	}

	result := UploadResult{
		FileName:      storedFileName,
		SavedPath:     filepath.ToSlash(resolvedPath),
		ResolvedPath:  filepath.ToSlash(resolvedPath),
		Paths:         paths,
		Size:          n,
		StorageServer: normalizedStorageServer,
		StorageTarget: normalizedStorageTarget,
		UploadToBaidu: actualUploadToBaidu,
	}

	if !actualUploadToBaidu {
		return result, nil
	}
	if s.BaiduUploader == nil {
		return UploadResult{}, ErrBaiduUploaderNil
	}

	baiduRemoteDir, err := s.PathService.ResolveRoot(category, StorageTargetBaiduNetdisk)
	if err != nil {
		return UploadResult{}, err
	}

	remotePath, err := s.BaiduUploader.Upload(resolvedPath, baiduRemoteDir)
	if err != nil {
		return UploadResult{}, err
	}
	result.BaiduUploaded = true
	result.BaiduPath = remotePath

	return result, nil
}

func sanitizeFileName(name string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(name) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}

	cleaned := strings.Trim(b.String(), "._")
	if cleaned == "" {
		return "file"
	}
	return cleaned
}

func (s *UploadService) resolveUploadTarget(storageTarget, storageServer string, uploadToBaidu bool) (string, bool, error) {
	candidate := strings.TrimSpace(storageTarget)
	if uploadToBaidu {
		candidate = StorageTargetBaiduNetdisk
	}
	if candidate == "" && isBaiduStorageServer(storageServer) {
		candidate = StorageTargetBaiduNetdisk
	}
	if candidate == "" {
		candidate = StorageTargetBackend
	}

	normalized, err := s.PathService.NormalizeStorageTarget(candidate)
	if err != nil {
		return "", false, err
	}
	return normalized, normalized == StorageTargetBaiduNetdisk, nil
}

func normalizeStorageServer(storageServer string) string {
	value := strings.TrimSpace(storageServer)
	if value == "" {
		return DefaultStorageServer
	}
	return value
}

func isBaiduStorageServer(storageServer string) bool {
	switch strings.ToLower(strings.TrimSpace(storageServer)) {
	case StorageTargetBaiduNetdisk, "baidu", "baidu-pan", "baidu_pan", "baidupan", "pan.baidu", "百度网盘":
		return true
	default:
		return false
	}
}
