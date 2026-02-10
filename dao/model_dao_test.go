package dao_test

import (
	"context"
	"errors"
	"fmt"
	"lucky_project/config"
	"lucky_project/dao"
	"lucky_project/entity"
	"lucky_project/infrastructure/db"
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
	if err := db.InitDB(); err != nil {
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
