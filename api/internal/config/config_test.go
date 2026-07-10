package config

import (
	"testing"

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
