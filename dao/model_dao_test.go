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
	return &entity.Model{
		Name:          fmt.Sprintf("unittest_model_%d", time.Now().UnixNano()),
		StorageServer: "nas-01",
		ModelPath:     "/test/path/model.weights",
		ImplType:      "yolo_ultralytics",
		DatasetID:     1,
		SizeMB:        100.500,
		Version:       "v1.0.0",
		TaskType:      "detect",
		CreatedAt:     time.Now(),
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
	model2.ModelPath = "/test/path/model_v2.weights"
	model2.SizeMB = 200.25

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
	assert.Equal(t, model2.ModelPath, got.ModelPath)
	assert.Equal(t, model2.SizeMB, got.SizeMB)
}
