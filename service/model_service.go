package service

import (
	"context"
	"fmt"
	"lucky_project/dao"
	entity2 "lucky_project/entity"
	"math"
	"os"
	"strings"
)

type ModelService struct {
	modelDAO    *dao.ModelDAO
	pathService *ArtifactPathService
}

type ModelDeleteByFileNameResult struct {
	FileName         string `json:"file_name"`
	DeletedRecords   int64  `json:"deleted_records"`
	LocalFileDeleted bool   `json:"local_file_deleted"`
}

func NewModelService() *ModelService {
	return &ModelService{
		modelDAO:    dao.NewModelDAO(),
		pathService: NewArtifactPathService(),
	}
}

func (s *ModelService) CreateModel(ctx context.Context, model *entity2.Model) error {
	if model == nil {
		return dao.ErrNilEntity
	}
	model.StorageServer = normalizeStorageServerField(model.StorageServer)
	model.WeightName = deriveModelWeightName(model)
	if model.WeightName == "" {
		return dao.ErrNilEntity
	}
	if model.WeightSizeMB <= 0 {
		if sizeMB, ok := s.resolveLocalWeightSizeMB(model.WeightName); ok {
			model.WeightSizeMB = sizeMB
		}
	}
	return s.modelDAO.Save(ctx, model)
}

func (s *ModelService) GetAllModels(ctx context.Context, params entity2.QueryParams) (entity2.PageResult, error) {
	models, total, err := s.modelDAO.FindAll(ctx, params)
	if err != nil {
		return entity2.PageResult{}, err
	}
	return entity2.PageResult{
		Total: total,
		List:  models,
	}, nil
}

func (s *ModelService) GetStorageServersByID(ctx context.Context, id uint) ([]string, error) {
	return s.modelDAO.GetStorageServersByID(ctx, id)
}

func (s *ModelService) UpdateStorageServersByID(ctx context.Context, id uint, action string, servers []string) ([]string, error) {
	return s.modelDAO.UpdateStorageServersByID(ctx, id, action, servers)
}

func (s *ModelService) FindByName(ctx context.Context, name string) (*entity2.Model, error) {
	return s.modelDAO.FindByName(ctx, name)
}

func (s *ModelService) GetByID(ctx context.Context, id uint) (*entity2.Model, error) {
	return s.modelDAO.FindByID(ctx, id)
}

func (s *ModelService) GetWeightNameByID(ctx context.Context, id uint) (string, error) {
	return s.modelDAO.FindWeightNameByID(ctx, id)
}

func (s *ModelService) ResolveFilePathByID(ctx context.Context, id uint, storageTarget string) (string, error) {
	if s.pathService == nil {
		return "", ErrArtifactPathServiceNil
	}
	fileName, err := s.modelDAO.FindWeightNameByID(ctx, id)
	if err != nil {
		return "", err
	}
	return s.pathService.BuildPath(ArtifactCategoryWeights, storageTarget, fileName)
}

func (s *ModelService) GetFileNameByID(ctx context.Context, id uint) (string, error) {
	return s.GetWeightNameByID(ctx, id)
}

func (s *ModelService) UpdateModelMetadata(ctx context.Context, id uint, updates map[string]interface{}) (*entity2.Model, error) {
	if len(updates) == 0 {
		return nil, dao.ErrNilEntity
	}

	if rawStorage, ok := updates["storage_server"]; ok {
		if storage, ok := rawStorage.(string); ok {
			updates["storage_server"] = normalizeStorageServerField(storage)
		}
	}

	if rawWeightName, ok := updates["weight_name"]; ok {
		weightName, _ := rawWeightName.(string)
		normalized := deriveFileName(strings.TrimSpace(weightName), "")
		if normalized == "" {
			return nil, dao.ErrNilEntity
		}
		updates["weight_name"] = normalized
		if _, hasSize := updates["weight_size_mb"]; !hasSize {
			if sizeMB, found := s.resolveLocalWeightSizeMB(normalized); found {
				updates["weight_size_mb"] = sizeMB
			}
		}
	}

	return s.modelDAO.UpdateMetadataByID(ctx, id, updates)
}

func (s *ModelService) SyncWeightSizeByFileName(ctx context.Context, fileName string, sizeBytes int64) (int64, float64, error) {
	name := deriveFileName(strings.TrimSpace(fileName), "")
	if name == "" {
		return 0, 0, dao.ErrNilEntity
	}
	if sizeBytes < 0 {
		return 0, 0, dao.ErrNilEntity
	}

	sizeMB := bytesToMB(sizeBytes)
	affected, err := s.modelDAO.UpdateWeightSizeByWeightName(ctx, name, sizeMB)
	if err != nil {
		return 0, 0, err
	}
	return affected, sizeMB, nil
}

func (s *ModelService) DeleteByFileName(ctx context.Context, fileName string) (ModelDeleteByFileNameResult, error) {
	name := deriveFileName(strings.TrimSpace(fileName), "")
	if name == "" {
		return ModelDeleteByFileNameResult{}, dao.ErrNilEntity
	}

	deletedRows, err := s.modelDAO.DeleteByWeightName(ctx, name)
	if err != nil {
		return ModelDeleteByFileNameResult{}, err
	}

	result := ModelDeleteByFileNameResult{
		FileName:       name,
		DeletedRecords: deletedRows,
	}

	if s.pathService == nil {
		return result, nil
	}

	localPath, err := s.pathService.BuildPath(ArtifactCategoryWeights, StorageTargetBackend, name)
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
		return result, fmt.Errorf("stat local model file failed: %w", statErr)
	}
	if info.IsDir() {
		return result, nil
	}

	if removeErr := os.Remove(localPath); removeErr != nil {
		return result, fmt.Errorf("remove local model file failed: %w", removeErr)
	}
	result.LocalFileDeleted = true
	return result, nil
}

func deriveModelWeightName(model *entity2.Model) string {
	if model == nil {
		return ""
	}

	if value := deriveFileName(model.WeightName, ""); value != "" {
		return value
	}
	if value := deriveFileName(model.LegacyFileName, model.LegacyModelPath); value != "" {
		return value
	}
	return ""
}

func (s *ModelService) resolveLocalWeightSizeMB(fileName string) (float64, bool) {
	if s == nil || s.pathService == nil {
		return 0, false
	}

	path, err := s.pathService.BuildPath(ArtifactCategoryWeights, StorageTargetBackend, fileName)
	if err != nil || strings.TrimSpace(path) == "" {
		return 0, false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0, false
	}
	return bytesToMB(info.Size()), true
}

func bytesToMB(sizeBytes int64) float64 {
	if sizeBytes <= 0 {
		return 0
	}
	value := float64(sizeBytes) / (1024 * 1024)
	return math.Round(value*1000) / 1000
}
