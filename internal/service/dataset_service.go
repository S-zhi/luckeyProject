package service

import (
	"context"
	user_api "lucky_project/baidusdk/user-api"
	"lucky_project/config"
	"lucky_project/internal/dao"
	"lucky_project/internal/entity"
)

type DatasetService struct {
	datasetDAO *dao.DatasetDAO
}

func NewDatasetService() *DatasetService {
	return &DatasetService{
		datasetDAO: dao.NewDatasetDAO(),
	}
}

func (s *DatasetService) CreateDataset(ctx context.Context, dataset *entity.Dataset) error {
	// Business logic...
	return s.datasetDAO.Save(ctx, dataset)
}

// UploadDataset 上传数据集（压缩文件夹后上传）
func (s *DatasetService) UploadDataset(ctx context.Context, id uint) error {
	dataset, err := s.datasetDAO.FindByID(ctx, id)
	if err != nil {
		return err
	}
	return dataset.Upload(config.AppConfig.Baidu.AccessToken, config.AppConfig.Baidu.ShardSize)
}

// ListRemoteFiles 获取远程文件列表
func (s *DatasetService) ListRemoteFiles(dir string) (*user_api.FileListResponse, error) {
	var dataset entity.Dataset
	return dataset.ListFiles(config.AppConfig.Baidu.AccessToken, dir)
}

func (s *DatasetService) GetAllDatasets(ctx context.Context, params entity.QueryParams) (entity.PageResult, error) {
	datasets, total, err := s.datasetDAO.FindAll(ctx, params)
	if err != nil {
		return entity.PageResult{}, err
	}
	return entity.PageResult{
		Total: total,
		List:  datasets,
	}, nil
}
