package service

import (
	"context"
	"fmt"
	"lucky_project/dao"
	entity2 "lucky_project/entity"
	"os"
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
	if dataset == nil {
		return dao.ErrNilEntity
	}

	dataset.StorageServer = normalizeStorageServerField(dataset.StorageServer)
	dataset.FileName = deriveFileName(dataset.FileName, dataset.DatasetPath)
	if dataset.FileName == "" {
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
	if len(updates) == 0 {
		return nil, dao.ErrNilEntity
	}

	if rawStorage, ok := updates["storage_server"]; ok {
		if storage, ok := rawStorage.(string); ok {
			updates["storage_server"] = normalizeStorageServerField(storage)
		}
	}

	if rawFileName, ok := updates["file_name"]; ok {
		fileName, _ := rawFileName.(string)
		normalized := deriveFileName(strings.TrimSpace(fileName), "")
		if normalized == "" {
			return nil, dao.ErrNilEntity
		}
		updates["file_name"] = normalized
		if _, hasSize := updates["size_mb"]; !hasSize {
			if sizeMB, found := s.resolveLocalDatasetSizeMB(normalized); found {
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
