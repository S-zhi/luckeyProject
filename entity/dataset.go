package entity

import (
	"encoding/json"
	"time"
)

type Dataset struct {
	ID            uint            `gorm:"primaryKey;column:id" json:"id"`
	Name          string          `gorm:"column:name" json:"name"`
	StorageServer string          `gorm:"column:storage_server" json:"storage_server"`
	Description   *string         `gorm:"column:description" json:"description"`
	TaskType      string          `gorm:"column:task_type" json:"task_type"`
	DatasetFormat string          `gorm:"column:dataset_format" json:"dataset_format"`
	DatasetPath   string          `gorm:"column:dataset_path" json:"dataset_path"`
	ConfigPath    *string         `gorm:"column:config_path" json:"config_path"`
	Version       string          `gorm:"column:version" json:"version"`
	NumClasses    *uint           `gorm:"column:num_classes" json:"num_classes"`
	ClassNames    json.RawMessage `gorm:"column:class_names;type:json" json:"class_names"`
	TrainCount    *uint           `gorm:"column:train_count" json:"train_count"`
	ValCount      *uint           `gorm:"column:val_count" json:"val_count"`
	TestCount     *uint           `gorm:"column:test_count" json:"test_count"`
	SizeMB        float64         `gorm:"column:size_mb" json:"size_mb"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Dataset) TableName() string {
	return "datasets"
}
