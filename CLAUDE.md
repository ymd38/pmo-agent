# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# 作業フロー指示（コスト最適化）

あなたはSonnet 5レベルのexecutorとしてメインで動く。複雑な判断・計画・高品質レビューが必要な時だけ「Fable Advisor」を呼び出す。

## Advisor呼び出しルール
- タスク開始時：計画立案が必要 → Fableに相談
- コードレビュー/アーキテクチャ決定時
- 自信度が低い時、またはエラーが続く時
- ユーザーが「高品質」「ベストプラクティス」と指定した時

呼び出し時は明確に「Fable Advisorに相談: [具体的な質問]」とツール呼び出し（または明示）せよ。
普段はSonnetレベルの速さとコストで進める。

## Project Overview

PMO Agent — プロジェクト統制基盤。プロジェクトコード発行から工数・コスト管理・経営レポートまでを一気通貫で管理するモノレポ。

詳細仕様: [`docs/SPEC.md`](docs/SPEC.md)

## Monorepo Structure

```
pmo-agent-prototype/
├── api/                   # Go API サーバー → api/CLAUDE.md
├── apps/
│   ├── pmo-dashboard/     # PMO管理画面（経営層・PMO管理者・PM向け） → apps/pmo-dashboard/CLAUDE.md
│   └── worktrack/         # 工数入力UI（全メンバー向け） → apps/worktrack/CLAUDE.md
├── db/
│   └── migrations/        # golang-migrate 連番マイグレーションファイル
├── docker/
├── n8n/workflows/         # Backlog連携・レポート自動生成ワークフロー
└── docs/SPEC.md
```

## Sub-project Guidance

各コンポーネントの言語仕様・開発コマンド・アーキテクチャ詳細は以下を参照:

- **API (Go)**: [`api/CLAUDE.md`](api/CLAUDE.md)
- **PMO Dashboard (Nuxt 3)**: [`apps/pmo-dashboard/CLAUDE.md`](apps/pmo-dashboard/CLAUDE.md)
- **Worktrack (Nuxt 3)**: [`apps/worktrack/CLAUDE.md`](apps/worktrack/CLAUDE.md)

## Rules

- **Git 運用規約**（ブランチ命名・Conventional Commits・PRテンプレート）: [`.claude/rules/git.md`](.claude/rules/git.md)
- **作業ログの記録**（意思決定・変更・課題・気付き）: [`MEMORY.md`](MEMORY.md)

### 作業ログ（MEMORY.md）

議論を経た意思決定事項・主要な変更内容・未解決の課題・気付きは [`MEMORY.md`](MEMORY.md) に追記して残す。
次のセッションや他メンバーが「なぜそうしたか」を追えるようにするのが目的。コード差分で追える詳細は書かず要点のみ記す。

記録する対象:

- **意思決定**: 選んだ案・採用理由・却下した案（例: スコープ制御は `project_members` 未実装のため今は Issue 化）
- **変更内容**: 何を・なぜ変えたか（1〜3行）と関連コミット
- **課題 / TODO**: 既知の未対応事項と関連 Issue 番号
- **気付き**: ハマりどころ・環境固有の注意点（再発見のコストが高いもの）

追記フォーマットは日付見出し（`## YYYY-MM-DD`）配下に上記カテゴリの箇条書き。新しい日付を上に積む。

## UI/UX Design System

全フロントエンドは [`DESIGN.md`](DESIGN.md) のデザインシステムに従う。

核心ルール:
- **ダークモード専用** — `#090909` (canvas) が基底サーフェス。ライトモード実装禁止
- **アクセントカラーは1色のみ** — `#0099ff` はリンク・フォーカス・選択状態のみ。背景や CTAフィル禁止
- **CTAは全てpill形状** — `border-radius: 100px`。ゴーストボタン（枠線のみ）は使わない
- **サーフェス階層**: canvas(`#090909`) → surface-1(`#141414`) → surface-2(`#1c1c1c`)
- **ディスプレイ書体**: GT Walsheim Medium（代替: Mona Sans / Geist）、極端なネガティブトラッキング必須
- **ボディ書体**: Inter Variable + OpenType変数 `cv01, cv05, cv09, cv11, ss03, ss07`

デザイントークンのリント: `npx @google/design.md lint DESIGN.md`

## Dev Environment

```bash
make up        # 全サービス起動（mysql / api / pmo-dashboard / worktrack）
make down      # 全サービス停止
make restart   # 再起動
make logs      # ログ表示（全サービス）
make ps        # 起動中コンテナ確認
make db        # mysqlのみ起動
make db-cli    # MySQL CLIに接続
make migrate-up                  # マイグレーション適用
make migrate-down                # マイグレーション1件ロールバック
make migrate-create name=<name>  # 新規マイグレーションファイル作成
make storybook # Storybook起動（pmo-dashboard:6006 / worktrack:6007）
```

### ポート一覧

