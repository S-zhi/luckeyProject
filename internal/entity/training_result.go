package entity

import (
	"encoding/json"
	"time"
)

type ModelTrainingResult struct {
	ID             uint            `gorm:"primaryKey;column:id" json:"id"`
	ModelID        uint            `gorm:"column:model_id" json:"model_id"`                     // 模型ID
	DatasetID      uint            `gorm:"column:dataset_id" json:"dataset_id"`                 // 数据集ID
	DatasetVersion float64         `gorm:"column:dataset_version" json:"dataset_version"`       // 数据集版本
	TrainingStatus int8            `gorm:"column:training_status" json:"training_status"`       // 1:训练中｜2:成功｜3:失败｜4:中断
	MetricDetail   json.RawMessage `gorm:"column:metric_detail;type:json" json:"metric_detail"` // 评估指标JSON
	WeightPath     string          `gorm:"column:weight_path" json:"weight_path"`               // 训练产出权重文件路径
	CometLogURL    string          `gorm:"column:comet_log_url" json:"comet_log_url"`           // Comet 实验日志URL
	TrainStartTime *time.Time      `gorm:"column:train_start_time" json:"train_start_time"`
	TrainEndTime   *time.Time      `gorm:"column:train_end_time" json:"train_end_time"`
	RemotePath     string          `gorm:"-" json:"remote_path"` // 百度网盘路径 (非数据库字段)
	CreateTime     time.Time       `gorm:"column:create_time;autoCreateTime" json:"create_time"`
}

func (ModelTrainingResult) TableName() string {
	return "lucky_model_training_result"
}
