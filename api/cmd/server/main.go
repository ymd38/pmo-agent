package main

import (
	"log"

	"pmo-agent/api/internal/config"
	"pmo-agent/api/internal/di"

	"github.com/gin-gonic/gin"
)

func main() {
	c, err := di.BuildContainer()
	if err != nil {
		log.Fatalf("DIコンテナの構築に失敗: %v", err)
	}
	if err := c.Invoke(func(e *gin.Engine, cfg config.Config) error {
		log.Printf("PMO Agent API を :%s で起動します", cfg.Port)
		return e.Run(":" + cfg.Port)
	}); err != nil {
		log.Fatalf("サーバー起動に失敗: %v", err)
	}
}
