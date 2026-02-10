package entity

import "time"

type Model struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`
	ModelName    string    `gorm:"column:model_name" json:"model_name"`
	ModelType    int8      `gorm:"column:model_type" json:"model_type"`
	ModelVersion float64   `gorm:"column:model_version" json:"model_version"`
	IsLatest     bool      `gorm:"column:is_latest" json:"is_latest"`
	IsBasicModel bool      `gorm:"column:is_basic_model" json:"is_basic_model"`
	Algorithm    string    `gorm:"column:algorithm" json:"algorithm"`
	Framework    string    `gorm:"column:framework" json:"framework"`
	WeightSizeMB float64   `gorm:"column:weight_size_mb" json:"weight_size_mb"`
	WeightPath   string    `gorm:"column:weight_path" json:"weight_path"`
	DatasetID    uint      `gorm:"column:dataset_id" json:"dataset_id"`
	CreateTime   time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
}

func (Model) TableName() string {
	return "lucky_model_information"
}
