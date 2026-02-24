package dao

import (
	"context"
	"fmt"
	"lucky_project/config"
	entity2 "lucky_project/entity"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ModelDAO struct {
	DB *gorm.DB
}

// NewModelDAO 创建 ModelDAO，并注入全局数据库连接。
func NewModelDAO() *ModelDAO {
	logger := daoLogger().With("func", "NewModelDAO")
	logger.Info("初始化 ModelDAO")
	return &ModelDAO{
		DB: config.DB,
	}
}

// Save 按 (name, version) 保存数据元信息；若唯一键冲突则执行更新（upsert）。
func (d *ModelDAO) Save(ctx context.Context, model *entity2.Model) error {
	logger := daoLogger().With("func", "Save")
	if model == nil {
		logger.Error("ModelDAO 保存数据失败：ModelDAO为空")
		return ErrNilEntity
	}
	logger.Info("保存数据开始", "name", model.Name, "version", model.Version)
	weightName, err := deriveWeightName(model.WeightName, model.LegacyFileName, model.LegacyModelPath)
	if err != nil {
		logger.Error("保存数据失败：权重文件名标准化失败", "name", model.Name, "error", err)
		return fmt.Errorf("保存数据失败: %w", err)
	}
	model.WeightName = weightName

	normalizedStorageServer, err := encodeStorageServerValue(parseStorageServerValue(model.StorageServer))
	if err != nil {
		logger.Error("保存数据失败：存储服务标准化失败", "name", model.Name, "error", err)
		return fmt.Errorf("保存数据失败: %w", err)
	}
	model.StorageServer = normalizedStorageServer

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("保存数据失败：绑定上下文失败", "name", model.Name, "error", err)
		return fmt.Errorf("保存数据失败: %w", err)
	}

	if err := dbConn.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}, {Name: "version"}},
		DoUpdates: clause.AssignmentColumns(updatableModelColumns()),
	}).Create(model).Error; err != nil {
		logger.Error("保存数据失败：创建或更新失败", "name", model.Name, "error", err)
		return fmt.Errorf("保存数据失败: %w", err)
	}

	if strings.TrimSpace(model.Name) != "" {
		if err := dbConn.Where("name = ? AND version = ?", model.Name, model.Version).First(model).Error; err != nil {
			logger.Error("保存数据失败：按唯一键回查失败", "name", model.Name, "version", model.Version, "error", err)
			return fmt.Errorf("保存数据失败: %w", err)
		}
	}

	logger.Info("保存数据成功", "id", model.ID, "name", model.Name, "version", model.Version)
	return nil
}

// GetStorageServersByID 查询数据的 storage_server 列并统一返回数组格式。
func (d *ModelDAO) GetStorageServersByID(ctx context.Context, id uint) ([]string, error) {
	logger := daoLogger().With("func", "GetStorageServersByID")
	if id == 0 {
		logger.Warn("查询数据存储服务已跳过：ID无效", "id", id)
		return nil, ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("查询数据存储服务失败：绑定上下文失败", "id", id, "error", err)
		return nil, fmt.Errorf("获取数据存储服务失败: %w", err)
	}

	var row struct {
		StorageServer string `gorm:"column:storage_server"`
	}
	if err := dbConn.Model(&entity2.Model{}).Select("storage_server").Where("id = ?", id).Take(&row).Error; err != nil {
		logger.Error("查询数据存储服务失败：数据库查询失败", "id", id, "error", err)
		return nil, err
	}

	servers := parseStorageServerValue(row.StorageServer)
	logger.Info("查询数据存储服务成功", "id", id, "count", len(servers))
	return servers, nil
}

