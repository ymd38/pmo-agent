package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		wantErr    bool
		wantSecure bool
	}{
		{
			name:    "JWT_SECRET未設定はエラー（既知のデフォルト値へのフォールバック禁止）",
			env:     map[string]string{"JWT_SECRET": ""},
			wantErr: true,
		},
		{
			name:       "JWT_SECRET設定済みなら読み込める。COOKIE_SECUREの既定はtrue",
			env:        map[string]string{"JWT_SECRET": "test-secret"},
			wantSecure: true,
		},
		{
			name:       "COOKIE_SECURE=falseで上書きできる（ローカルHTTP開発用）",
			env:        map[string]string{"JWT_SECRET": "test-secret", "COOKIE_SECURE": "false"},
			wantSecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg, err := Load()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "test-secret", cfg.JWTSecret)
			assert.Equal(t, tt.wantSecure, cfg.CookieSecure)
		})
	}
}

func TestLoad_DBPool(t *testing.T) {
	tests := []struct {
		name         string
		env          map[string]string
		wantErr      bool
		wantOpen     int
		wantIdle     int
		wantLifetime time.Duration
	}{
		{
			name:         "未設定なら安全な既定値（open=25 / idle=25 / lifetime=5m）",
			env:          map[string]string{},
			wantOpen:     25,
			wantIdle:     25,
			wantLifetime: 5 * time.Minute,
		},
		{
			name: "環境変数で上書きできる",
			env: map[string]string{
				"DB_MAX_OPEN_CONNS":    "50",
				"DB_MAX_IDLE_CONNS":    "10",
				"DB_CONN_MAX_LIFETIME": "1h30m",
			},
			wantOpen:     50,
			wantIdle:     10,
			wantLifetime: 90 * time.Minute,
		},
		{
			name:    "DB_MAX_OPEN_CONNS が非整数ならエラー",
			env:     map[string]string{"DB_MAX_OPEN_CONNS": "abc"},
			wantErr: true,
		},
		{
			name:    "DB_MAX_IDLE_CONNS が非整数ならエラー",
			env:     map[string]string{"DB_MAX_IDLE_CONNS": "1.5"},
			wantErr: true,
		},
		{
			name:    "DB_CONN_MAX_LIFETIME が不正な継続時間ならエラー",
			env:     map[string]string{"DB_CONN_MAX_LIFETIME": "5minutes"},
			wantErr: true,
		},
		// 負数は database/sql が「無制限・無期限」と解釈するため設定ミスとして拒否する。
		{
			name:    "DB_MAX_OPEN_CONNS が負数ならエラー（黙って無制限にしない）",
			env:     map[string]string{"DB_MAX_OPEN_CONNS": "-1"},
			wantErr: true,
		},
		{
			name:    "DB_MAX_IDLE_CONNS が負数ならエラー",
			env:     map[string]string{"DB_MAX_IDLE_CONNS": "-5"},
			wantErr: true,
		},
		{
			name:    "DB_CONN_MAX_LIFETIME が負の継続時間ならエラー（黙って無期限にしない）",
			env:     map[string]string{"DB_CONN_MAX_LIFETIME": "-5m"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", "test-secret")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg, err := Load()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOpen, cfg.DBMaxOpenConns)
			assert.Equal(t, tt.wantIdle, cfg.DBMaxIdleConns)
			assert.Equal(t, tt.wantLifetime, cfg.DBConnMaxLifetime)
		})
	}
}

func TestLoad_AuthRateLimit(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		wantErr    bool
		wantPerMin int
		wantBurst  int
	}{
		{
			name:       "未設定なら既定値（per_min=10 / burst=5）",
			env:        map[string]string{},
			wantPerMin: 10,
			wantBurst:  5,
		},
		{
			name: "環境変数で上書きできる",
			env: map[string]string{
				"AUTH_RATE_LIMIT_PER_MIN": "30",
				"AUTH_RATE_LIMIT_BURST":   "8",
			},
			wantPerMin: 30,
			wantBurst:  8,
		},
		{
			name:    "AUTH_RATE_LIMIT_PER_MIN が非整数ならエラー",
			env:     map[string]string{"AUTH_RATE_LIMIT_PER_MIN": "abc"},
			wantErr: true,
		},
		// 0・負数は「制限しない/不能」を意味するため設定ミスとして拒否する。
		{
			name:    "AUTH_RATE_LIMIT_PER_MIN が0ならエラー（黙って無制限にしない）",
			env:     map[string]string{"AUTH_RATE_LIMIT_PER_MIN": "0"},
			wantErr: true,
		},
		{
			name:    "AUTH_RATE_LIMIT_PER_MIN が負数ならエラー",
			env:     map[string]string{"AUTH_RATE_LIMIT_PER_MIN": "-1"},
			wantErr: true,
		},
		{
			name:    "AUTH_RATE_LIMIT_BURST が0ならエラー",
			env:     map[string]string{"AUTH_RATE_LIMIT_BURST": "0"},
			wantErr: true,
		},
		{
			name:    "AUTH_RATE_LIMIT_BURST が負数ならエラー",
			env:     map[string]string{"AUTH_RATE_LIMIT_BURST": "-3"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", "test-secret")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg, err := Load()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPerMin, cfg.AuthRateLimitPerMin)
			assert.Equal(t, tt.wantBurst, cfg.AuthRateLimitBurst)
		})
	}
}
