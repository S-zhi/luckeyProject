package dao_test

import (
	"context"
	"errors"
	"fmt"
	"lucky_project/config"
	"lucky_project/dao"
	"lucky_project/entity"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	// 切换到项目根目录读取配置
	os.Chdir("..")

	// 初始化配置
	if err := config.InitConfig(); err != nil {
		panic(err)
	}

	// 初始化数据库
	if err := config.InitDB(); err != nil {
		panic(err)
	}

	// 运行测试
	code := m.Run()
	os.Exit(code)
}

func newTestModel() *entity.Model {
	algorithmID := "yolo_ultralytics"
	framework := "pytorch"
	return &entity.Model{
		Name:          fmt.Sprintf("unittest_model_%d", time.Now().UnixNano()),
		Version:       1.00,
		BaseModelID:   0,
		AlgorithmID:   &algorithmID,
		TaskType:      "detect",
		Framework:     &framework,
		WeightSizeMB:  100.500,
		StorageServer: "nas-01",
		WeightName:    "model.weights",
	}
}

func TestModelDAOSave(t *testing.T) {
	modelDAO := dao.NewModelDAO()
	model := newTestModel()

	err := modelDAO.Save(context.Background(), model)
	assert.NoError(t, err, "save should succeed")
	assert.NotZero(t, model.ID, "model id should be generated")

	//	t.Cleanup(func() {
	//	if model.ID > 0 && modelDAO.DB != nil {
	//		_ = modelDAO.DB.Delete(&entity.Model{}, model.ID).Error
	//	}
	//})
}

func TestModelDAODeleteByID(t *testing.T) {
	modelDAO := dao.NewModelDAO()
	model := newTestModel()

	err := modelDAO.Save(context.Background(), model)
	assert.NoError(t, err, "setup save should succeed")
	assert.NotZero(t, model.ID, "setup model id should be generated")

	t.Cleanup(func() {
		if model.ID > 0 && modelDAO.DB != nil {
			_ = modelDAO.DB.Delete(&entity.Model{}, model.ID).Error
		}
	})

	err = modelDAO.DeleteByID(context.Background(), model.ID)
	assert.NoError(t, err, "delete should succeed")

	_, err = modelDAO.FindByID(context.Background(), model.ID)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound), "find deleted record should return record not found")
}

func TestModelDAODeleteByIDInvalidID(t *testing.T) {
	modelDAO := dao.NewModelDAO()

	err := modelDAO.DeleteByID(context.Background(), 0)
	assert.True(t, errors.Is(err, dao.ErrInvalidID), "id=0 should return ErrInvalidID")
}

func TestModelDAOSaveNilEntity(t *testing.T) {
	modelDAO := dao.NewModelDAO()

	err := modelDAO.Save(context.Background(), nil)
	assert.True(t, errors.Is(err, dao.ErrNilEntity), "nil entity should return ErrNilEntity")
}

func TestModelDAOSaveDuplicateName(t *testing.T) {
	modelDAO := dao.NewModelDAO()

	baseName := fmt.Sprintf("unittest_dup_model_%d", time.Now().UnixNano())
	model1 := newTestModel()
	model1.Name = baseName
	model2 := newTestModel()
	model2.Name = baseName
	model2.StorageServer = "nas-02"
	model2.WeightName = "model_v2.weights"
	model2.WeightSizeMB = 200.25

	err := modelDAO.Save(context.Background(), model1)
	assert.NoError(t, err, "first save should succeed")

	t.Cleanup(func() {
		if model1.ID > 0 && modelDAO.DB != nil {
			_ = modelDAO.DB.Delete(&entity.Model{}, model1.ID).Error
		}
		if model2.ID > 0 && modelDAO.DB != nil {
			_ = modelDAO.DB.Delete(&entity.Model{}, model2.ID).Error
		}
	})

	err = modelDAO.Save(context.Background(), model2)
	assert.NoError(t, err, "duplicate name should upsert instead of returning duplicate error")
	assert.Equal(t, model1.ID, model2.ID, "upsert should keep same primary key")

	got, err := modelDAO.FindByID(context.Background(), model1.ID)
	assert.NoError(t, err)
	assert.Equal(t, model2.StorageServer, got.StorageServer)
	assert.Equal(t, model2.WeightName, got.WeightName)
	assert.Equal(t, model2.WeightSizeMB, got.WeightSizeMB)
}

func TestModelDAOUpdateMetadataByID(t *testing.T) {
	modelDAO := dao.NewModelDAO()
	model := newTestModel()

	err := modelDAO.Save(context.Background(), model)
	assert.NoError(t, err, "setup save should succeed")
	assert.NotZero(t, model.ID, "setup model id should be generated")

	t.Cleanup(func() {
		if model.ID > 0 && modelDAO.DB != nil {
			_ = modelDAO.DB.Delete(&entity.Model{}, model.ID).Error
		}
	})

	updatedModel, err := modelDAO.UpdateMetadataByID(context.Background(), model.ID, map[string]interface{}{
		"name":           model.Name + "_updated",
		"version":        1.10,
		"weight_name":    "updated_weight.pt",
		"weight_size_mb": 222.333,
	})
	assert.NoError(t, err, "update metadata should succeed")
	assert.NotNil(t, updatedModel)
	assert.Equal(t, model.ID, updatedModel.ID)
	assert.Equal(t, model.Name+"_updated", updatedModel.Name)
	assert.InDelta(t, 1.10, updatedModel.Version, 0.0001)
	assert.Equal(t, "updated_weight.pt", updatedModel.WeightName)
	assert.InDelta(t, 222.333, updatedModel.WeightSizeMB, 0.0001)
}

func TestModelDAOUpdateMetadataByIDInvalidID(t *testing.T) {
	modelDAO := dao.NewModelDAO()
	_, err := modelDAO.UpdateMetadataByID(context.Background(), 0, map[string]interface{}{
		"name": "invalid",
	})
	assert.True(t, errors.Is(err, dao.ErrInvalidID), "id=0 should return ErrInvalidID")
}
