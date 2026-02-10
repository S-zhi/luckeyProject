package dao_test

import (
	"context"
	"log"
	"lucky_project/config"
	"lucky_project/internal/dao"
	"lucky_project/internal/entity"
	"lucky_project/pkg/db"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// 切换到项目根目录读取配置
	os.Chdir("../../")

	// 初始化配置
	if err := config.InitConfig(); err != nil {
		panic(err)
	}

	// 初始化数据库
	if err := db.InitDB(); err != nil {
		panic(err)
	}

	// 运行测试
	code := m.Run()
	os.Exit(code)
}

func TestModelDAOSave(t *testing.T) {
	// 创建 DAO 实例
	modelDAO := dao.NewModelDAO()

	t.Run("正常保存模型", func(t *testing.T) {
		// 创建测试模型数据
		model := &entity.Model{
			ModelName:    "unittest_model_" + time.Now().Format("20060102150405"),
			ModelType:    1,
			ModelVersion: 1.0,
			IsLatest:     true,
			IsBasicModel: false,
			Algorithm:    "YOLOv8",
			Framework:    "PyTorch",
			WeightSizeMB: 100.500,
			WeightPath:   "/test/path/model.weights",
			DatasetID:    1,
			CreateTime:   time.Now(),
		}

		// 执行保存操作
		err := modelDAO.Save(context.Background(), model)
		log.Println("ModelDAO.Save err:", err)
		// 验证结果
		//assert.NoError(t, err, "保存模型应该成功")
		assert.NotZero(t, model.ID, "模型ID应该被自动分配")

		// 如果保存成功，清理测试数据
		if model.ID > 0 && modelDAO.DB != nil {
			modelDAO.DB.Delete(&entity.Model{}, model.ID)
		}
	})

	//t.Run("保存带可选字段的模型", func(t *testing.T) {
	//	model := &entity.Model{
	//		ModelName:    "unittest_model_optional_" + time.Now().Format("20060102150405"),
	//		ModelType:    2,
	//		ModelVersion: 2.0,
	//		IsLatest:     true,
	//		IsBasicModel: true,
	//		Algorithm:    "ResNet",
	//		Framework:    "TensorFlow",
	//		WeightSizeMB: 200.750,
	//		WeightPath:   "/test/path/model2.weights",
	//		DatasetID:    2,
	//		CreateTime:   time.Now(),
	//	}
	//
	//	err := modelDAO.Save(model)
	//	assert.NoError(t, err, "保存带可选字段的模型应该成功")
	//	assert.NotZero(t, model.ID, "模型ID应该被自动分配")
	//
	//	// 清理测试数据
	//	if model.ID > 0 && modelDAO.DB != nil {
	//		modelDAO.DB.Delete(&entity.Model{}, model.ID)
	//	}
	//})
	//
	//t.Run("保存模型名称唯一性约束", func(t *testing.T) {
	//	// 创建基础模型
	//	uniqueName := "unique_test_model_" + time.Now().Format("20060102150405")
	//	baseModel := &entity.Model{
	//		ModelName:    uniqueName,
	//		ModelType:    3,
	//		ModelVersion: 1.0,
	//		IsLatest:     true,
	//		IsBasicModel: false,
	//		Algorithm:    "ResNet",
	//		Framework:    "PyTorch",
	//		WeightSizeMB: 50.250,
	//		WeightPath:   "/test/path/base.weights",
	//		DatasetID:    3,
	//		CreateTime:   time.Now(),
	//	}
	//
	//	// 保存第一个模型
	//	err := modelDAO.Save(baseModel)
	//	assert.NoError(t, err, "第一次保存应该成功")
	//
	//	// 尝试保存同名模型
	//	duplicateModel := &entity.Model{
	//		ModelName:    uniqueName,
	//		ModelType:    3,
	//		ModelVersion: 1.1,
	//		IsLatest:     false,
	//		IsBasicModel: false,
	//		Algorithm:    "ResNet",
	//		Framework:    "PyTorch",
	//		WeightSizeMB: 60.300,
	//		WeightPath:   "/test/path/duplicate.weights",
	//		DatasetID:    4,
	//		CreateTime:   time.Now(),
	//	}
	//
	//	err = modelDAO.Save(duplicateModel)
	//	assert.Error(t, err, "保存重复名称的模型应该失败")
	//
	//	// 清理测试数据
	//	if baseModel.ID > 0 && modelDAO.DB != nil {
	//		modelDAO.DB.Delete(&entity.Model{}, baseModel.ID)
	//	}
	//})
}
