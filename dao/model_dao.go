package dao

import (
	"context"
	"fmt"
	entity2 "lucky_project/entity"
	"lucky_project/infrastructure/db"
	"strings"

	"gorm.io/gorm"
)

type ModelDAO struct {
	DB *gorm.DB
}

func NewModelDAO() *ModelDAO {
	return &ModelDAO{
		DB: db.DB,
	}
}

func (d *ModelDAO) Save(ctx context.Context, model *entity2.Model) error {
	if model == nil {
		return ErrNilEntity
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return fmt.Errorf("save model failed: %w", err)
	}
	return dbConn.Create(model).Error
}

func (d *ModelDAO) FindByID(ctx context.Context, id uint) (*entity2.Model, error) {
	if id == 0 {
		return nil, ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return nil, fmt.Errorf("find model by id failed: %w", err)
	}

	var model entity2.Model
	err = dbConn.First(&model, id).Error
	return &model, err
}

func (d *ModelDAO) FindAll(ctx context.Context, params entity2.QueryParams) ([]entity2.Model, int64, error) {
	var models []entity2.Model
	var total int64

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("find models failed: %w", err)
	}

	dbConn = dbConn.Model(&entity2.Model{})

	// 1. 基础模糊搜索
	if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
		dbConn = dbConn.Where("model_name LIKE ?", "%"+keyword+"%")
	}

	// 2. 指标组合过滤 (多选不同的指标进行组合查询)
	if name := strings.TrimSpace(params.Name); name != "" {
		dbConn = dbConn.Where("model_name = ?", name)
	}
	if params.ModelType != nil {
		dbConn = dbConn.Where("model_type = ?", *params.ModelType)
	}
	if params.IsLatest != nil {
		dbConn = dbConn.Where("is_latest = ?", *params.IsLatest)
	}
	if params.IsBasicModel != nil {
		dbConn = dbConn.Where("is_basic_model = ?", *params.IsBasicModel)
	}
	if params.Algorithm != "" {
		dbConn = dbConn.Where("algorithm = ?", params.Algorithm)
	}
	if params.Framework != "" {
		dbConn = dbConn.Where("framework = ?", params.Framework)
	}
	if params.DatasetID != nil {
		dbConn = dbConn.Where("dataset_id = ?", *params.DatasetID)
	}

	// 3. 排序规则 (根据 WeightSizeMB)
	orderStr := "id DESC" // 默认按 ID 降序
	switch strings.ToLower(strings.TrimSpace(params.WeightSort)) {
	case "asc":
		orderStr = "weight_size_mb ASC"
	case "desc":
		orderStr = "weight_size_mb DESC"
	}

	// 4. 获取总数
	err = dbConn.Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("count models failed: %w", err)
	}

	// 5. 执行分页查询
	offset, limit := pagination(params)
	err = dbConn.Order(orderStr).Offset(offset).Limit(limit).Find(&models).Error
	if err != nil {
		return nil, 0, fmt.Errorf("query models failed: %w", err)
	}

	return models, total, err
}
