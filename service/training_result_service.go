package service

import (
	"context"
	"lucky_project/dao"
	entity2 "lucky_project/entity"
)

type TrainingResultService struct {
	trainingDAO *dao.TrainingResultDAO
}

func NewTrainingResultService() *TrainingResultService {
	return &TrainingResultService{
		trainingDAO: dao.NewTrainingResultDAO(),
	}
}

func (s *TrainingResultService) CreateTrainingResult(ctx context.Context, result *entity2.ModelTrainingResult) error {
	return s.trainingDAO.Save(ctx, result)
}

func (s *TrainingResultService) GetAllResults(ctx context.Context, params entity2.QueryParams) (entity2.PageResult, error) {
	results, total, err := s.trainingDAO.FindAll(ctx, params)
	if err != nil {
		return entity2.PageResult{}, err
	}
	return entity2.PageResult{
		Total: total,
		List:  results,
	}, nil
}
