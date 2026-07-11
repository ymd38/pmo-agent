# MEMORY.md — 作業ログ

議論を経た意思決定・主要な変更・未解決の課題・気付きを残す。運用ルールは [`CLAUDE.md`](CLAUDE.md) の「作業ログ（MEMORY.md）」を参照。新しい日付を上に積む。

## 2026-07-11

### 意思決定

- **govulncheck fail は go.mod のツールチェーン更新で解消**（PR #13 で表面化）。検出26件は全て PR コード起因ではなく、`go 1.25.0` 宣言の標準ライブラリ25件（最遅 fix は go1.25.12）＋間接依存 quic-go v0.59.0 の1件。`go 1.25.12` へ更新し quic-go を v0.59.1 に上げて解消。CI は `go-version-file: api/go.mod` 参照のため go.mod の1箇所更新で完結。1PR=1目的の規約に従い PR #13 とは別の chore PR として対応（マージ後に PR #13 のブランチ更新が必要）。

- **スコープ制御（担当PJのみ）は今は実装せず Issue 化**（[#1](https://github.com/ymd38/pmo-agent/issues/1)）。SPEC が前提とする `project_members` テーブルとアサイン CRUD が未実装で、ミドルウェア追加ではなく機能追加が必要なため。P1「即修正」の範囲を超えると判断。→ その間 pm/member は全プロジェクト閲覧可能なままである点に注意。
- **`security-scan.yml` は他プロジェクト（migiudedirect-beta）由来をそのまま使わず本リポジトリ向けに最適化**。pnpm/`apps/frontend` は存在せず npm/`apps/pmo-dashboard` 構成のため。

### 変更内容

- コードレビュー(medium)の Critical Business Rule 違反のうち P1 の3件を修正（commit `34e9a60`）:
  - **#1 project_code 不変性**: `IssueCode` の UPDATE に `WHERE project_code IS NULL AND status='planning'` ＋ RowsAffected チェックを追加し、並行リクエストでの上書きを DB 層で遮断。
  - **無効化ユーザーの即時遮断**: `middleware.Authenticate` が毎リクエストで `is_active` を再確認（`UserRepo.IsActive` 追加）。`UserUsecase.Deactivate`／明示的 `is_active=false` でリフレッシュトークンを失効。
  - **#4 PUT /users の意図しない無効化防止**: `is_active` を `*bool` 化し未指定時は現在値維持。
  - 上記を `-race` テストで保護。
- 初期コードベース一式を `feat/demo` でコミットし `main` に fast-forward マージ（`34e9a60`）。
- `security-scan.yml` を追加後、本リポジトリ構成に最適化（commit `ecdce83`）。pnpm→npm、gitleaks は既定ルール、trivy の欠損ファイル参照削除、`govulncheck` 追加。

### 課題 / TODO

- レビュー指摘の残りは [#1〜#12](https://github.com/ymd38/pmo-agent/issues) として Issue 化済み。優先度高: #1(スコープ), #2(リフレッシュトークン再利用検知), #4(DBプール)。
- `origin/main` は `34e9a60` まで進んでいる（明示的な push はしていないが remote に存在）。ローカル `main`(`ecdce83`) は 1 commit ahead・未 push。`feat/demo` は `34e9a60` のままで main と分岐。
- `security-scan.yml` はまだ実運用未確認（actionlint 未導入・初回 PR で要検証）。`.gitleaks.toml`／`.trivyignore` は未配置（必要になれば追加）。

### 気付き

- **ローカルツールチェーンが壊れている**。Go: `/usr/local/go` の `go`(1.26.2) と同梱 `compile`(1.24.0) が不整合で stdlib すらビルド不可 → `GOTOOLCHAIN=go1.25.12 go test -race ./...` で回避（go.mod のツールチェーン更新に合わせて 1.25.0→1.25.12）。フロント: PATH 先頭の `node` が v0.10.25 と古く `vitest: command not found` → nvm の Node 22（`~/.nvm/versions/node/v22.14.0/bin`）を使う。
- セッション開始時、リポジトリはほぼ未コミット（`first commit` は README のみ、全コードが untracked）だった。`main` へのマージは実質「初期インポートのコミット」を伴った。
