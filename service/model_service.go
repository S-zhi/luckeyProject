package service

import (
	"context"
	"lucky_project/dao"
	entity2 "lucky_project/entity"
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