// UpdateStorageServersByID 按 action(set/add/remove) 更新数据 storage_server。 TODO 请再次检查
func (d *ModelDAO) UpdateStorageServersByID(ctx context.Context, id uint, action string, servers []string) ([]string, error) {
	logger := daoLogger().With("func", "UpdateStorageServersByID")
	if id == 0 {
		logger.Warn("更新数据存储服务已跳过：ID无效", "id", id)
		return nil, ErrInvalidID
	}

	current, err := d.GetStorageServersByID(ctx, id)
	if err != nil {
		logger.Error("更新数据存储服务失败：加载当前值失败", "id", id, "error", err)
		return nil, err
	}
	next, err := applyStorageServerAction(current, action, servers)
	if err != nil {
		logger.Error("更新数据存储服务失败：applyStorageServerAction Failed", "id", id, "action", action, "error", err)
		return nil, err
	}

	encoded, err := encodeStorageServerValue(next)
	if err != nil {
		logger.Error("更新数据存储服务失败：编码失败", "id", id, "error", err)
		return nil, fmt.Errorf("更新数据存储服务失败: %w", err)
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("更新数据存储服务失败：绑定上下文失败", "id", id, "error", err)
		return nil, fmt.Errorf("更新数据存储服务失败: %w", err)
	}

	if err := dbConn.Model(&entity2.Model{}).Where("id = ?", id).Update("storage_server", encoded).Error; err != nil {
		logger.Error("更新数据存储服务失败：数据库更新失败", "id", id, "error", err)
		return nil, fmt.Errorf("更新数据存储服务失败: %w", err)
	}

	logger.Info("更新数据存储服务成功", "id", id, "action", action, "count", len(next))
	return next, nil
}

// updatableModelColumns 返回 upsert 时允许更新的字段列表。
func updatableModelColumns() []string {
	columns := []string{
		"storage_server",
		"base_model_id",
		"algorithm_id",
		"task_type",
		"description",
		"framework",
		"weight_size_mb",
		"paper",
		"params_url",
		"weight_name",
	}
	daoLogger().With("func", "updatableModelColumns").Debug("可更新字段已准备", "count", len(columns))
	return columns
}

// FindWeightNameByID 根据主键查询数据权重文件名。
func (d *ModelDAO) FindWeightNameByID(ctx context.Context, id uint) (string, error) {
	logger := daoLogger().With("func", "FindWeightNameByID")
	if id == 0 {
		logger.Warn("查询数据权重文件名已跳过：ID无效", "id", id)
		return "", ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("查询数据权重文件名失败：绑定上下文失败", "id", id, "error", err)
		return "", fmt.Errorf("查询数据 weight_name 失败: %w", err)
	}

	var row struct {
		WeightName string `gorm:"column:weight_name"`
	}
	if err := dbConn.Model(&entity2.Model{}).Select("weight_name").Where("id = ?", id).Take(&row).Error; err != nil {
		logger.Error("查询数据权重文件名失败：数据库查询失败", "id", id, "error", err)
		return "", err
	}

	weightName := strings.TrimSpace(row.WeightName)
	if weightName == "" {
		logger.Warn("查询数据权重文件名为空", "id", id)
		return "", ErrNilEntity
	}
	logger.Info("查询数据权重文件名成功", "id", id, "weight_name", weightName)
	return weightName, nil
}

// FindFileNameByID 兼容旧调用，内部映射到 weight_name。
func (d *ModelDAO) FindFileNameByID(ctx context.Context, id uint) (string, error) {
	return d.FindWeightNameByID(ctx, id)
}

// UpdateWeightSizeByWeightName 按权重文件名更新数据文件大小（MB）。
func (d *ModelDAO) UpdateWeightSizeByWeightName(ctx context.Context, weightName string, weightSizeMB float64) (int64, error) {
	logger := daoLogger().With("func", "UpdateWeightSizeByWeightName")

	name := strings.TrimSpace(filepath.Base(weightName))
	if name == "" || name == "." || name == string(filepath.Separator) {
		logger.Warn("更新数据权重大小已跳过：权重文件名无效", "weight_name", weightName)
		return 0, ErrNilEntity
	}
	if weightSizeMB < 0 {
		logger.Warn("更新数据权重大小已跳过：权重大小MB无效", "weight_name", name, "weight_size_mb", weightSizeMB)
		return 0, ErrNilEntity
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("更新数据权重大小失败：绑定上下文失败", "weight_name", name, "error", err)
		return 0, fmt.Errorf("更新数据权重大小失败: %w", err)
	}

	result := dbConn.Model(&entity2.Model{}).Where("weight_name = ?", name).Update("weight_size_mb", weightSizeMB)
	if result.Error != nil {
		logger.Error("更新数据权重大小失败：数据库更新失败", "weight_name", name, "weight_size_mb", weightSizeMB, "error", result.Error)
		return 0, fmt.Errorf("更新数据权重大小失败: %w", result.Error)
	}

	logger.Info("更新数据权重大小成功", "weight_name", name, "weight_size_mb", weightSizeMB, "rows_affected", result.RowsAffected)
	return result.RowsAffected, nil
}

