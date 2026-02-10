package dao

import (
	"context"
	"fmt"
	"lucky_project/internal/entity"
	"lucky_project/pkg/db"

	"gorm.io/gorm"
)

type TrainingResultDAO struct {
	DB *gorm.DB
}

func NewTrainingResultDAO() *TrainingResultDAO {
	return &TrainingResultDAO{
		DB: db.DB,
	}
}

func (d *TrainingResultDAO) Save(ctx context.Context, result *entity.ModelTrainingResult) error {
	if result == nil {
		return ErrNilEntity
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return fmt.Errorf("save training result failed: %w", err)
	}
	return dbConn.Create(result).Error
}

func (d *TrainingResultDAO) FindByID(ctx context.Context, id uint) (*entity.ModelTrainingResult, error) {
	if id == 0 {
		return nil, ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return nil, fmt.Errorf("find training result by id failed: %w", err)
	}

	var result entity.ModelTrainingResult
	err = dbConn.First(&result, id).Error
	return &result, err
}

func (d *TrainingResultDAO) FindAll(ctx context.Context, params entity.QueryParams) ([]entity.ModelTrainingResult, int64, error) {
	var results []entity.ModelTrainingResult
	var total int64

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("find training results failed: %w", err)
	}

	dbConn = dbConn.Model(&entity.ModelTrainingResult{})

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
		return nil, 0, fmt.Errorf("count training results failed: %w", err)
	}

	// 3. 执行分页查询 (默认 ID 降序)
	offset, limit := pagination(params)
	err = dbConn.Order("id DESC").Offset(offset).Limit(limit).Find(&results).Error
	if err != nil {
		return nil, 0, fmt.Errorf("query training results failed: %w", err)
	}

	return results, total, err
}
