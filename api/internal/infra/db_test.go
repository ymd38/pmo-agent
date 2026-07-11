package infra

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPingWithRetry は起動時 ping の再試行ロジックを検証する。
// 実 DB 接続は不要で、ping の成否を制御するフェイク関数で振る舞いを確認する。
func TestPingWithRetry(t *testing.T) {
	errPing := errors.New("connection refused")

	tests := []struct {
		name         string
		failCount    int // 何回目まで失敗させるか（それ以降は成功）
		maxRetries   int
		wantErr      bool
		wantAttempts int
	}{
		{
			name:         "初回で成功すれば再試行しない",
			failCount:    0,
			maxRetries:   3,
			wantAttempts: 1,
		},
		{
			name:         "失敗後に成功すれば nil を返す",
			failCount:    2,
			maxRetries:   5,
			wantAttempts: 3,
		},
		{
			name:         "上限まで失敗したら最後のエラーを返す",
			failCount:    10,
			maxRetries:   3,
			wantErr:      true,
			wantAttempts: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			ping := func() error {
				attempts++
				if attempts <= tt.failCount {
					return errPing
				}
				return nil
			}

			// バックオフはテストを速く保つため極小値にする。
			err := pingWithRetry(ping, tt.maxRetries, time.Millisecond)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantAttempts, attempts)
		})
	}
}