// DeleteByID 根据主键删除数据记录。
func (d *ModelDAO) DeleteByID(ctx context.Context, id uint) error {
	logger := daoLogger().With("func", "DeleteByID")
	if id == 0 {
		logger.Warn("删除数据已跳过：ID无效", "id", id)
		return ErrInvalidID
	}
	logger.Info("删除数据开始", "id", id)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("删除数据失败：绑定上下文失败", "id", id, "error", err)
		return fmt.Errorf("按 ID 删除数据失败: %w", err)
	}

	result := dbConn.Delete(&entity2.Model{}, id)
	if result.Error != nil {
		logger.Error("删除数据失败：数据库删除失败", "id", id, "error", result.Error)
		return fmt.Errorf("按 ID 删除数据失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		logger.Warn("删除数据未找到记录", "id", id)
		return gorm.ErrRecordNotFound
	}

	logger.Info("删除数据成功", "id", id)
	return nil
}

// DeleteByWeightName 根据权重文件名删除数据记录。
func (d *ModelDAO) DeleteByWeightName(ctx context.Context, weightName string) (int64, error) {
	logger := daoLogger().With("func", "DeleteByWeightName")

	name := strings.TrimSpace(filepath.Base(weightName))
	if name == "" || name == "." || name == string(filepath.Separator) {
		logger.Warn("按权重文件名删除数据已跳过：权重文件名无效", "weight_name", weightName)
		return 0, ErrNilEntity
	}
	logger.Info("按权重文件名删除数据开始", "weight_name", name)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("按权重文件名删除数据失败：绑定上下文失败", "weight_name", name, "error", err)
		return 0, fmt.Errorf("按 weight_name 删除数据失败: %w", err)
	}

	result := dbConn.Where("weight_name = ?", name).Delete(&entity2.Model{})
	if result.Error != nil {
		logger.Error("按权重文件名删除数据失败：数据库删除失败", "weight_name", name, "error", result.Error)
		return 0, fmt.Errorf("按 weight_name 删除数据失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		logger.Warn("按权重文件名删除数据未找到记录", "weight_name", name)
		return 0, gorm.ErrRecordNotFound
	}

	logger.Info("按权重文件名删除数据成功", "weight_name", name, "rows_affected", result.RowsAffected)
	return result.RowsAffected, nil
}

// FindByID 根据主键查询单条数据记录。
func (d *ModelDAO) FindByID(ctx context.Context, id uint) (*entity2.Model, error) {
	logger := daoLogger().With("func", "FindByID")
	if id == 0 {
		logger.Warn("按ID查询数据已跳过：ID无效", "id", id)
		return nil, ErrInvalidID
	}
	logger.Info("按ID查询数据开始", "id", id)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("按ID查询数据失败：绑定上下文失败", "id", id, "error", err)
		return nil, fmt.Errorf("按 ID 查询数据失败: %w", err)
	}

	var model entity2.Model
	err = dbConn.First(&model, id).Error
	if err != nil {
		logger.Error("按ID查询数据失败：数据库查询失败", "id", id, "error", err)
		return &model, err
	}
	logger.Info("按ID查询数据成功", "id", model.ID, "name", model.Name)
	return &model, err
}

