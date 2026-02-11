package service

import (
	"context"
	"lucky_project/dao"
	entity2 "lucky_project/entity"
	"strings"
)

type ModelService struct {
	modelDAO    *dao.ModelDAO
	pathService *ArtifactPathService
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
	}

	return s.modelDAO.UpdateMetadataByID(ctx, id, updates)
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
