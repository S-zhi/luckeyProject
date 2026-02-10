package service

import (
	"context"
	"lucky_project/baidusdk/user-api"
	"lucky_project/config"
	"lucky_project/internal/dao"
	"lucky_project/internal/entity"
)

type TrainingResultService struct {
	trainingDAO *dao.TrainingResultDAO
}

func NewTrainingResultService() *TrainingResultService {
	return &TrainingResultService{
		trainingDAO: dao.NewTrainingResultDAO(),
	}
}

func (s *TrainingResultService) CreateTrainingResult(ctx context.Context, result *entity.ModelTrainingResult) error {
	return s.trainingDAO.Save(ctx, result)
}

func (s *TrainingResultService) GetAllResults(ctx context.Context, params entity.QueryParams) (entity.PageResult, error) {
	results, total, err := s.trainingDAO.FindAll(ctx, params)
	if err != nil {
		return entity.PageResult{}, err
	}
	return entity.PageResult{
		Total: total,
		List:  results,
	}, nil
}

func (s *TrainingResultService) UploadWeight(ctx context.Context, id uint, remotePath string) error {
	result, err := s.trainingDAO.FindByID(ctx, id)
	if err != nil {
		return err
	}
	result.RemotePath = remotePath
	return result.Upload(config.AppConfig.Baidu.AccessToken, config.AppConfig.Baidu.ShardSize)
}

func (s *TrainingResultService) ListRemoteFiles(dir string) (*user_api.FileListResponse, error) {
	var result entity.ModelTrainingResult
	return result.ListFiles(config.AppConfig.Baidu.AccessToken, dir)
}
