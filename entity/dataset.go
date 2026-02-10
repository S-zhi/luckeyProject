package entity

import (
	"time"
)

type Dataset struct {
	ID             uint      `gorm:"primaryKey;column:id" json:"id"`
	DatasetName    string    `gorm:"column:dataset_name" json:"dataset_name"`       // 数据集名称
	DatasetType    int8      `gorm:"column:dataset_type" json:"dataset_type"`       // 1:检测｜2:分割｜3:分类｜4:姿态估计｜5:OBB
	DatasetVersion float64   `gorm:"column:dataset_version" json:"dataset_version"` // 数据集版本
	IsLatest       bool      `gorm:"column:is_latest" json:"is_latest"`             // 是否最新版本
	SampleCount    uint      `gorm:"column:sample_count" json:"sample_count"`       // 样本数量
	LabelCount     *uint     `gorm:"column:label_count" json:"label_count"`         // 标注数量
	StorageType    int8      `gorm:"column:storage_type" json:"storage_type"`       // 存储类型 1:本地｜2:OSS｜3:S3
	DatasetPath    string    `gorm:"column:dataset_path" json:"dataset_path"`       // 数据集存储路径 (对应之前的 LocalPath)
	AnnotationType int8      `gorm:"column:annotation_type" json:"annotation_type"` // 标注格式 1:YOLO｜2:COCO｜3:VOC
	Description    string    `gorm:"column:description" json:"description"`         // 数据集描述
	CreateTime     time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime     time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
}

func (Dataset) TableName() string {
	return "lucky_dataset_information"
}
