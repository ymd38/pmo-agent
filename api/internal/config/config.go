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

	// コネクションプール設定。無制限接続による max_connections 枯渇と、
	// wait_timeout 経過後の死んだ接続の再利用（invalid connection）を防ぐ。
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration

	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	SetTokenTTL     time.Duration

	AppBaseURL   string // パスワード設定リンクの生成に使う（例: http://localhost:3000）
	Port         string
	CookieSecure bool // 認証Cookieの Secure 属性。HTTP のローカル開発でのみ false にする

	// 認証系エンドポイントのレート制限。オンライン総当り・クレデンシャル
	// スタッフィングを抑止する。キー（クライアントIP）ごとに 1 分あたりの
	// 許容リクエスト数と、瞬間的なバースト許容数を指定する。
	AuthRateLimitPerMin int
	AuthRateLimitBurst  int
}

// Load は環境変数から設定を読み込む。未設定の項目には開発用デフォルトを当てるが、
// JWT_SECRET は署名鍵のためデフォルト値を持たず、未設定ならエラーを返す
// （公開リポジトリに既知のフォールバック値を置くとトークン偽造が可能になるため）。
func Load() (Config, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return Config{}, errors.New("環境変数 JWT_SECRET が未設定です。十分にランダムな文字列を設定してください")
	}

	// プール設定は未設定なら安全な既定値を使うが、設定された値が不正な場合は
	// 黙って既定へフォールバックせずエラーにする（設定ミスの気付きを早める）。
	// 負数は database/sql が「無制限・無期限」と解釈するため、プール制限を
	// 目的とする本設定では設定ミスとして拒否する。
	maxOpen, err := envInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return Config{}, err
	}
	if maxOpen < 0 {
		return Config{}, errors.New("環境変数 DB_MAX_OPEN_CONNS は 0 以上である必要があります")
	}
	maxIdle, err := envInt("DB_MAX_IDLE_CONNS", 25)
	if err != nil {
		return Config{}, err
	}
	if maxIdle < 0 {
		return Config{}, errors.New("環境変数 DB_MAX_IDLE_CONNS は 0 以上である必要があります")
	}
	connMaxLifetime, err := envDurationStrict("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}
	if connMaxLifetime < 0 {
		return Config{}, errors.New("環境変数 DB_CONN_MAX_LIFETIME は 0 以上である必要があります")
	}

	// レート制限は「制限」が目的のため、0・負数は無効化（=無制限）や不能状態を
	// 意味してしまう。設定ミスとして起動時に拒否する（strict validation）。
	rateLimitPerMin, err := envInt("AUTH_RATE_LIMIT_PER_MIN", 10)
	if err != nil {
		return Config{}, err
	}
	if rateLimitPerMin <= 0 {
		return Config{}, errors.New("環境変数 AUTH_RATE_LIMIT_PER_MIN は 1 以上である必要があります")
	}
	rateLimitBurst, err := envInt("AUTH_RATE_LIMIT_BURST", 5)
	if err != nil {
		return Config{}, err
	}
	if rateLimitBurst <= 0 {
		return Config{}, errors.New("環境変数 AUTH_RATE_LIMIT_BURST は 1 以上である必要があります")
	}

	return Config{
		DBHost:              env("DB_HOST", "localhost"),
		DBPort:              env("DB_PORT", "3306"),
		DBUser:              env("DB_USER", "root"),
		DBPassword:          env("DB_PASSWORD", "root"),
		DBName:              env("DB_NAME", "pmo"),
		DBMaxOpenConns:      maxOpen,
		DBMaxIdleConns:      maxIdle,
		DBConnMaxLifetime:   connMaxLifetime,
		JWTSecret:           secret,
		AccessTokenTTL:      envDuration("ACCESS_TOKEN_TTL", 8*time.Hour),
		RefreshTokenTTL:     envDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		SetTokenTTL:         envDuration("SET_TOKEN_TTL", 72*time.Hour),
		AppBaseURL:          env("APP_BASE_URL", "http://localhost:3000"),
		Port:                env("PORT", "8080"),
		CookieSecure:        envBool("COOKIE_SECURE", true),
		AuthRateLimitPerMin: rateLimitPerMin,
		AuthRateLimitBurst:  rateLimitBurst,
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

// envDurationStrict は未設定なら fallback を返し、設定済みで不正な値ならエラーを返す。
func envDurationStrict(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("環境変数 %s は継続時間（例: 5m）である必要があります: %w", key, err)
	}
	return d, nil
}

// envInt は未設定なら fallback を返し、設定済みで不正な値ならエラーを返す。
func envInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("環境変数 %s は整数である必要があります: %w", key, err)
	}
	return n, nil
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
