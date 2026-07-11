# MEMORY.md — 作業ログ

議論を経た意思決定・主要な変更・未解決の課題・気付きを残す。運用ルールは [`CLAUDE.md`](CLAUDE.md) の「作業ログ（MEMORY.md）」を参照。新しい日付を上に積む。

## 2026-07-11

### 意思決定 / 変更内容（認証エンドポイントのレート制限＋タイミング平準化 — Issue #7）

- **レート制限はインメモリ実装**（`middleware.RateLimiter`）。`golang.org/x/time/rate` の `rate.Limiter` を key（クライアントIP）ごとに map で保持。フェーズ1は単一インスタンス前提のため Redis 等の外部ストアは持たない（YAGNI）。エントリは最終アクセスから `staleTTL`(10m) 超過で破棄（毎リクエスト線形走査で map 無限増加を防止。認証系は低頻度のため十分軽量）。
- **超過は 429 + `Retry-After: 60`**。既存 `middleware.abort`（`{"error": ...}`）を再利用。適用は公開認証系（`/auth/login`・`/auth/refresh`・`GET/POST /auth/set-password`）グループのみ。全体適用はしない（router.go で group 分割）。
- **設定は strict validation**（PR #18 方針踏襲）。`AUTH_RATE_LIMIT_PER_MIN`(既定10)/`AUTH_RATE_LIMIT_BURST`(既定5) を追加し、0・負数・非整数は `Load` で起動時エラー（「制限」目的のため0=無制限を許さない）。
- **タイミング平準化は usecase 層**（層分離）。`Login` で未知メール／未アクティベート／無効化のいずれでも、起動時に生成した使い捨てダミー bcrypt ハッシュへ `Hasher.Compare` を1回実行し bcrypt 相当コストを払う（応答時間差によるユーザー列挙を防止）。ダミーハッシュ生成失敗時は有効な固定フォールバックハッシュ（cost10）を使用。DB エラー経路では実行しない。
- **キー分離/回復のテスト容易化**のため `RateLimiter` に `now func()`/`keyFunc` を注入可能化（`AllowN(now,1)`）。テストは上限内通過・超過429・時間経過で回復・IP別独立・ttl クリーンアップを表駆動で保護。平準化は Compare 回数を数える `countingHasher` フェイクで検証。
- 依存追加は `golang.org/x/time`（準標準）のみ。`GOTOOLCHAIN=go1.25.12 go test -race ./...` / `golangci-lint run ./...` 0 issues。

### 意思決定 / 変更内容（DBコネクションプール設定 — Issue #4）

- **プール3設定（`SetMaxOpenConns`/`SetMaxIdleConns`/`SetConnMaxLifetime`）を `infra.NewDB` で `db.DB()` から適用**。無制限接続による `max_connections` 枯渇と、MySQL `wait_timeout`(既定8h)経過後の死んだ接続再利用（`invalid connection`）を塞ぐ。値は既存 config パターンに合わせ env 化（`DB_MAX_OPEN_CONNS`/`DB_MAX_IDLE_CONNS`/`DB_CONN_MAX_LIFETIME`）。デフォルトは open=25 / idle=25 / lifetime=5m（wait_timeout より十分短い）。
- **新 env の不正値は黙ってデフォルトへフォールバックせず `Load` でエラー**にする（設定ミスの早期発見）。既存 TTL 用 `envDuration` は挙動維持のまま残し、strict 版（`envInt`/`envDurationStrict`）を追加（TTL のフォールバック挙動を壊さないため二本立て）。
- **起動時 ping は回数上限つき線形バックオフ再試行**（`pingWithRetry`、10回・500ms×試行回数）。docker-compose で mysql より api が先に起動するケースに対応。リトライ回数/間隔は env 化せず定数（KISS/YAGNI）。
- テスト: config パース（デフォルト/上書き/非整数・不正 duration のエラー）と `pingWithRetry` の再試行ロジックをテーブル駆動で保護。プール適用・実 ping は DB 実接続が必要なためユニット対象外。`GOTOOLCHAIN=go1.25.12 go test -race ./...` / `golangci-lint run ./...` 0 issues。


