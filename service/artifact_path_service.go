package service

import (
	"errors"
	"path/filepath"
	"strings"
)

const (
	ArtifactCategoryWeights  = "weights"
	ArtifactCategoryDatasets = "datasets"

	StorageTargetBackend      = "backend"
	StorageTargetBaiduNetdisk = "baidu_netdisk"
	StorageTargetOtherLocal   = "other_local"

	DefaultBackendWeightsRoot  = "/Users/wenzhengfeng/code/go/lucky_project/weights"
	DefaultBackendDatasetsRoot = "/Users/wenzhengfeng/code/go/lucky_project/datasets"
	DefaultBaiduWeightsRoot    = "/project/luckyProject/weights"
	DefaultBaiduDatasetsRoot   = "/project/luckyProject/datasets"
	DefaultOtherWeightsRoot    = "/project/luckyProject/weights"
	DefaultOtherDatasetsRoot   = "/project/luckyProject/datasets"
)

var (
	ErrInvalidArtifactCategory = errors.New("invalid artifact category")
	ErrInvalidStorageTarget    = errors.New("invalid storage target")
	ErrArtifactFileNameEmpty   = errors.New("artifact file_name is required")
)

type ArtifactPaths struct {
	BackendPath    string `json:"backend_path"`
	BaiduPath      string `json:"baidu_path"`
	OtherLocalPath string `json:"other_local_path"`
}

type ArtifactPathService struct {
	BackendWeightsRoot  string
	BackendDatasetsRoot string
	BaiduWeightsRoot    string
	BaiduDatasetsRoot   string
	OtherWeightsRoot    string
	OtherDatasetsRoot   string
}

func NewArtifactPathService() *ArtifactPathService {
	return &ArtifactPathService{
		BackendWeightsRoot:  DefaultBackendWeightsRoot,
		BackendDatasetsRoot: DefaultBackendDatasetsRoot,
		BaiduWeightsRoot:    DefaultBaiduWeightsRoot,
		BaiduDatasetsRoot:   DefaultBaiduDatasetsRoot,
		OtherWeightsRoot:    DefaultOtherWeightsRoot,
		OtherDatasetsRoot:   DefaultOtherDatasetsRoot,
	}
}

func (s *ArtifactPathService) NormalizeCategory(category string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(category))
	switch value {
	case "", "weight", "weights", "model", "models":
		return ArtifactCategoryWeights, nil
	case "dataset", "datasets":
		return ArtifactCategoryDatasets, nil
	default:
		return "", ErrInvalidArtifactCategory
	}
}

func (s *ArtifactPathService) NormalizeStorageTarget(storageTarget string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(storageTarget))
	switch value {
	case "", StorageTargetBackend:
		return StorageTargetBackend, nil
	case StorageTargetBaiduNetdisk, "baidu", "baidu-pan", "baidu_pan", "baidupan", "pan.baidu", "百度网盘":
		return StorageTargetBaiduNetdisk, nil
	case StorageTargetOtherLocal:
		return StorageTargetOtherLocal, nil
	default:
		return "", ErrInvalidStorageTarget
	}
}

func (s *ArtifactPathService) ResolveRoot(category, storageTarget string) (string, error) {
	normalizedCategory, err := s.NormalizeCategory(category)
	if err != nil {
		return "", err
	}
	normalizedTarget, err := s.NormalizeStorageTarget(storageTarget)
	if err != nil {
		return "", err
	}

	switch normalizedTarget {
	case StorageTargetBackend:
		if normalizedCategory == ArtifactCategoryWeights {
			return strings.TrimSpace(s.BackendWeightsRoot), nil
		}
		return strings.TrimSpace(s.BackendDatasetsRoot), nil
	case StorageTargetBaiduNetdisk:
		if normalizedCategory == ArtifactCategoryWeights {
			return strings.TrimSpace(s.BaiduWeightsRoot), nil
		}
		return strings.TrimSpace(s.BaiduDatasetsRoot), nil
	case StorageTargetOtherLocal:
		if normalizedCategory == ArtifactCategoryWeights {
			return strings.TrimSpace(s.OtherWeightsRoot), nil
		}
		return strings.TrimSpace(s.OtherDatasetsRoot), nil
	default:
		return "", ErrInvalidStorageTarget
	}
}

func (s *ArtifactPathService) BuildPath(category, storageTarget, fileName string) (string, error) {
	root, err := s.ResolveRoot(category, storageTarget)
	if err != nil {
		return "", err
	}
	name, err := normalizeArtifactFileName(fileName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(root) == "" {
		return "", ErrInvalidStorageTarget
	}
	return filepath.ToSlash(filepath.Join(root, name)), nil
}

func (s *ArtifactPathService) BuildAllPaths(category, fileName string) (ArtifactPaths, error) {
	backendPath, err := s.BuildPath(category, StorageTargetBackend, fileName)
	if err != nil {
		return ArtifactPaths{}, err
	}
	baiduPath, err := s.BuildPath(category, StorageTargetBaiduNetdisk, fileName)
	if err != nil {
		return ArtifactPaths{}, err
	}
	otherPath, err := s.BuildPath(category, StorageTargetOtherLocal, fileName)
	if err != nil {
		return ArtifactPaths{}, err
	}
	return ArtifactPaths{
		BackendPath:    backendPath,
		BaiduPath:      baiduPath,
		OtherLocalPath: otherPath,
	}, nil
}

func (s *ArtifactPathService) GenerateStoredFileName(artifactName, originalFilename string) (string, error) {
	original := strings.TrimSpace(filepath.Base(originalFilename))
	if original == "" || original == "." || original == string(filepath.Separator) {
		return "", ErrInvalidUploadFile
	}

	originalExt := filepath.Ext(original)
	ext := originalExt
	baseCandidate := strings.TrimSuffix(original, originalExt)

	rawArtifactName := strings.TrimSpace(artifactName)
	if rawArtifactName != "" {
		candidateName := strings.TrimSpace(filepath.Base(rawArtifactName))
		if candidateName != "" && candidateName != "." && candidateName != string(filepath.Separator) {
			candidateExt := filepath.Ext(candidateName)
			switch {
			case candidateExt == "":
				baseCandidate = candidateName
				ext = originalExt
			case strings.EqualFold(candidateExt, originalExt):
				baseCandidate = strings.TrimSuffix(candidateName, candidateExt)
				ext = candidateExt
			default:
				// 支持 artifact_name 包含版本号小数点（如 yolo26n_v6.0），此时沿用原始扩展名。
				baseCandidate = candidateName
				ext = originalExt
			}
		}
	}

	base := sanitizeFileName(baseCandidate)
	if strings.TrimSpace(base) == "" {
		base = "file"
	}

	return base + ext, nil
}

func normalizeArtifactFileName(fileName string) (string, error) {
	name := strings.TrimSpace(filepath.Base(fileName))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "", ErrArtifactFileNameEmpty
	}
	return name, nil
}

func deriveFileName(fileName, legacyPath string) string {
	name := strings.TrimSpace(fileName)
	if name != "" {
		if normalized, err := normalizeArtifactFileName(name); err == nil {
			return normalized
		}
	}

	legacy := strings.TrimSpace(strings.ReplaceAll(legacyPath, "\\", "/"))
	if legacy == "" {
		return ""
	}
	derived := strings.TrimSpace(filepath.Base(legacy))
	if derived == "" || derived == "." || derived == string(filepath.Separator) {
		return ""
	}
	return derived
}
