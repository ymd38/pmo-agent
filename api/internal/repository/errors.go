package repository

import (
	"errors"

	"pmo-agent/api/internal/domain"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// mysqlDuplicateEntry は MySQL の重複キーエラーコード（ER_DUP_ENTRY）。
const mysqlDuplicateEntry = 1062

// wrapNotFound は GORM の未検出エラーをドメインエラーへ写像する。
func wrapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}
	return err
}

// wrapConflict は一意制約違反（採番の同時実行やコード・メールの重複）を
// domain.ErrConflict へ写像する。handler はこれを 409 として返す。
func wrapConflict(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return domain.ErrConflict
	}
	var me *mysql.MySQLError
	if errors.As(err, &me) && me.Number == mysqlDuplicateEntry {
		return domain.ErrConflict
	}
	return err
}