### 意思決定 / 変更内容（パスワード変更時のトークン失効 — Issue #3）

- **`ChangePassword` 成功後に `refToks.RevokeAllForUser` を呼び既存セッションを全失効**させる。トークン窃取後に被害者がパスワードを変更しても盗まれたリフレッシュトークンが最大7日間有効なままになる穴を塞ぐ。関連コミットは本 Issue #3 ブランチ。
- **失効失敗は `_ =` で握りつぶさずエラー伝播**（`fmt.Errorf("usecase.ChangePassword revokeAll: %w", err)`）。PR #16 レビューの「セキュリティ制御の silent failure は不可」指摘に倣う。
- **同一クラスの `SetPassword` に残っていた `_ = uc.refToks.RevokeAllForUser(...)` の握りつぶしも同 PR でエラー伝播に統一**（1行修正・関連性が高いため）。
- テスト: `TestAuthUsecase_ChangePassword` に「変更成功で既存トークンが失効」「失効失敗はエラー伝播」ケースを追加。`SetPassword` にも失効失敗ケースを追加。既存フェイク（`fakeRefreshRepo.revokedUser` / `revokeAllErr`）で足りるため `fakes_test.go` は無変更（PR #16 とのコンフリクト回避）。`GOTOOLCHAIN=go1.25.12 go test -race ./...` / `golangci-lint run ./...` 0 issues。


### 意思決定（リフレッシュトークン再利用検知 — Issue #2 実装）

- **ローテーションのトランザクション境界は repository（`RefreshTokenRepo.Rotate`）に置く**。旧失効＋新規発行という複合 DB 操作の原子性は「DBアクセスの詳細」と判断し GORM の `Transaction` を repository 内に閉じた。usecase は「再利用検知 → チェーン全失効」という**ビジネス判断**のみ保持（層分離を維持）。汎用 TxManager 抽象は現状 tx 基盤が無く YAGNI/KISS に反するため導入せず。
- **失効判定はアプリ層 CAS でなく単一 UPDATE のガードで担保**。`Revoke`/`Rotate` を `WHERE id = ? AND revoked_at IS NULL` ＋ `RowsAffected==0→ErrTokenReuse` にし、MySQL の行ロックで並行リプレイでも失効成功は1件のみ（FindByHash→IsUsable→Revoke の非原子な TOCTOU を解消）。
- **再利用検知は2経路**: (1) FindByHash で `RevokedAt != nil`（失効済みの再提示）、(2) `Rotate` の CAS 敗北（並行リプレイ）。いずれも `RevokeAllForUser` でチェーン全失効させ `ErrTokenReuse`(401) を返す。盗用時は正規クライアントのセッションも巻き添えで失効するが、被害最小化を優先する設計。
- 新設 `domain.ErrTokenReuse` は `ErrTokenInvalid` と同様 401 に写像。`Logout` は失効済み(`ErrTokenReuse`)を冪等成功として黙認。

### 変更内容（Issue #2）

- `repository/token_repo.go`: `Revoke` に `revoked_at IS NULL` ガード＋RowsAffected チェック、`Rotate`（トランザクション原子ローテーション）を追加。
- `usecase/auth.go`: `Refresh` を再利用検知つきに再構成、`prepareTokens`（DB非依存の発行）を切り出し、`Logout` を冪等化。`ports.go` の `RefreshTokenRepository` に `Rotate` を追加。
- `domain/errors.go`・`handler/response.go`: `ErrTokenReuse`(401) を追加。
- テスト: 並行リプレイ（16 goroutine 同時 Refresh → 成功ちょうど1件・残り全て `ErrTokenReuse`・チェーン失効）を `-race` で保護。フェイクは mutex で DB の CAS を模擬。`GOTOOLCHAIN=go1.25.12 go test -race ./...` / `golangci-lint run ./...` 0 issues。

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
