package dao

import (
	"context"
	"fmt"
	"lucky_project/config"
	entity2 "lucky_project/entity"

	"gorm.io/gorm"
)

type TrainingResultDAO struct {
	DB *gorm.DB
}

// NewTrainingResultDAO 创建 TrainingResultDAO，并注入全局数据库连接。
func NewTrainingResultDAO() *TrainingResultDAO {
	logger := daoLogger().With("dao", "TrainingResultDAO", "method", "NewTrainingResultDAO")
	logger.Info("init training result dao")
	return &TrainingResultDAO{
		DB: config.DB,
	}
}

// Save 保存一条训练结果记录。
func (d *TrainingResultDAO) Save(ctx context.Context, result *entity2.ModelTrainingResult) error {
	logger := daoLogger().With("dao", "TrainingResultDAO", "method", "Save")
	if result == nil {
		logger.Warn("save training result skipped: result is nil")
		return ErrNilEntity
	}
	logger.Info("save training result begin", "model_id", result.ModelID, "dataset_id", result.DatasetID)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("save training result failed: with context", "error", err)
		return fmt.Errorf("save training result failed: %w", err)
	}
	if err := dbConn.Create(result).Error; err != nil {
		logger.Error("save training result failed: db create", "error", err)
		return fmt.Errorf("save training result failed: %w", err)
	}
	logger.Info("save training result success", "id", result.ID)
	return nil
}

// FindByID 根据主键查询单条训练结果记录。
func (d *TrainingResultDAO) FindByID(ctx context.Context, id uint) (*entity2.ModelTrainingResult, error) {
	logger := daoLogger().With("dao", "TrainingResultDAO", "method", "FindByID")
	if id == 0 {
		logger.Warn("find training result by id skipped: invalid id", "id", id)
		return nil, ErrInvalidID
	}
	logger.Info("find training result by id begin", "id", id)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("find training result by id failed: with context", "id", id, "error", err)
		return nil, fmt.Errorf("find training result by id failed: %w", err)
	}

	var result entity2.ModelTrainingResult
	err = dbConn.First(&result, id).Error
	if err != nil {
		logger.Error("find training result by id failed: db query", "id", id, "error", err)
		return &result, err
	}
	logger.Info("find training result by id success", "id", result.ID)
	return &result, err
}

// FindAll 按查询参数分页获取训练结果列表与总数。
func (d *TrainingResultDAO) FindAll(ctx context.Context, params entity2.QueryParams) ([]entity2.ModelTrainingResult, int64, error) {
	logger := daoLogger().With("dao", "TrainingResultDAO", "method", "FindAll")
	var results []entity2.ModelTrainingResult
	var total int64
	logger.Info("find training results begin",
		"page", params.Page,
		"page_size", params.PageSize,
		"training_model_id", params.TrainingModelID,
		"training_dataset_id", params.TrainingDatasetID,
		"training_status", params.TrainingStatus,
	)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("find training results failed: with context", "error", err)
		return nil, 0, fmt.Errorf("find training results failed: %w", err)
	}

	dbConn = dbConn.Model(&entity2.ModelTrainingResult{})

	// 1. 指标组合过滤
	if params.TrainingModelID != nil {
		dbConn = dbConn.Where("model_id = ?", *params.TrainingModelID)
	}
	if params.TrainingDatasetID != nil {
		dbConn = dbConn.Where("dataset_id = ?", *params.TrainingDatasetID)
	}
	if params.TrainingStatus != nil {
		dbConn = dbConn.Where("training_status = ?", *params.TrainingStatus)
	}

	// 2. 获取总数
	err = dbConn.Count(&total).Error
	if err != nil {
		logger.Error("count training results failed", "error", err)
		return nil, 0, fmt.Errorf("count training results failed: %w", err)
	}

	// 3. 执行分页查询 (默认 ID 降序)
	offset, limit := pagination(params)
	err = dbConn.Order("id DESC").Offset(offset).Limit(limit).Find(&results).Error
	if err != nil {
		logger.Error("query training results failed", "error", err)
		return nil, 0, fmt.Errorf("query training results failed: %w", err)
	}

	logger.Info("find training results success", "total", total, "returned", len(results))
	return results, total, err
}
