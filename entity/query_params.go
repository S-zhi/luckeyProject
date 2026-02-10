package entity

// QueryParams 定义通用的查询参数
type QueryParams struct {
	Page     int    `form:"page"`      // 页码
	PageSize int    `form:"page_size"` // 每页数量
	Keyword  string `form:"keyword"`   // 搜索关键字 (模糊匹配名称等)
	Name     string `form:"name"`      // 过滤字段：名称

	// 模型特有过滤指标 (支持组合多选指标，每个指标为单选形式)
	ModelType    *int8  `form:"model_type"`     // 模型类型 (1:检测, 2:分割等)
	IsLatest     *bool  `form:"is_latest"`      // 是否最新
	IsBasicModel *bool  `form:"is_basic_model"` // 是否基础模型
	Algorithm    string `form:"algorithm"`      // 算法
	Framework    string `form:"framework"`      // 框架
	DatasetID    *uint  `form:"dataset_id"`     // 数据集ID

	// 数据集特有过滤指标
	DatasetType    *int8 `form:"dataset_type"`    // 数据集类型 (1:检测, 2:分割等)
	StorageType    *int8 `form:"storage_type"`    // 存储类型 (1:本地, 2:OSS等)
	AnnotationType *int8 `form:"annotation_type"` // 标注格式 (1:YOLO, 2:COCO等)

	// 训练结果特有过滤指标
	TrainingModelID   *uint `form:"training_model_id"`   // 训练关联的模型ID
	TrainingDatasetID *uint `form:"training_dataset_id"` // 训练关联的数据集ID
	TrainingStatus    *int8 `form:"training_status"`     // 训练状态

	// 排序字段
	WeightSort string `form:"weight_sort"` // 权重大小 (WeightSizeMB) 排序: "asc" 或 "desc"
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
