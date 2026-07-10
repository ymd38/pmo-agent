# PMO Agent

![CI](https://github.com/ymd38/pmo-agent/actions/workflows/ci.yml/badge.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)

プロジェクト統制基盤（PMO プラットフォーム）。プロジェクトコードの発行から工数・コスト管理、経営レポートまでを一気通貫で管理するモノレポです。

- 仕様の正本: [`docs/SPEC.md`](docs/SPEC.md)
- デザインシステム: [`DESIGN.md`](DESIGN.md)
- 開発ガイド: [`CLAUDE.md`](CLAUDE.md) / [`api/CLAUDE.md`](api/CLAUDE.md) / [`apps/pmo-dashboard/CLAUDE.md`](apps/pmo-dashboard/CLAUDE.md)

## English Overview

**PMO Agent** is a project governance platform for PMO (Project Management Office) operations — issuing project codes, tracking work hours and true project costs (external spend + internal labor), and generating executive reports, all in one place.

**Stack**: Go 1.25 (Gin / dig / GORM, clean architecture), Nuxt 4 + Tailwind CSS v4, MySQL 8.0, golang-migrate, n8n, Docker Compose, GitHub Actions.

**Highlights**:

- Role-based access control stored in the database (`roles` / `functions` / `role_functions`) — new roles require data changes, not code changes
- JWT (HS256) auth in httpOnly cookies with refresh-token rotation; invitation/reset via single-use hashed tokens
- Field-level response control by role (e.g. unit rates are visible to PMO admins only)
- Immutable project codes issued per program (`INV-2026-0001-001`), enforced at the usecase layer
- Automated daily / weekly / executive reporting via n8n workflows, with AI-generated PMO commentary (OpenAI via LangChain nodes)

**Quick start** (Docker Desktop only — no host Go/Node/MySQL needed):

```bash
make env && make up && make migrate-up && make seed-link
# open the printed set-password link, then visit http://localhost:3000
```

---

## アーキテクチャ

```
[ タスク管理ツール ] → [ Google Sheets ]
                           ↓ n8n 日次収集・AI分析
[ MySQL 8.0 ]  ←→  [ Go API (:8080) ]  ←→  [ pmo-dashboard (:3000) ]
```

| コンポーネント | 技術 | 役割 |
|---|---|---|
| `api/` | Go 1.25 / Gin / dig / GORM | RESTful API。クリーンアーキテクチャ（handler → usecase → repository） |
| `apps/pmo-dashboard/` | Nuxt 4 / TypeScript strict / Tailwind v4 | PMO管理画面（経営層・PMO管理者・PM向け） |
| `apps/worktrack/` | Nuxt 4（未実装） | 工数入力UI（全メンバー向け） |
| `db/migrations/` | golang-migrate | 連番スキーマ・seedマイグレーション |
| `n8n/workflows/` | n8n | 進捗収集・日次/週次/エグゼクティブレポート自動生成ワークフロー |

設計の特徴:

- **RBAC を DB で管理** — `roles` / `functions` / `role_functions` テーブル。新ロール・新権限はデータ追加のみで対応（開放閉鎖原則）
- **認証**: パスワード（bcrypt）+ JWT HS256 を httpOnly Cookie に格納。リフレッシュトークンはDB保管・ローテーション。招待・リセットは単回利用のハッシュ化トークン
- **ロール別フィールド制御**: 単価（`grade_rates`）は `pmo_admin` のみ参照可。コスト集計APIはロールで返却フィールドを変える
- **プロジェクトコードの不変性**: プログラム単位でプレフィックス発行（`INV-2026-0001`）、承認時に枝番発行（`-001`）。発行後の変更は usecase 層で拒否

---

## n8n ワークフロー（レポート自動生成）

[`n8n/workflows/`](n8n/workflows/) に、進捗収集とレポート生成を自動化する2つのワークフロー定義（JSON）を同梱しています。

| ワークフロー | スケジュール | 処理内容 |
|---|---|---|
| `daily_workflow.json` | 毎日10時 | 進行中プロジェクトの進捗データを Google Sheets から収集 → MySQL に保存 → 直近7日を分析して日次レポート生成 → AI（OpenAI）が PMO 視点のコメントを付与 |
| `weekly_workflow.json` | 毎週月曜9時 | 日次レポートを集約して週次サマリーを生成 → 全プロジェクト横断のエグゼクティブレポートを作成 → AI が週次・経営向けコメントを生成 |

利用手順:

1. n8n インスタンス（Cloud / self-hosted）にワークフロー JSON をインポートする
2. クレデンシャルを設定する: **MySQL**（本スタックのDB）/ **Google Sheets** / **OpenAI**
3. インポートしたワークフローをオンにする

### Google Sheets フォーマット

各プロジェクトのスプレッドシートはシート名「**プロジェクト進捗**」で、以下の列構成にします。

| 列 | 列名 | 内容 |
|---|---|---|
| A | フェーズ | 要件定義、設計、実装、テスト など |
| B | タスク | フェーズ内のタスク名 |
| C | ステータス | 未着手 / 進行中 / 完了 |
| D | 進捗 | 進捗率（0〜100） |
| E | 期限 | タスクの期限日 |

> **注意（情報取得元）**: 進捗データの取得元は環境に依存します。同梱ワークフローはタスク管理ツールのデータを Google Sheets 経由で受け取る構成のため、利用する環境に合わせて取得ノード（Google Sheets 部分）を各自のタスク管理ツール（Backlog / Jira / GitHub Issues 等）の API ノードに差し替えてください。取得後の整形・保存・AI分析のフローはそのまま流用できます。

---

## 前提

**Docker Desktop だけあれば動きます。** Go・Node・MySQL・golang-migrate のホストへのインストールは不要です。
すべての操作は `make` 経由で行います（`docker` / `go` / `npm` を直接叩く必要はありません）。

利用可能なコマンドの一覧は次で確認できます:

```bash
make help
```

---

## クイックスタート

```bash
# 1. 環境変数ファイルを用意（初回のみ）
make env

# 2. 全サービスを起動（初回はイメージビルドで数分かかります）
make up

# 3. データベースにマイグレーションを適用
make migrate-up

# 4. 初期管理者のパスワード設定リンクを発行
make seed-link
#   → 出力された http://localhost:3000/set-password?token=... を
#     ブラウザで開き、パスワードを設定する
```

設定が終わったら **http://localhost:3000** を開いてログインします。

> 初期管理者は `admin@example.com`（パスワード未設定）。
> 別アカウントのリンクが欲しい場合: `make seed-link email=<メールアドレス>`

---

## 動作確認チェックリスト

ログイン後、以下の流れで主要機能を確認できます。

| # | 操作 | 期待結果 |
|---|---|---|
| 1 | `/`（未ログイン） | 公開LPが表示される（ログインリンクあり） |
| 2 | `/login` でログイン | `/home` ダッシュボードへ遷移 |
| 3 | ナビ「プログラム」→「新規プログラム」 | 種別(例 `INV`)＋会計年度＋名称を入力 |
| 4 | プログラム作成 | コードが自動採番される（`INV-2026-0001`）。連続作成で連番が増える |
| 5 | プログラム詳細 →「新規プロジェクト」 | プロジェクトを起案（status=planning・コード未発行） |
| 6 | プロジェクトの「コード発行」 | 枝番が採番され（`INV-2026-0001-001`）status=進行中になる |
| 7 | ナビ「メンバー管理」→「新規メンバー」 | 作成すると招待リンクが表示される |
| 8 | ナビ「属性マスタ」 | カテゴリ選択 → 値の追加・無効化ができる |
| 9 | ログアウト | `/login` に戻る |

### API だけ叩いて確認する場合

```bash
# ヘルスチェック
curl http://localhost:8080/api/health        # {"status":"ok"}

# ログイン（Cookie 取得）
curl -c /tmp/c.txt -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"<設定したパスワード>"}'

# 自分の権限を確認
curl -b /tmp/c.txt http://localhost:8080/api/auth/me
```

---

## よく使う make コマンド

| コマンド | 説明 |
|---|---|
| `make up` | 全サービス起動（mysql / api / pmo-dashboard） |
| `make down` | 全サービス停止 |
| `make restart` | 再起動 |
| `make ps` | 起動中コンテナ確認 |
| `make logs` | 全ログ追従（`make api-logs` / `make web-logs` で個別） |
| `make migrate-up` / `make migrate-down` | マイグレーション適用 / 1件ロールバック |
| `make migrate-create name=add_xxx` | 新規マイグレーション作成 |
| `make seed-link [email=...]` | パスワード設定リンク発行 |
| `make db-cli` | MySQL CLI に接続 |
| `make reset` | **DBを全削除して作り直す**（クリーンな状態に戻したいとき） |
| `make test` | バックエンド(`-race`)＋フロントのテスト実行 |
| `make rebuild` | 依存更新時にイメージを再ビルドして起動 |

---

## ポート一覧

| サービス | URL / ポート | 備考 |
|---|---|---|
| pmo-dashboard | http://localhost:3000 | PMO管理画面 |
| api | http://localhost:8080 | Go API（プレフィックス `/api`） |
| mysql | `localhost:3307` | ホスト公開ポート。コンテナ内は `mysql:3306` |

> ホストの MySQL ポートを `3307` にしているのは、既存の MySQL が `3306` を使っている場合の競合を避けるためです（`.env` の `MYSQL_PORT` で変更可）。

worktrack（工数入力UI）と Storybook は未実装のため、現在 compose では無効化しています。

---

## トラブルシュート

- **`make up` でポート競合エラー** — 3000 / 8080 / 3307 が他プロセスで使われています。`.env` の `MYSQL_PORT` を変更するか、競合プロセスを停止してください。
- **ログイン直後の画面で一瞬エラーが出る** — API コンテナの起動直後はSSRの初回取得が間に合わないことがあります。リロードで解消します。
- **マイグレーションが `dirty` で失敗する** — `make migrate-force version=<直前の成功番号>` で解除してから `make migrate-up`。
- **データを完全に作り直したい** — `make reset`（全データ削除 → 再マイグレーション。その後 `make seed-link`）。
- **依存（go.mod / package.json）を変更した** — `make rebuild` でイメージを作り直します。

---

## 構成

```
pmo-agent/
├── api/                   # Go API（Gin + dig + GORM, JWT=HS256）
├── apps/
│   ├── pmo-dashboard/     # PMO管理画面（Nuxt 4 + Tailwind v4）
│   └── worktrack/         # 工数入力UI（未実装）
├── db/migrations/         # golang-migrate 連番マイグレーション
├── n8n/workflows/         # 進捗収集・レポート自動生成ワークフロー
├── docs/SPEC.md           # 仕様の正本
├── docker-compose.yml
└── Makefile               # すべての開発タスク
```

## License

[MIT](LICENSE)
