package infra

import (
	"pmo-agent/api/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDB は GORM の MySQL 接続を確立する。
// マイグレーションは golang-migrate で管理するため AutoMigrate は呼ばない。
func NewDB(cfg config.Config) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
}
