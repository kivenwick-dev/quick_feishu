package db

import (
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"quick-feishu/internal/model"
)

func Init(dataDir string) (*gorm.DB, error) {
	path := filepath.Join(dataDir, "quick-feishu.db")
	gdb, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
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
