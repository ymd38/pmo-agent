package handler

import (
	"errors"
	"net/http"
	"strconv"

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
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "サーバー内部エラーが発生しました"})
	}
}

// pathParam は name のパスパラメータを int で取り出す。不正なら 400（msg）を返して false。
// pathID / pathValueID / pathUserID の共通実装。
func pathParam(c *gin.Context, name, msg string) (int, bool) {
	v, err := strconv.Atoi(c.Param(name))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return 0, false
	}
	return v, true
}

// bindJSON はリクエストボディを dst にバインドする。失敗時は 400（msg）を返して false。
// ShouldBindJSON+400 の定型を集約する（エラー文言は呼び出し側の文脈に合わせて渡す）。
func bindJSON(c *gin.Context, dst any, msg string) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return false
	}
	return true
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
