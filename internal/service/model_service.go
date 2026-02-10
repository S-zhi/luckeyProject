package service

import (
	"context"
	user_api "lucky_project/baidusdk/user-api"
	"lucky_project/config"
	"lucky_project/internal/dao"
	"lucky_project/internal/entity"
)

type ModelService struct {
	modelDAO *dao.ModelDAO
}

func NewModelService() *ModelService {
	return &ModelService{
		modelDAO: dao.NewModelDAO(),
	}
}

func (s *ModelService) CreateModel(ctx context.Context, model *entity.Model) error {
	// Business logic...
	return s.modelDAO.Save(ctx, model)
}

// UploadModel 上传模型到百度网盘
func (s *ModelService) UploadModel(ctx context.Context, id uint) error {
	model, err := s.modelDAO.FindByID(ctx, id)
	if err != nil {
		return err
	}
	return model.Upload(config.AppConfig.Baidu.AccessToken, config.AppConfig.Baidu.ShardSize)
}

// ListRemoteFiles 获取远程文件列表
func (s *ModelService) ListRemoteFiles(dir string) (*user_api.FileListResponse, error) {
	var model entity.Model
	return model.ListFiles(config.AppConfig.Baidu.AccessToken, dir)
}

func (s *ModelService) GetAllModels(ctx context.Context, params entity.QueryParams) (entity.PageResult, error) {
	models, total, err := s.modelDAO.FindAll(ctx, params)
	if err != nil {
		return entity.PageResult{}, err
	}
	return entity.PageResult{
		Total: total,
		List:  models,
	}, nil
}
