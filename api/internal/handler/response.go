package handler

import (
	"errors"
	"net/http"

	"pmo-agent/api/internal/domain"
	"pmo-agent/api/internal/middleware"

	"github.com/gin-gonic/gin"
)

// respondError はドメインエラーを HTTP ステータスへ写像して JSON で返す。
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrTokenInvalid),
		errors.Is(err, domain.ErrTokenReuse):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrForbidden), errors.Is(err, domain.ErrInactiveUser), errors.Is(err, domain.ErrNotActivated):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "サーバー内部エラーが発生しました"})
	}
}

// requireScope はスコープミドルウェアが解決したプロジェクト範囲を取得する。
// ミドルウェア未適用（コンテキスト未設定）の場合は 500 を返し ok=false。
// ルーティング構成ミスによる「スコープ未適用のまま全件返却」を防ぐフェイルセーフ。
func requireScope(c *gin.Context) (domain.ProjectScope, bool) {
	scope, ok := middleware.ProjectScope(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "サーバー内部エラーが発生しました"})
		return domain.ProjectScope{}, false
	}
	return scope, true
}
