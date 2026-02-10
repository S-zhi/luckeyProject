package entity

import "time"

type Model struct {
	ID            uint      `gorm:"primaryKey;column:id" json:"id"`
	Name          string    `gorm:"column:name" json:"name"`
	StorageServer string    `gorm:"column:storage_server" json:"storage_server"`
	Paper         *string   `gorm:"column:paper" json:"paper"`
	ParamsURL     *string   `gorm:"column:params_url" json:"params_url"`
	ModelPath     string    `gorm:"column:model_path" json:"model_path"`
	BaseModelID   *uint     `gorm:"column:base_model_id" json:"base_model_id"`
	ImplType      string    `gorm:"column:impl_type" json:"impl_type"`
	DatasetID     uint      `gorm:"column:dataset_id" json:"dataset_id"`
	SizeMB        float64   `gorm:"column:size_mb" json:"size_mb"`
	Version       string    `gorm:"column:version" json:"version"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	TrainTaskID   *uint     `gorm:"column:train_task_id" json:"train_task_id"`
	Description   *string   `gorm:"column:description" json:"description"`
	TaskType      string    `gorm:"column:task_type" json:"task_type"`
}

func (Model) TableName() string {
	return "models"
}
