package db

import (
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"quick-feishu/internal/model"
)

// Path 返回账号对应的数据库文件路径。userID 为空时使用旧的单库路径。
func Path(dataDir, userID string) string {
	if userID == "" {
		return filepath.Join(dataDir, "quick-feishu.db")
	}
	return filepath.Join(dataDir, "accounts", userID, "quick-feishu.db")
}

// Open 打开（必要时创建父目录与文件）指定路径的数据库并迁移表结构。
func Open(path string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
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

// Init 保留旧签名，打开 dataDir 下的单库（兼容既有测试）。
func Init(dataDir string) (*gorm.DB, error) {
	return Open(Path(dataDir, ""))
}

// MigrateLegacy 首次切到某账号时，把旧单库移动到该账号目录。
// 仅当 userID 非空、旧库存在且目标库不存在时执行；不覆盖已存在的账号库。
func MigrateLegacy(dataDir, userID string) error {
	if userID == "" {
		return nil
	}
	legacy := Path(dataDir, "")
	target := Path(dataDir, userID)
	if _, err := os.Stat(legacy); err != nil {
		if os.IsNotExist(err) {
			return nil // 旧库不存在，无需迁移
		}
		return err
	}
	if _, err := os.Stat(target); err == nil {
		return nil // 目标库已存在，不覆盖
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	return os.Rename(legacy, target)
}
