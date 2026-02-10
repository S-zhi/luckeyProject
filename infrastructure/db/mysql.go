package db

import (
	"errors"
	"fmt"
	"lucky_project/config"
	entity2 "lucky_project/entity"
	"net/url"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() error {
	if config.AppConfig == nil {
		return errors.New("app config is not initialized")
	}

	cfg := config.AppConfig.DB
	if !strings.EqualFold(cfg.Driver, "mysql") {
		return fmt.Errorf("unsupported db driver: %s", cfg.Driver)
	}

	loc := url.QueryEscape("Asia/Shanghai")
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=%s&timeout=5s&readTimeout=10s&writeTimeout=10s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
		loc,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf(
			"connect mysql failed (host=%s port=%d db=%s user=%s): %w",
			cfg.Host, cfg.Port, cfg.DBName, cfg.User, err,
		)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB failed: %w", err)
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("mysql ping failed: %w", err)
	}

	if err := ensureTables(db); err != nil {
		return err
	}

	DB = db
	return nil
}

func ensureTables(db *gorm.DB) error {
	models := []interface{}{
		&entity2.Model{},
		&entity2.Dataset{},
		&entity2.ModelTrainingResult{},
	}

	for _, m := range models {
		if db.Migrator().HasTable(m) {
			continue
		}
		if err := db.AutoMigrate(m); err != nil {
			return fmt.Errorf("auto migrate missing table failed: %w", err)
		}
	}

	return nil
}
