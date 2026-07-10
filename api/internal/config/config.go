package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config はアプリ全体の設定。値はすべて環境変数から読む（ハードコード禁止）。
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	SetTokenTTL     time.Duration

	AppBaseURL   string // パスワード設定リンクの生成に使う（例: http://localhost:3000）
	Port         string
	CookieSecure bool // 認証Cookieの Secure 属性。HTTP のローカル開発でのみ false にする
}

// Load は環境変数から設定を読み込む。未設定の項目には開発用デフォルトを当てるが、
// JWT_SECRET は署名鍵のためデフォルト値を持たず、未設定ならエラーを返す
// （公開リポジトリに既知のフォールバック値を置くとトークン偽造が可能になるため）。
func Load() (Config, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return Config{}, errors.New("環境変数 JWT_SECRET が未設定です。十分にランダムな文字列を設定してください")
	}
	return Config{
		DBHost:          env("DB_HOST", "localhost"),
		DBPort:          env("DB_PORT", "3306"),
		DBUser:          env("DB_USER", "root"),
		DBPassword:      env("DB_PASSWORD", "root"),
		DBName:          env("DB_NAME", "pmo"),
		JWTSecret:       secret,
		AccessTokenTTL:  envDuration("ACCESS_TOKEN_TTL", 8*time.Hour),
		RefreshTokenTTL: envDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		SetTokenTTL:     envDuration("SET_TOKEN_TTL", 72*time.Hour),
		AppBaseURL:      env("APP_BASE_URL", "http://localhost:3000"),
		Port:            env("PORT", "8080"),
		CookieSecure:    envBool("COOKIE_SECURE", true),
	}, nil
}

// DSN は GORM/MySQL 用のデータソース名を返す。
func (c Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
