# MEMORY.md — 作業ログ

議論を経た意思決定・主要な変更・未解決の課題・気付きを残す。運用ルールは [`CLAUDE.md`](CLAUDE.md) の「作業ログ（MEMORY.md）」を参照。新しい日付を上に積む。

## 2026-07-11

### 意思決定（担当PJスコープ制御 — Issue #1 実装）

- **スコープはミドルウェアでロール→許可PJ ID集合（`domain.ProjectScope`）に解決し、usecase 引数で渡す**。gin.Context 依存を usecase に持ち込まず（クリーンアーキテクチャ維持）、`middleware.ResolveProjectScope()` が context に格納 → handler が `requireScope` で取り出し usecase に委譲。
- **担当外リソースは 403 でなく 404（`ErrNotFound`）**で返し存在を秘匿する（列挙攻撃・予算/ベンダー漏洩の防止）。`ProjectUsecase.Get` / `MemberUsecase.*` で適用。
- **スコープ適用範囲は Issue #1 が列挙した参照系＋メンバー系に限定**: `GET /projects`・`GET /projects/:id`・`GET /programs/:id/projects`・`/projects/:id/members`(CRUD)。
- **`GET /programs/:id`（プログラム詳細）が返す配下プロジェクト配列は本 PR では非スコープ**。PMO 集計視点（aggregate は全件前提）であり Issue の対象外。→ 残課題（下記 TODO）。
- スコープ規則: pmo_admin/executive=全件、pm/member=`project_members ∪ projects.pm_id`、planner=`created_by`。複数ロールは和集合。
- `requireScope` はミドルウェア未適用時に 500 を返すフェイルセーフ（ルーティング構成ミスで全件返却する事故を防ぐ）。

### 意思決定（CI/インフラ）

- **PR Agent は `config.model` だけでは動かない — `config.custom_model_max_tokens` が必須**。`openai/gpt-5-codex` は PR Agent 内部の MAX_TOKENS レジストリ未定義のため、上限未指定だと毎回エラー → fallback(`gpt-5.4-mini`) に落ち、code suggestions は「Failed to generate code suggestions for PR」で失敗していた（コミット 5bda3e7「モデル制御」は実質未機能だった）。ci.yml の両ジョブに `config.custom_model_max_tokens: "200000"` を追加して解消（PR #15・マージ済み）。
- **govulncheck fail は go.mod のツールチェーン更新で解消**（PR #13 で表面化）。検出26件は全て PR コード起因ではなく、`go 1.25.0` 宣言の標準ライブラリ25件（最遅 fix は go1.25.12）＋間接依存 quic-go v0.59.0 の1件。`go 1.25.12` へ更新し quic-go を v0.59.1 に上げて解消。CI は `go-version-file: api/go.mod` 参照のため go.mod の1箇所更新で完結。1PR=1目的の規約に従い PR #13 とは別の chore PR として対応（PR #14・マージ済み）。

### 変更内容

- `db/migrations/000006_project_members` 追加（SPEC.md:430 準拠。`user_id` FK は `ON DELETE RESTRICT`、スコープ逆引き用 `idx_member_user`）。`make migrate-up`/`migrate-down` の往復適用を確認。
- domain: `ProjectMember` / `ProjectScope`（+ロールコード定数）を追加。
- usecase: `ScopeUsecase.ResolveProjectScope`、`MemberUsecase`（アサイン CRUD・スコープ強制）、`ProjectUsecase.List/Get` にスコープ適用、`ProgramUsecase.ListProjectsScoped` 追加。
- repository: `MemberRepo` 新設、`ProjectRepo` に `ListByIDs/IDsByPM/IDsByCreator` 追加。
- middleware: `ScopeResolver` + `ResolveProjectScope()` + `ProjectScope()` ゲッター。`RequireFunction` を `slices.Contains` に整理。
- handler/router/di を配線（member 系エンドポイント追加）。`go test -race ./...`（`GOTOOLCHAIN=go1.25.0`）と golangci-lint 0 issues を確認。

### 意思決定（初期実装時点・履歴）

- ~~スコープ制御は今は実装せず Issue 化~~ → **本日 Issue #1 で実装済み**（上記）。当初は `project_members` 未実装のため P1 範囲外と判断していた。
- **`security-scan.yml` は他プロジェクト（migiudedirect-beta）由来をそのまま使わず本リポジトリ向けに最適化**。pnpm/`apps/frontend` は存在せず npm/`apps/pmo-dashboard` 構成のため。

### 変更内容（履歴）

- コードレビュー(medium)の Critical Business Rule 違反のうち P1 の3件を修正（commit `34e9a60`）:
  - **#1 project_code 不変性**: `IssueCode` の UPDATE に `WHERE project_code IS NULL AND status='planning'` ＋ RowsAffected チェックを追加し、並行リクエストでの上書きを DB 層で遮断。
  - **無効化ユーザーの即時遮断**: `middleware.Authenticate` が毎リクエストで `is_active` を再確認（`UserRepo.IsActive` 追加）。`UserUsecase.Deactivate`／明示的 `is_active=false` でリフレッシュトークンを失効。
  - **#4 PUT /users の意図しない無効化防止**: `is_active` を `*bool` 化し未指定時は現在値維持。
  - 上記を `-race` テストで保護。
- 初期コードベース一式を `feat/demo` でコミットし `main` に fast-forward マージ（`34e9a60`）。
- `security-scan.yml` を追加後、本リポジトリ構成に最適化（commit `ecdce83`）。pnpm→npm、gitleaks は既定ルール、trivy の欠損ファイル参照削除、`govulncheck` 追加。

### 課題 / TODO

- **`GET /programs/:id`（プログラム詳細）の配下プロジェクト配列がスコープ非適用**。pm/planner が担当外PJの予算・ベンダーを詳細経由で閲覧できる余地。集計値との整合（担当PJのみ表示 vs 全件集計）の設計判断が必要なため別 Issue 化を推奨。
- レビュー指摘の残りは [#2〜#12](https://github.com/ymd38/pmo-agent/issues) として Issue 化済み。優先度高: #2(リフレッシュトークン再利用検知), #4(DBプール)。
- `security-scan.yml` はまだ実運用未確認（actionlint 未導入・初回 PR で要検証）。`.gitleaks.toml`／`.trivyignore` は未配置（必要になれば追加）。

### 気付き

- **担当PJスコープの単体テストは fake repo で完結**。`fakeProjectRepo` は map 実装のため `List` は ID 昇順で返すよう `sortedWhere` を追加（実 DB の `Order("id")` と挙動を揃えないと List/scope テストが不安定になる）。
- **ローカルツールチェーンが壊れている**。Go: `/usr/local/go` の `go`(1.26.2) と同梱 `compile`(1.24.0) が不整合で stdlib すらビルド不可 → `GOTOOLCHAIN=go1.25.12 go test -race ./...` で回避（go.mod のツールチェーン更新に合わせて 1.25.0→1.25.12）。フロント: PATH 先頭の `node` が v0.10.25 と古く `vitest: command not found` → nvm の Node 22（`~/.nvm/versions/node/v22.14.0/bin`）を使う。
- セッション開始時、リポジトリはほぼ未コミット（`first commit` は README のみ、全コードが untracked）だった。`main` へのマージは実質「初期インポートのコミット」を伴った。
