package service

import (
	"context"
	"lucky_project/dao"
	entity2 "lucky_project/entity"
)

type DatasetService struct {
	datasetDAO  *dao.DatasetDAO
	pathService *ArtifactPathService
}

func NewDatasetService() *DatasetService {
	return &DatasetService{
		datasetDAO:  dao.NewDatasetDAO(),
		pathService: NewArtifactPathService(),
	}
}

func (s *DatasetService) CreateDataset(ctx context.Context, dataset *entity2.Dataset) error {
	if dataset != nil {
		dataset.StorageServer = normalizeStorageServerField(dataset.StorageServer)
		dataset.FileName = deriveFileName(dataset.FileName, dataset.DatasetPath)
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
