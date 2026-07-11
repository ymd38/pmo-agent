package infra

import (
	"fmt"
	"log"
	"time"

	"pmo-agent/api/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// pingMaxRetries と pingBaseBackoff は起動時 ping のリトライ挙動を決める。
// docker-compose で MySQL より API が先に起動しても接続確立を待てるようにする
// （回数上限つきの単純な線形バックオフ。過剰な汎用化はしない）。
const (
	pingMaxRetries  = 10
	pingBaseBackoff = 500 * time.Millisecond
)

// NewDB は GORM の MySQL 接続を確立し、コネクションプールを設定する。
// マイグレーションは golang-migrate で管理するため AutoMigrate は呼ばない。
func NewDB(cfg config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("infra: DB接続のオープンに失敗: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("infra: *sql.DB の取得に失敗: %w", err)
	}

	// コネクションプール設定。無制限接続による max_connections 枯渇と、
	// wait_timeout 経過後の死んだ接続の再利用（invalid connection）を防ぐ。
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	// 起動時 ping で接続を検証する。MySQL の起動待ちに備えバックオフ再試行する。
	if err := pingWithRetry(sqlDB.Ping, pingMaxRetries, pingBaseBackoff); err != nil {
		return nil, fmt.Errorf("infra: DB への ping に失敗: %w", err)
	}

	return db, nil
}

// pingWithRetry は ping が成功するまで maxRetries 回まで線形バックオフで再試行する。
// 全て失敗した場合は最後のエラーを返す。
func pingWithRetry(ping func() error, maxRetries int, baseBackoff time.Duration) error {
	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err = ping(); err == nil {
			return nil
		}
		if attempt < maxRetries {
			backoff := time.Duration(attempt) * baseBackoff
			log.Printf("infra: DB ping 失敗（%d/%d）: %v。%s 後に再試行します", attempt, maxRetries, err, backoff)
			time.Sleep(backoff)
		}
	}
	return err
}
