package service

import (
	"context"
	"fmt"
	"lucky_project/dao"
	entity2 "lucky_project/entity"
	"strings"
)

type ModelService struct {
	modelDAO *dao.ModelDAO
}

func NewModelService() *ModelService {
	return &ModelService{
		modelDAO: dao.NewModelDAO(),
	}
}

func (s *ModelService) CreateModel(ctx context.Context, model *entity2.Model) error {
	normalizeModelNameWithVersion(model)
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

func normalizeModelNameWithVersion(model *entity2.Model) {
	if model == nil {
		return
	}

	baseName := strings.TrimSpace(model.Name)
	version := strings.TrimSpace(model.Version)
	if baseName == "" || version == "" {
		return
	}

	suffix := "_" + version
	if strings.HasSuffix(strings.ToLower(baseName), strings.ToLower(suffix)) {
		model.Name = baseName
		return
	}

	model.Name = fmt.Sprintf("%s%s", baseName, suffix)
}
