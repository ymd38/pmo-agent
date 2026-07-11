package main

import (
	"context"
	"log"

	"pmo-agent/api/internal/config"
	"pmo-agent/api/internal/di"
	"pmo-agent/api/internal/usecase"

	"github.com/gin-gonic/gin"
)

func main() {
	c, err := di.BuildContainer()
	if err != nil {
		log.Fatalf("DIコンテナの構築に失敗: %v", err)
	}
	// 起動時に期限切れトークンを一度掃除する（テーブルの無限増加を防ぐ低頻度メンテナンス）。
	// 失敗しても起動は継続する（クリーンアップはベストエフォート）。
	if err := c.Invoke(func(auth *usecase.AuthUsecase) {
		refN, setN, err := auth.CleanupExpiredTokens(context.Background())
		if err != nil {
			log.Printf("期限切れトークンのクリーンアップに失敗（起動は継続）: %v", err)
			return
		}
		log.Printf("期限切れトークンを削除しました: refresh=%d, set=%d", refN, setN)
	}); err != nil {
		log.Fatalf("トークンクリーンアップの実行に失敗: %v", err)
	}
	if err := c.Invoke(func(e *gin.Engine, cfg config.Config) error {
		log.Printf("PMO Agent API を :%s で起動します", cfg.Port)
		return e.Run(":" + cfg.Port)
	}); err != nil {
		log.Fatalf("サーバー起動に失敗: %v", err)
	}
}