| サービス | ポート |
|---|---|
| api | 8080 |
| pmo-dashboard | 3000 |
| worktrack | 3001 |
| mysql | 3306 |
| storybook-pmo | 6006 |
| storybook-worktrack | 6007 |

## System Architecture

```
[ Backlog API ]
    ↓ n8n 日次収集
[ MySQL 8.0 ]  ←→  [ Go API (:8080) ]  ←→  [ pmo-dashboard (:3000) ]
                                         ←→  [ worktrack (:3001) ]
                          ↑
                    [ Claude MCP ]（フェーズ2）
```

## Critical Business Rules

コード実装時に必ず守るルール（詳細は `docs/SPEC.md`）:

- プロジェクトは必ずプログラムに属する（`projects.program_id` は NOT NULL）— プログラムはPMO集計単位。配下プロジェクトが存在するプログラムは削除不可
- `project_code` は発行後変更禁止 — API レベルで UPDATE をブロック。コードはプログラム作成時にプレフィックス発行、プロジェクト承認時に枝番発行
- ユーザーは論理削除のみ（`is_active=false`）— `work_hours`/`project_members` の FK は `ON DELETE RESTRICT` で履歴保護
- カテゴリ値削除は論理削除のみ（`is_active=false`）— 物理削除で過去のアサインが壊れる
- 工数（`work_hours`）の他ユーザー分編集禁止 — middleware でリクエストユーザーと `user_id` を照合
- 単価（`grade_rates`）は `pmo_admin` のみ参照可 — コスト集計APIはロールで返却フィールドを制御
- スコープ制御（「担当プロジェクトのみ」等）はAPIミドルウェアで実施 — フロントの表示制御に依存しない

## コーディング原則

全コンポーネント共通の指針。言語・フレームワーク固有の規約は各サブプロジェクトの CLAUDE.md を参照。

### DRY（Don't Repeat Yourself）

- 仕様・ルールの正本は [`docs/SPEC.md`](docs/SPEC.md)。CLAUDE.md やコードに詳細を再記述せず参照で繋ぐ（ドキュメントの二重管理を避ける）
- 権限定義は `role_functions` テーブルに一元化。ロール判定をコードにハードコードしない
- フロント2アプリ（pmo-dashboard / worktrack）で共通する処理（`useAuth` / `useApiError` / API型）は同一パターンで実装し重複を排除する
- ただし「偶然似ているだけ」のコードを無理に共通化しない（過度な抽象化は KISS に反する）

### KISS（Keep It Simple, Stupid）

- フェーズ1スコープ（認証・プロジェクト管理・工数・レポート閲覧）を超える作り込みをしない
- 責務を最小に保つ（例：プログラムは状態を持たず集計のみ）
- 凝った抽象より素直な実装を優先。重複が3回以上現れてから共通化を検討する

### YAGNI（You Aren't Gonna Need It）

- OTP / Microsoft Entra ID SSO / PMO Agent MCP は将来フェーズ。今は拡張点（`users` の拡張カラム等）だけ確保し、実装はしない
- プログラムの `budget` / 期間は列を持たず集計で算出する（将来必要になるまで持たせない）
- 「いつか使うかも」のための汎用化・設定項目を足さない

### SOLID（主に Go API バックエンド、フロントにも適用）

- **S 単一責任**: handler=I/O、usecase=ビジネスロジック、repository=DBアクセス。層をまたぐ責務混在を禁止。フロントは「1コンポーネント1責務」（Atomic Design）
- **O 開放閉鎖**: 新ロール・新権限は DBデータ追加で対応し、コード変更を不要にする
- **L リスコフ置換**: repository は usecase が定義する interface の契約を厳守する
- **I インターフェース分離**: 大きな interface を避け、利用側が必要とする最小単位で定義する
- **D 依存性逆転**: usecase は具体実装でなく interface に依存し、`di/`（dig）でバインドする。グローバル変数によるシングルトン共有は禁止

## テスト方針

共通方針（実行コマンド・フレームワーク詳細は各サブプロジェクトの CLAUDE.md を参照）:

- **テーブル駆動テスト**を基本とする（backend Go / frontend Vitest 共通）
- 実装の詳細ではなく **振る舞い** をテストする
- 上記 **Critical Business Rules は必ずテストで保護**する（リグレッション防止の最優先対象）:
  - `project_code` 発行後の UPDATE 拒否
  - 工数（`work_hours`）の他ユーザー分編集の拒否
  - スコープ制御（担当PJのみ）の境界
  - コスト集計の単価フィールドのロール別出し分け（PM に単価が漏れない）
  - ログイン可否（`password_hash IS NULL` / `is_active=false` の拒否）
- **backend**: `go test -race ./...` を PR の必須ゲートとする。DBを使うテストは test DB / testcontainers を使用
- **frontend**: Vitest + `@nuxt/test-utils`。新機能は期待出力を先に書く（TDD推奨）。Storybook の Story はビジュアル回帰の最小単位として実装と同時に作成する
