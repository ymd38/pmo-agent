package di

import (
	"testing"

	"pmo-agent/api/internal/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/dig"
)

// TestBuildContainer_GraphResolves は依存グラフが解決可能であることを検証する。
// dig.DryRun でコンストラクタを実行せずグラフのみを検査するため、DB 接続なしで
// 「プロバイダの登録漏れ」を検出できる（go build では捕捉できない実行時エラー）。
func TestBuildContainer_GraphResolves(t *testing.T) {
	c, err := buildContainer(dig.DryRun(true))
	if err != nil {
		t.Fatalf("BuildContainer: %v", err)
	}
	if err := c.Invoke(func(*gin.Engine, config.Config) {}); err != nil {
		t.Fatalf("依存グラフの解決に失敗: %v", err)
	}
}
