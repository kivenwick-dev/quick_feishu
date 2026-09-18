package db

import (
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"quick-feishu/internal/model"
)

func Init(dataDir string) (*gorm.DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dataDir, "quick-feishu.db")
	gdb, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	if err := gdb.AutoMigrate(
		&model.Snapshot{},
		&model.TokenSnapshot{},
		&model.SendLog{},
		&model.DictAccountField{},
		&model.DictTokenField{},
		&model.DictUsageField{},
	); err != nil {
		return nil, err
	}
	return gdb, nil
}
