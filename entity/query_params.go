package entity

// QueryParams 定义通用的查询参数
type QueryParams struct {
	Page     int    `form:"page"`      // 页码
	PageSize int    `form:"page_size"` // 每页数量
	Keyword  string `form:"keyword"`   // 搜索关键字 (模糊匹配名称等)
	Name     string `form:"name"`      // 过滤字段：名称

	// models 表过滤字段
	StorageServer string `form:"storage_server"`
	TaskType      string `form:"task_type"`
	AlgorithmID   string `form:"algorithm_id"`
	Framework     string `form:"framework"`
	Version       string `form:"version"`
	BaseModelID   *uint  `form:"base_model_id"`
	SizeSort      string `form:"size_sort"` // weight_size_mb 排序: asc|desc

	// 兼容旧参数
	Algorithm   string `form:"algorithm"`     // 兼容映射到 algorithm_id
	ImplType    string `form:"impl_type"`     // 兼容映射到 algorithm_id
	WeightSort  string `form:"weight_sort"`   // 兼容映射到 weight_size_mb 排序
	DatasetID   *uint  `form:"dataset_id"`    // 兼容旧参数（models 新表已不使用）
	TrainTaskID *uint  `form:"train_task_id"` // 兼容旧参数（models 新表已不使用）

	// 数据集特有过滤指标
	DatasetFormat string `form:"dataset_format"`
	ConfigPath    string `form:"config_path"`
	NumClasses    *uint  `form:"num_classes"`

	// 兼容旧数据集参数
	DatasetType    *int8 `form:"dataset_type"`    // 兼容映射到 task_type
	IsLatest       *bool `form:"is_latest"`       // 新结构无此字段，保留兼容
	StorageType    *int8 `form:"storage_type"`    // 新结构无此字段，保留兼容
	AnnotationType *int8 `form:"annotation_type"` // 新结构无此字段，保留兼容

	// 训练结果特有过滤指标
	TrainingModelID   *uint `form:"training_model_id"`   // 训练关联的模型ID
	TrainingDatasetID *uint `form:"training_dataset_id"` // 训练关联的数据集ID
	TrainingStatus    *int8 `form:"training_status"`     // 训练状态
}

// GetOffset 计算数据库偏移量
func (p *QueryParams) GetOffset() int {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 10
	}
	return (p.Page - 1) * p.PageSize
}

// GetLimit 获取限制条数
func (p *QueryParams) GetLimit() int {
	if p.PageSize <= 0 {
		p.PageSize = 10
	}
	return p.PageSize
}

// PageResult 通用的分页返回结构
type PageResult struct {
	Total int64       `json:"total"` // 总条数
	List  interface{} `json:"list"`  // 数据列表
}