// FindByName 根据名称查询单条数据记录。
func (d *ModelDAO) FindByName(ctx context.Context, name string) (*entity2.Model, error) {
	logger := daoLogger().With("func", "FindByName")
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		logger.Warn("按名称查询数据已跳过：名称为空")
		return nil, ErrNilEntity
	}
	logger.Info("按名称查询数据开始", "name", trimmed)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("按名称查询数据失败：绑定上下文失败", "name", trimmed, "error", err)
		return nil, fmt.Errorf("按名称查询数据失败: %w", err)
	}

	var model entity2.Model
	err = dbConn.Where("name = ?", trimmed).Order("version DESC, id DESC").Take(&model).Error
	if err != nil {
		logger.Error("按名称查询数据失败：数据库查询失败", "name", trimmed, "error", err)
		return nil, err
	}

	logger.Info("按名称查询数据成功", "id", model.ID, "name", model.Name)
	return &model, nil
}

// FindAll 按查询参数分页获取数据列表与总数。
func (d *ModelDAO) FindAll(ctx context.Context, params entity2.QueryParams) ([]entity2.Model, int64, error) {
	logger := daoLogger().With("func", "FindAll")
	var models []entity2.Model
	var total int64
	logger.Info("分页查询数据开始",
		"page", params.Page,
		"page_size", params.PageSize,
		"name", params.Name,
		"keyword", params.Keyword,
		"storage_server", params.StorageServer,
		"task_type", params.TaskType,
		"algorithm_id", params.AlgorithmID,
		"framework", params.Framework,
	)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("分页查询数据失败：绑定上下文失败", "error", err)
		return nil, 0, fmt.Errorf("查询数据列表失败: %w", err)
	}

	dbConn = dbConn.Model(&entity2.Model{})

	// 1. 基础模糊搜索
	if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
		dbConn = dbConn.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 2. 指标组合过滤
	if name := strings.TrimSpace(params.Name); name != "" {
		dbConn = dbConn.Where("name = ?", name)
	}

	if storageServer := strings.TrimSpace(params.StorageServer); storageServer != "" {
		dbConn = dbConn.Where(
			"(storage_server = ?) OR (JSON_VALID(storage_server) AND JSON_CONTAINS(storage_server, JSON_QUOTE(?)))",
			storageServer,
			storageServer,
		)
	}
	if taskType := strings.TrimSpace(params.TaskType); taskType != "" {
		dbConn = dbConn.Where("task_type = ?", taskType)
	}
	algorithmID := strings.TrimSpace(params.AlgorithmID)
	if algorithmID == "" {
		algorithmID = strings.TrimSpace(params.ImplType)
	}
	if algorithmID == "" {
		algorithmID = strings.TrimSpace(params.Algorithm)
	}
	if algorithmID != "" {
		dbConn = dbConn.Where("algorithm_id = ?", algorithmID)
	}
	if framework := strings.TrimSpace(params.Framework); framework != "" {
		dbConn = dbConn.Where("framework = ?", framework)
	}
	if version := strings.TrimSpace(params.Version); version != "" {
		dbConn = dbConn.Where("version = ?", version)
	}
	if params.BaseModelID != nil {
		dbConn = dbConn.Where("base_model_id = ?", *params.BaseModelID)
	}

	// 兼容旧参数（旧表字段已删除），仅记录日志并忽略。
	if params.DatasetID != nil || params.TrainTaskID != nil {
		logger.Warn("新数据结构已忽略旧筛选条件",
			"dataset_id_set", params.DatasetID != nil,
			"train_task_id_set", params.TrainTaskID != nil,
		)
	}

	// 3. 排序规则 (根据 weight_size_mb)
	orderStr := "id DESC" // 默认按 ID 降序
	sortValue := strings.ToLower(strings.TrimSpace(params.SizeSort))
	if sortValue == "" {
		sortValue = strings.ToLower(strings.TrimSpace(params.WeightSort))
	}
	switch sortValue {
	case "asc":
		orderStr = "weight_size_mb ASC"
	case "desc":
		orderStr = "weight_size_mb DESC"
	}

	// 4. 获取总数
	err = dbConn.Count(&total).Error
	if err != nil {
		logger.Error("统计数据总数失败", "error", err)
		return nil, 0, fmt.Errorf("统计数据数量失败: %w", err)
	}

	// 5. 执行分页查询
	offset, limit := pagination(params)
	err = dbConn.Order(orderStr).Offset(offset).Limit(limit).Find(&models).Error
	if err != nil {
		logger.Error("查询数据列表失败", "error", err)
		return nil, 0, fmt.Errorf("执行数据查询失败: %w", err)
	}

	logger.Info("分页查询数据成功", "total", total, "returned", len(models))
	return models, total, err
}

