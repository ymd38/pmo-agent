# =============================================================
# PMO Agent — 開発タスクランナー
# すべての操作はこの Makefile 経由で行う（docker / go / npm を直接叩かない）。
# 前提: Docker Desktop のみ。Go / Node のホストインストールは不要。
# =============================================================

.DEFAULT_GOAL := help

# .env を読み込む（無くてもデフォルトで動く）
-include .env
MYSQL_ROOT_PASSWORD ?= rootpassword
MYSQL_DATABASE ?= pmo_proto
email ?= admin@example.com

# コンテナ内 MySQL への接続URL（compose ネットワーク経由・ホスト名 mysql）
DB_URL_DOCKER = mysql://root:$(MYSQL_ROOT_PASSWORD)@tcp(mysql:3306)/$(MYSQL_DATABASE)

.PHONY: help env up down restart build rebuild logs api-logs web-logs ps \
        migrate-up migrate-down migrate-create migrate-force \
        seed-link db-cli reset test test-api test-web setup

# ---- ヘルプ ----
help: ## このヘルプを表示する
	@echo "PMO Agent — make targets"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(firstword $(MAKEFILE_LIST)) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

env: ## .env が無ければ .env.example から作成する
	@test -f .env && echo ".env は既に存在します" || (cp .env.example .env && echo ".env を作成しました")

# ---- ライフサイクル ----
up: ## 全サービスを起動（初回はビルド込み）: mysql / api / pmo-dashboard
	docker compose up -d

down: ## 全サービスを停止
	docker compose down

restart: ## 全サービスを再起動
	docker compose restart

build: ## イメージをビルド
	docker compose build

rebuild: ## イメージを再ビルドして起動（依存更新時）
	docker compose up -d --build

logs: ## 全サービスのログを追従表示
	docker compose logs -f

api-logs: ## API のログを追従表示
	docker compose logs -f api

web-logs: ## pmo-dashboard のログを追従表示
	docker compose logs -f pmo-dashboard

ps: ## 起動中コンテナを確認
	docker compose ps

# ---- マイグレーション ----
migrate-up: ## マイグレーションを適用
	docker compose run --rm migrate -path=/migrations -database "$(DB_URL_DOCKER)" up

migrate-down: ## マイグレーションを1件ロールバック
	docker compose run --rm migrate -path=/migrations -database "$(DB_URL_DOCKER)" down 1

migrate-force: ## 強制的にバージョンを設定（dirty 解除）: make migrate-force version=N
	docker compose run --rm migrate -path=/migrations -database "$(DB_URL_DOCKER)" force $(version)

migrate-create: ## 新規マイグレーションを作成: make migrate-create name=add_xxx
	docker compose run --rm migrate create -ext sql -dir /migrations -seq $(name)

# ---- データ / 認証 ----
seed-link: ## パスワード設定リンクを発行（既定: admin@example.com）: make seed-link email=foo@example.com
	docker compose exec api go run ./cmd/issuelink -email=$(email)

db-cli: ## MySQL CLI に接続
	docker compose exec mysql mysql -uroot -p$(MYSQL_ROOT_PASSWORD) $(MYSQL_DATABASE)

reset: ## DB を初期化して作り直す（全データ削除 → 起動 → マイグレーション）
	docker compose down -v
	docker compose up -d
	@echo "MySQL の初回初期化を待機中..."
	@for i in $$(seq 1 30); do \
		docker compose run --rm migrate -path=/migrations -database "$(DB_URL_DOCKER)" up && break; \
		echo "  DB 未準備 — リトライ $$i/30"; sleep 2; \
	done
	@echo "リセット完了。'make seed-link' で管理者リンクを発行してください。"

# ---- テスト ----
test: test-api test-web ## バックエンド・フロントエンドのテストを実行

test-api: ## Go テスト（-race）
	docker compose run --rm -e CGO_ENABLED=1 api go test -race ./...

test-web: ## フロントエンドテスト（Vitest）
	docker compose run --rm pmo-dashboard npm run test

# ---- その他 ----
setup: ## Agent Skills をインストール
	npx skills experimental_install
