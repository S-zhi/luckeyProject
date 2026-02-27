package service

import (
	"context"
	"fmt"
	"lucky_project/dao"
	entity2 "lucky_project/entity"
	"os"
	"strconv"
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

// CreateModel 创建新的模型记录
func (s *ModelService) CreateModel(ctx context.Context, model *entity2.Model) error {
	if model == nil {
		return dao.ErrNilEntity
	}
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

// GetAllModels 获取所有模型的分页结果
func (s *ModelService) GetAllModels(ctx context.Context, params entity2.QueryParams) (entity2.PageResult, error) {
	models, total, err := s.modelDAO.FindAll(ctx, params)
	if err != nil {
		return entity2.PageResult{}, err
	}
	for i := range models {
		models[i].StorageServer = normalizeStorageServerCodeField(models[i].StorageServer)
	}
	return entity2.PageResult{
		Total: total,
		List:  models,
	}, nil
}

// GetStorageServersByID 根据ID获取存储服务器列表
func (s *ModelService) GetStorageServersByID(ctx context.Context, id uint) ([]string, error) {
	return s.modelDAO.GetStorageServersByID(ctx, id)
}

// UpdateStorageServersByID 根据ID更新存储服务器列表
func (s *ModelService) UpdateStorageServersByID(ctx context.Context, id uint, action string, servers []string) ([]string, error) {
	return s.modelDAO.UpdateStorageServersByID(ctx, id, action, servers)
}

// FindByName 根据名称查找模型
func (s *ModelService) FindByName(ctx context.Context, name string) (*entity2.Model, error) {
	return s.modelDAO.FindByName(ctx, name)
}

// GetByID 根据ID获取模型
func (s *ModelService) GetByID(ctx context.Context, id uint) (*entity2.Model, error) {
	return s.modelDAO.FindByID(ctx, id)
}

// GetWeightNameByID 根据ID获取权重文件名
func (s *ModelService) GetWeightNameByID(ctx context.Context, id uint) (string, error) {
	return s.modelDAO.FindWeightNameByID(ctx, id)
}

// ResolveFilePathByID 根据ID解析文件路径
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

// GetFileNameByID 根据ID获取文件名
func (s *ModelService) GetFileNameByID(ctx context.Context, id uint) (string, error) {
	return s.GetWeightNameByID(ctx, id)
}

// UpdateModelMetadata 更新模型元数据
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

// SyncWeightSizeByFileName 同步权重文件大小
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

// DeleteByFileName 根据文件名删除模型
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
		return result, fmt.Errorf("获取本地模型文件信息失败: %w", statErr)
	}
	if info.IsDir() {
		return result, nil
	}

	if removeErr := os.Remove(localPath); removeErr != nil {
		return result, fmt.Errorf("删除本地模型文件失败: %w", removeErr)
	}
	result.LocalFileDeleted = true
	return result, nil
}

// deriveModelWeightName 推导模型权重文件名
func deriveModelWeightName(model *entity2.Model) string {
	if model == nil {
		return ""
	}
	var array []string

	array = strings.Split(model.WeightName, ".")
	model.WeightName = model.Name + "_v" + strconv.FormatFloat(model.Version, 'f', -1, 64) + "." + array[len(array)-1]
	return model.WeightName

}

// resolveLocalWeightSizeMB 解析本地权重文件大小(MB)
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
