package repository

import (
	"errors"
	"fmt"
	"testing"

	"pmo-agent/api/internal/domain"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestWrapConflict(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "MySQLの重複キーエラー(1062)はErrConflictに写像する",
			err:  &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"},
			want: domain.ErrConflict,
		},
		{
			name: "GORMのErrDuplicatedKeyもErrConflictに写像する",
			err:  gorm.ErrDuplicatedKey,
			want: domain.ErrConflict,
		},
		{
			name: "ラップされた重複キーエラーも検出する",
			err:  fmt.Errorf("insert: %w", &mysql.MySQLError{Number: 1062}),
			want: domain.ErrConflict,
		},
		{
			name: "無関係なエラーはそのまま返す",
			err:  gorm.ErrInvalidData,
			want: gorm.ErrInvalidData,
		},
		{
			name: "nilはnilのまま",
			err:  nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapConflict(tt.err)
			if tt.want == nil {
				assert.NoError(t, got)
				return
			}
			assert.True(t, errors.Is(got, tt.want), "got=%v want=%v", got, tt.want)
		})
	}
}
