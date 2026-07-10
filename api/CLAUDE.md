# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Go製のRESTful APIサーバー。JWT認証・ロールベースACL・Backlog連携レポートのバックエンドを担う。
全エンドポイントのプレフィックスは `/api`。

## Stack

Go 1.24.0+ / Gin / dig（DI）/ GORM / MySQL 8.0 / golang-migrate / testify

## Commands

```bash
# 開発サーバー起動（ホットリロード）
air

# ビルド
go build -o bin/server ./cmd/server

# テスト実行（全体・競合検知必須）
go test -race ./...

# テスト実行（単一パッケージ）
go test -race ./internal/usecase/...

# テスト実行（単一テスト関数）
go test -race ./internal/usecase/... -run TestProjectUsecase_IssueCode

# Lint
golangci-lint run

# マイグレーション（make経由で実行すること → ルートのMakefile参照）
make migrate-up
make migrate-down
make migrate-create name=add_projects_table
```

## Architecture（クリーンアーキテクチャ）

```
cmd/server/main.go          エントリーポイント・digコンテナ初期化
internal/
  domain/     エンティティ・値オブジェクト（DBに依存しない純粋な構造体）
  usecase/    ビジネスロジック（domainとrepositoryのインターフェースのみに依存）
  repository/ GORMによるDB実装（usecaseで定義したinterfaceを実装）
  handler/    Ginハンドラ（リクエスト解析 → usecase呼び出し → レスポンス）
  infra/      DB接続・外部連携（Backlog API 等。メール送信はフェーズ1では未使用）
  di/         digコンテナ定義（全依存グラフをここで宣言）
```

### Request Flow

```
HTTP Request
  → AuthMiddleware（JWTをAuthorizationヘッダ or Cookieから検証）
  → ACLMiddleware（role_functions テーブルで権限チェック）
  → Handler（Ginバリデーション → usecase委譲）
  → Usecase（ビジネスロジック・トランザクション・ドメインバリデーション）
  → Repository（GORM実行）
```

### Dependency Injection（dig）

- `di/` に全プロバイダを登録し、`main.go` から `di.BuildContainer()` を呼ぶだけにする
- 各層は **インターフェースに依存**し、具体実装は `di/` でバインドする
- グローバル変数によるシングルトン共有は禁止

## Key Patterns

### ACL Middleware

`role_functions` テーブルを参照して、リクエストユーザーが必要な `function.code` を持つか検証する。
スコープ制御（「担当プロジェクトのみ」等）は各ハンドラー/ユースケース内ではなく、専用のスコープミドルウェアで実装する。

```go
// middleware に渡す function コード例
RequireFunction("manage_projects")
RequireFunction("view_project_cost")
```

### Authentication（パスワード + JWT HS256）

- **パスワード認証方式**: bcrypt でハッシュ化。`POST /api/auth/login` でメール+パスワード検証 → トークン発行
- **JWT**: HS256署名（`JWT_SECRET` 環境変数）。クレームは `sub`=userID のみ（権限は埋め込まず ACLミドルウェアが毎回 DB から解決＝ロール変更を即時反映）。アクセストークン（8時間）+ リフレッシュトークン（7日・DB保管・ローテーション）の2トークン構成。`POST /api/auth/refresh` で再発行、ログアウト・ユーザー無効化時に失効
- httpOnly Cookie に格納
- **アカウント発行**: PMO管理者が `POST /api/users` で作成（`password_hash=NULL` の未アクティベート状態）→ ワンタイムトークン付きリンクを手動共有 → `POST /api/auth/set-password` で本人がパスワード設定。招待とリセットは同一トークン機構（`password_set_tokens`）。フェーズ1ではメール送信に依存しない
- **ログイン可否**: `password_hash IS NOT NULL` かつ `is_active=true` のユーザーのみ許可。OTP・Microsoft Entra ID SSO は将来の移行パス

### Project Code Issuance

- プログラム作成（`POST /api/programs`）でコードプレフィックス（例: `INV-2026-0001`）を発行する。
- `POST /api/projects/:id/issue-code` のみがプロジェクトコード（枝番付き・例: `INV-2026-0001-001`）を発行でき、同時に `planning → active` へ遷移する。枝番は `UNIQUE(program_id, branch_no)` で採番する。
- `projects.project_code` への更新は、発行済みの場合はusecase層でエラーを返す（発行後変更禁止）。

### Cost Aggregation

`GET /api/projects/:id/cost` はリクエストユーザーのロールによって返却フィールドを変える:
- `pmo_admin`: 外注コスト + 工数コスト内訳（ユーザー別単価込み）
- `pm`: 合計コストのみ（単価は非表示）

## DB

- MySQL 8.0、GORMドライバは `gorm.io/driver/mysql`
- 接続設定は環境変数（`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`）
- マイグレーション: `db/migrations/` 配下に連番管理（`golang-migrate`）。DDL初期スクリプト: `docker/mysql/init/01_schema.sql`
- **AutoMigrate は開発・本番ともに禁止**

スキーマ概要（詳細は `../docs/SPEC.md`）:

| テーブル | 用途 |
|---|---|
| `users` | 認証 + 社員情報（grade: manager/staff、`password_hash` はNULL許容＝未アクティベート） |
| `roles` / `functions` / `role_functions` | RBAC（DBで管理） |
| `password_set_tokens` / `refresh_tokens` | 招待・リセットトークン / リフレッシュトークン（共にハッシュ保存・単回/ローテーション） |
| `programs` | プログラムマスタ（PMO集計単位・コードプレフィックスの名前空間） |
| `projects` | プロジェクトマスタ（`program_id` 必須・`project_code` はUNIQUE・NULL許容） |
| `project_categories` / `project_category_values` / `project_attribute_assignments` | EAVパターンの属性管理 |
| `project_members` | プロジェクトへのメンバーアサイン |
| `grade_rates` | グレード別・年度別時間単価 |
| `work_hours` | 工数実績 |
| `project_progress` | Backlogから収集したタスク進捗 |
| `daily_reports` / `weekly_reports` / `executive_reports` | レポート（JSONカラム） |

## API Endpoints Summary

全エンドポイントと必要権限は `../docs/SPEC.md` の「API設計」セクションを参照。
認証: `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/me`, `/auth/change-password`, `/auth/set-password`
主要グループ: `/users`, `/roles`, `/programs`, `/projects`, `/categories`, `/work-hours`, `/grade-rates`, `/dashboard`, `/reports`

## Go Conventions

- DI: dig で依存性逆転を徹底。グローバル変数多用は禁止
- エラー: `errors.Is` / `errors.As` + `fmt.Errorf("usecase: %w", err)` でラップ。`panic()` は最小限
- `context.Context` は全レイヤーで第一引数として伝播
- バリデーション: Ginの `validator` タグ（handlerレイヤー）+ ドメインレイヤー両方で実施
- 構造体タグ: `json:"snake_case"` と `gorm:"column:snake_case"` を揃える
- テスト: テーブル駆動テスト + `-race` フラグ必須。DBを使うテストは testcontainers または docker-compose の test DB を使用
- 禁止: グローバル変数多用、ハードコードされた秘密情報、未使用import
