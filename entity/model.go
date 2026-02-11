package entity

import "time"

type Model struct {
	ID            uint      `gorm:"primaryKey;column:id" json:"id"`
	Name          string    `gorm:"column:name" json:"name"`
	Version       float64   `gorm:"column:version" json:"version"`
	BaseModelID   uint      `gorm:"column:base_model_id" json:"base_model_id"`
	AlgorithmID   *string   `gorm:"column:algorithm_id" json:"algorithm_id"`
	TaskType      string    `gorm:"column:task_type" json:"task_type"`
	Description   *string   `gorm:"column:description" json:"description"`
	Framework     *string   `gorm:"column:framework" json:"framework"`
	WeightSizeMB  float64   `gorm:"column:weight_size_mb" json:"weight_size_mb"`
	CreateTime    time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	Paper         *string   `gorm:"column:paper" json:"paper"`
	ParamsURL     *string   `gorm:"column:params_url" json:"params_url"`
	StorageServer string    `gorm:"column:storage_server" json:"storage_server"`
	WeightName    string    `gorm:"column:weight_name" json:"weight_name"`

	// Legacy compatibility fields from older request payloads.
	LegacyFileName  string `gorm:"-" json:"file_name,omitempty"`
	LegacyModelPath string `gorm:"-" json:"model_path,omitempty"`
}

func (Model) TableName() string {
	return "models"
}
