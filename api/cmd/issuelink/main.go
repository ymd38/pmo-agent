// issuelink は初期管理者やロックアウトユーザー向けに、パスワード設定用リンクを発行する。
// 使い方: go run ./cmd/issuelink -email=admin@example.com
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"pmo-agent/api/internal/config"
	"pmo-agent/api/internal/infra"
	"pmo-agent/api/internal/repository"
	"pmo-agent/api/internal/usecase"
)

func main() {
	email := flag.String("email", "", "対象ユーザーのメールアドレス")
	flag.Parse()
	if *email == "" {
		log.Fatal("usage: issuelink -email=<address>")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗: %v", err)
	}
	db, err := infra.NewDB(cfg)
	if err != nil {
		log.Fatalf("DB接続に失敗: %v", err)
	}

	userRepo := repository.NewUserRepo(db)
	uc := usecase.NewUserUsecase(
		userRepo,
		repository.NewRoleRepo(db),
		repository.NewPasswordSetTokenRepo(db),
		repository.NewRefreshTokenRepo(db),
		infra.NewTokenManager(),
		cfg.SetTokenTTL,
		cfg.AppBaseURL,
	)

	ctx := context.Background()
	user, err := userRepo.FindByEmail(ctx, *email)
	if err != nil {
		log.Fatalf("ユーザーが見つかりません (%s): %v", *email, err)
	}
	link, err := uc.ReissueLink(ctx, user.ID)
	if err != nil {
		log.Fatalf("リンク発行に失敗: %v", err)
	}

	fmt.Printf("\nパスワード設定リンク（%s 宛・%s 有効）:\n%s\n\n", *email, cfg.SetTokenTTL, link)
}
