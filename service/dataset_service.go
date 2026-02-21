package service

import (
	"context"
	"fmt"
	"lucky_project/dao"
	entity2 "lucky_project/entity"
	"os"
	"path/filepath"
	"strings"
)

type DatasetService struct {
	datasetDAO  *dao.DatasetDAO
	pathService *ArtifactPathService
}

type DatasetDeleteByFileNameResult struct {
	FileName         string `json:"file_name"`
	DeletedRecords   int64  `json:"deleted_records"`
	LocalFileDeleted bool   `json:"local_file_deleted"`
}

func NewDatasetService() *DatasetService {
	return &DatasetService{
		datasetDAO:  dao.NewDatasetDAO(),
		pathService: NewArtifactPathService(),
	}
}

func (s *DatasetService) CreateDataset(ctx context.Context, dataset *entity2.Dataset) error {
	logger := serviceLogger().With("service", "DatasetService", "method", "CreateDataset")
	if dataset == nil {
		logger.Warn("create dataset skipped: dataset is nil")
		return dao.ErrNilEntity
	}

	dataset.StorageServer = normalizeStorageServerField(dataset.StorageServer)
	originalFileName := dataset.FileName
	dataset.FileName = deriveFileName(dataset.FileName, dataset.DatasetPath)
	dataset.FileName = applyDatasetVersionToFileName(dataset.FileName, dataset.Version)
	logger.Info(
		"normalize dataset file_name",
		"original_file_name", strings.TrimSpace(originalFileName),
		"normalized_file_name", dataset.FileName,
		"version", strings.TrimSpace(dataset.Version),
	)
	if dataset.FileName == "" {
		logger.Warn("create dataset skipped: file_name is empty after normalization", "version", strings.TrimSpace(dataset.Version))
		return dao.ErrNilEntity
	}
	if dataset.SizeMB <= 0 {
		if sizeMB, ok := s.resolveLocalDatasetSizeMB(dataset.FileName); ok {
			dataset.SizeMB = sizeMB
		}
	}

	return s.datasetDAO.Save(ctx, dataset)
}

func (s *DatasetService) GetAllDatasets(ctx context.Context, params entity2.QueryParams) (entity2.PageResult, error) {
	datasets, total, err := s.datasetDAO.FindAll(ctx, params)
	if err != nil {
		return entity2.PageResult{}, err
	}
	for i := range datasets {
		datasets[i].StorageServer = normalizeStorageServerField(datasets[i].StorageServer)
	}
	return entity2.PageResult{
		Total: total,
		List:  datasets,
	}, nil
}

func (s *DatasetService) GetStorageServersByID(ctx context.Context, id uint) ([]string, error) {
	return s.datasetDAO.GetStorageServersByID(ctx, id)
}

func (s *DatasetService) UpdateStorageServersByID(ctx context.Context, id uint, action string, servers []string) ([]string, error) {
	return s.datasetDAO.UpdateStorageServersByID(ctx, id, action, servers)
}

func (s *DatasetService) FindByName(ctx context.Context, name string) (*entity2.Dataset, error) {
	return s.datasetDAO.FindByName(ctx, name)
}

func (s *DatasetService) GetByID(ctx context.Context, id uint) (*entity2.Dataset, error) {
	return s.datasetDAO.FindByID(ctx, id)
}

func (s *DatasetService) GetFileNameByID(ctx context.Context, id uint) (string, error) {
	return s.datasetDAO.FindFileNameByID(ctx, id)
}

func (s *DatasetService) ResolveFilePathByID(ctx context.Context, id uint, storageTarget string) (string, error) {
	if s.pathService == nil {
		return "", ErrArtifactPathServiceNil
	}
	fileName, err := s.datasetDAO.FindFileNameByID(ctx, id)
	if err != nil {
		return "", err
	}
	return s.pathService.BuildPath(ArtifactCategoryDatasets, storageTarget, fileName)
}