// UpdateMetadataByID 按主键更新数据元信息，updates 仅包含允许更新的字段。
func (d *ModelDAO) UpdateMetadataByID(ctx context.Context, id uint, updates map[string]interface{}) (*entity2.Model, error) {
	logger := daoLogger().With("func", "UpdateMetadataByID")
	if id == 0 {
		logger.Warn("更新数据元数据已跳过：ID无效", "id", id)
		return nil, ErrInvalidID
	}
	if len(updates) == 0 {
		logger.Warn("更新数据元数据已跳过：更新字段为空", "id", id)
		return nil, ErrNilEntity
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("更新数据元数据失败：绑定上下文失败", "id", id, "error", err)
		return nil, fmt.Errorf("更新数据元数据失败: %w", err)
	}

	var current entity2.Model
	if err := dbConn.First(&current, id).Error; err != nil {
		logger.Error("更新数据元数据失败：查询当前记录失败", "id", id, "error", err)
		return nil, err
	}

	result := dbConn.Model(&entity2.Model{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		logger.Error("更新数据元数据失败：数据库更新失败", "id", id, "error", result.Error)
		if isDuplicateKeyError(result.Error) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("更新数据元数据失败: %w", result.Error)
	}

	var updated entity2.Model
	if err := dbConn.First(&updated, id).Error; err != nil {
		logger.Error("更新数据元数据失败：回查失败", "id", id, "error", err)
		return nil, err
	}

	logger.Info("更新数据元数据成功", "id", id, "updated_fields", len(updates))
	return &updated, nil
}

// ======================= 辅助函数 ============================

// deriveWeightName 从多个可能的来源字段中推导出一个合法的权重文件名。
//
// 处理顺序：
//  1. 优先使用 weightName（新字段）
//  2. 若无效，则使用 legacyFileName（旧文件名字段）
//  3. 若仍无效，则从 legacyModelPath（旧路径字段）中解析文件名
//
// 清洗逻辑包括：
//   - 去除首尾空白字符
//   - 统一路径分隔符（兼容 Windows 与 Linux）
//   - 仅保留路径中的文件名部分（防止路径污染）
//   - 过滤非法结果（"", ".", "/" 等）
//
// 返回值：
//   - string：合法的权重文件名
//   - error：若无法推导出合法文件名则返回错误
func deriveWeightName(
	weightName string,
	legacyFileName string,
	legacyModelPath string,
) (string, error) {
	name := strings.TrimSpace(filepath.Base(weightName))
	if isInvalidFileName(name) {
		// -------- Step 2: 尝试旧文件名字段 --------
		legacy := strings.TrimSpace(legacyFileName)
		if legacy != "" {
			name = strings.TrimSpace(filepath.Base(legacy))
		}
	}

	if isInvalidFileName(name) {
		// -------- Step 3: 从旧路径中提取 --------
		// 统一 Windows 路径分隔符为 "/"
		cleanPath := strings.TrimSpace(
			strings.ReplaceAll(legacyModelPath, "\\", "/"),
		)

		if cleanPath != "" {
			derived := strings.TrimSpace(filepath.Base(cleanPath))
			if !isInvalidFileName(derived) {
				name = derived
			}
		}
	}
	logger := daoLogger().With("func", "deriveWeightName")
	logger.Info("推导权重文件名结束", "weight_name", weightName, "legacy_file_name", legacyFileName, "legacy_model_path", legacyModelPath, "derived_name", name)
	// -------- Final 校验 --------
	if isInvalidFileName(name) {
		return "", ErrNilEntity
	}
	return name, nil
}

func isInvalidFileName(name string) bool {
	return name == "" ||
		name == "." ||
		name == string(filepath.Separator)
}
