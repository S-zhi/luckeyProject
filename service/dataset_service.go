package service

import (
	"context"
	"lucky_project/dao"
	entity2 "lucky_project/entity"
)

type DatasetService struct {
	datasetDAO *dao.DatasetDAO
}

func NewDatasetService() *DatasetService {
	return &DatasetService{
		datasetDAO: dao.NewDatasetDAO(),
	}
}

func (s *DatasetService) CreateDataset(ctx context.Context, dataset *entity2.Dataset) error {
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