func (s *DatasetService) UpdateDatasetMetadata(ctx context.Context, id uint, updates map[string]interface{}) (*entity2.Dataset, error) {
	logger := serviceLogger().With("service", "DatasetService", "method", "UpdateDatasetMetadata", "id", id)
	if len(updates) == 0 {
		logger.Warn("update dataset metadata skipped: empty updates")
		return nil, dao.ErrNilEntity
	}

	if rawStorage, ok := updates["storage_server"]; ok {
		if storage, ok := rawStorage.(string); ok {
			updates["storage_server"] = normalizeStorageServerField(storage)
		}
	}

	_, hasFileName := updates["file_name"]
	_, hasVersion := updates["version"]
	if hasFileName || hasVersion {
		current, err := s.datasetDAO.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}

		targetVersion := strings.TrimSpace(current.Version)
		if rawVersion, ok := updates["version"]; ok {
			version, _ := rawVersion.(string)
			targetVersion = strings.TrimSpace(version)
		}

		targetFileName := strings.TrimSpace(current.FileName)
		if rawFileName, ok := updates["file_name"]; ok {
			fileName, _ := rawFileName.(string)
			normalized := deriveFileName(strings.TrimSpace(fileName), "")
			if normalized == "" {
				logger.Warn("update dataset metadata rejected: invalid file_name", "file_name", fileName)
				return nil, dao.ErrNilEntity
			}
			targetFileName = normalized
		}

		versionedFileName := applyDatasetVersionToFileName(targetFileName, targetVersion)
		if versionedFileName == "" {
			logger.Warn("update dataset metadata rejected: file_name is empty after version normalization", "version", targetVersion)
			return nil, dao.ErrNilEntity
		}
		updates["file_name"] = versionedFileName

		logger.Info(
			"normalize dataset file_name for metadata update",
			"original_file_name", current.FileName,
			"target_file_name", targetFileName,
			"normalized_file_name", versionedFileName,
			"version", targetVersion,
		)

		if _, hasSize := updates["size_mb"]; !hasSize {
			if sizeMB, found := s.resolveLocalDatasetSizeMB(versionedFileName); found {
				updates["size_mb"] = sizeMB
			}
		}
	}

	return s.datasetDAO.UpdateMetadataByID(ctx, id, updates)
}

func (s *DatasetService) SyncSizeByFileName(ctx context.Context, fileName string, sizeBytes int64) (int64, float64, error) {
	name := deriveFileName(strings.TrimSpace(fileName), "")
	if name == "" {
		return 0, 0, dao.ErrNilEntity
	}
	if sizeBytes < 0 {
		return 0, 0, dao.ErrNilEntity
	}

	sizeMB := bytesToMB(sizeBytes)
	affected, err := s.datasetDAO.UpdateSizeByFileName(ctx, name, sizeMB)
	if err != nil {
		return 0, 0, err
	}
	return affected, sizeMB, nil
}

func (s *DatasetService) DeleteByFileName(ctx context.Context, fileName string) (DatasetDeleteByFileNameResult, error) {
	name := deriveFileName(strings.TrimSpace(fileName), "")
	if name == "" {
		return DatasetDeleteByFileNameResult{}, dao.ErrNilEntity
	}

	deletedRows, err := s.datasetDAO.DeleteByFileName(ctx, name)
	if err != nil {
		return DatasetDeleteByFileNameResult{}, err
	}

	result := DatasetDeleteByFileNameResult{
		FileName:       name,
		DeletedRecords: deletedRows,
	}

	if s.pathService == nil {
		return result, nil
	}

	localPath, err := s.pathService.BuildPath(ArtifactCategoryDatasets, StorageTargetBackend, name)
	if err != nil {
		return result, err
	}
	if localPath == "" {
		return result, nil
	}

	info, statErr := os.Stat(localPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return result, nil
		}
		return result, fmt.Errorf("stat local dataset file failed: %w", statErr)
	}
	if info.IsDir() {
		return result, nil
	}

	if removeErr := os.Remove(localPath); removeErr != nil {
		return result, fmt.Errorf("remove local dataset file failed: %w", removeErr)
	}
	result.LocalFileDeleted = true
	return result, nil
}

func (s *DatasetService) resolveLocalDatasetSizeMB(fileName string) (float64, bool) {
	if s == nil || s.pathService == nil {
		return 0, false
	}

	path, err := s.pathService.BuildPath(ArtifactCategoryDatasets, StorageTargetBackend, fileName)
	if err != nil || strings.TrimSpace(path) == "" {
		return 0, false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0, false
	}
	return bytesToMB(info.Size()), true
}

func applyDatasetVersionToFileName(fileName, version string) string {
	name := strings.TrimSpace(fileName)
	if name == "" {
		return ""
	}

	versionToken := normalizeDatasetVersionToken(version)
	if versionToken == "" {
		return name
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	suffix := "_" + versionToken
	if strings.HasSuffix(base, suffix) {
		return base + ext
	}
	return base + suffix + ext
}

func normalizeDatasetVersionToken(version string) string {
	token := strings.TrimSpace(version)
	if token == "" {
		return ""
	}

	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	token = replacer.Replace(token)
	token = strings.Trim(token, "._-")
	return token
}
