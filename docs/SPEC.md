# PMO Agent — システム設計仕様書

## 概要

組織横断の「プロジェクト統制の欠如」を解決するための管理システム。プロジェクトコードの発行からタスク管理・工数・コスト管理・経営報告までを一気通貫で管理する。

### 解決する課題

- プロジェクトの予算・スケジュール・委託先の妥当性が評価できていない
- 進行中プロジェクトの状況・課題・リスクが可視化されていない
- 内部工数コストが把握されておらず、プロジェクトの真のコストが不明
- 管理ツールが統一されておらず、ベンダー側やExcelに分散している

### システム構成

```
[ Backlog ]
    ↓ n8n 日次収集（Backlog API）
[ MySQL ]  ←→  [ Go API Server ]  ←→  [ PMO Dashboard (Nuxt 4) ]
                                   ←→  [ 工数管理UI (Nuxt 4) ]
                    ↑
              [ Claude / Cowork ]
              （フェーズ2: PMO Agent MCP）
```

---

## アーキテクチャ

### 技術スタック

| 役割 | 技術 |
|---|---|
| API サーバー | Go（Gin） |
| フロントエンド（PMO） | Nuxt 4（Vue 3） |
| フロントエンド（工数管理） | Nuxt 4（Vue 3） |
| スタイリング | Tailwind CSS |
| DB | MySQL 8.0 |
| 自動ワークフロー | n8n |
| コンテナ | Docker / docker-compose |

### リポジトリ構成（モノレポ）

```
pmo-agent/
├── api/                          # Go API サーバー
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── auth/                 # 認証・JWT
│   │   ├── middleware/           # 認証・ACLミドルウェア
│   │   ├── handler/              # HTTPハンドラー
│   │   ├── model/                # DB モデル
│   │   ├── repository/           # DBアクセス層
│   │   └── service/              # ビジネスロジック
│   ├── go.mod
│   └── Dockerfile
├── apps/
│   ├── pmo-dashboard/            # PMO Agent UI（経営層・PMO管理者・PM向け）
│   └── worktrack/                # 工数管理 UI（全メンバー向け）
├── docker/
│   └── mysql/init/01_schema.sql
├── n8n/workflows/
├── docs/
│   └── SPEC.md
└── docker-compose.yml
```

将来的にリポジトリを `api/` と `apps/` で分割することを想定し、インターフェースを明確に保つ。

---

## 認証・認可

### 認証方式

- **パスワード認証 + JWT**（httpOnly Cookie）
- bcrypt でパスワードをハッシュ化
- 認証トークンは2種：**アクセストークン（JWT・有効期限8時間）** と **リフレッシュトークン（有効期限7日・httpOnly Cookie・DB保管）**。アクセストークン失効時は `/auth/refresh` で再発行し、リフレッシュトークンはローテーション（旧トークン失効）する
- 将来的に OTP（メールワンタイムパスワード）や Microsoft Entra ID SSO への移行パスを確保するため、`users` テーブルに拡張カラムを追加できる設計にしておく

```
POST /api/auth/login   → メール+パスワード検証 → アクセス/リフレッシュトークン発行 → Cookie set
POST /api/auth/refresh → リフレッシュトークン検証 → アクセストークン再発行＋リフレッシュトークンをローテーション
POST /api/auth/logout  → リフレッシュトークン失効 → Cookie クリア
GET  /api/auth/me      → アクセストークン検証 → ユーザー情報返却
```

### アカウント発行・パスワード設定

セルフサインアップは行わず、**PMO管理者がアカウントを発行する**（`manage_users`）。メール送信基盤には依存せず、**ワンタイムトークン付きリンクを管理者が手動共有**する方式とする。「招待」と「パスワードリセット」は同一のトークン機構で実現する。

```
1. PMO管理者がユーザー作成（email / name / grade / role(s) を入力）
     → password_hash = NULL（未アクティベート）でレコード作成
     → ワンタイムトークンを発行し、レスポンスに設定用URLを返す
2. 管理者が設定用URLを本人へ共有（Slack / Teams 等の既存チャネル）
3. 本人がURLを開き、自分でパスワードを設定 → password_hash 確定 → ログイン可能
4. パスワード忘れ・ロックアウト時は、管理者がリンクを再発行して共有（同一機構）
```

- **ロールはユーザー作成時に同時付与**する（`user_roles`）。後から `PUT /users/:id` で変更可。最低1ロールを必須とする
- `password_hash IS NULL` を「未アクティベート」状態として扱う（`is_active` は無効化フラグとして別管理）。**ログインは `password_hash IS NOT NULL` かつ `is_active = true` のユーザーのみ許可**する
- トークンは**ハッシュ化して保存・単回利用・有効期限付き（72時間）**。再発行時は当該ユーザーの未使用トークンを失効させ、有効トークンは常に最新1件のみとする
- メール送信基盤が整い次第、リンク配信をメール自動送信へ差し替えられる設計とする（トークン機構・エンドポイントは変更不要）
- **初期管理者（コールドスタート）**: PMO管理者は `password_hash = NULL` でマイグレーションにより seed し、`pmo_admin` ロールを付与する（メールアドレスは環境ごとに設定）。初回のパスワード設定リンクは `make seed-link email=<管理者メール>` で発行しコンソール出力する。以降の自己リセットは pmo_admin が自分自身に対して `reissue-link` を実行すればよい
- **ログイン中のパスワード変更**は `POST /auth/change-password`（現パスワード検証付き）で行う。忘れた場合のリセット（トークン経由）とは別経路

### ロール設計

ロールと機能権限はハードコードせず、DBテーブルで管理する。例外ケースは専用ロールを作成することで対応する。

#### 標準ロール（初期データ）

| code | name | 説明 |
|---|---|---|
| `executive` | 経営層 | エグゼクティブレポート・ダッシュボード閲覧専用 |
| `pmo_admin` | PMO管理者 | 全機能へのアクセス |
| `pm` | プロジェクトマネージャー | 担当プロジェクトの管理・レポート閲覧 |
| `member` | メンバー | 担当プロジェクトの工数入力・確認 |
| `planner` | 担当者（起案者） | プロジェクト起案・AIレビューSkill実行 |

#### 機能権限一覧（functions）

| code | name |
|---|---|
| `view_dashboard` | ダッシュボード閲覧 |
| `view_executive_report` | エグゼクティブレポート閲覧 |
| `view_project_report` | プロジェクト別レポート閲覧 |
| `issue_project_code` | プログラム作成・プロジェクトコード発行 |
| `manage_projects` | プロジェクト作成・編集・削除 |
| `view_project_detail` | プロジェクト詳細閲覧 |
| `manage_members` | ユーザー・単価管理 |
| `assign_project_members` | プロジェクトへのメンバーアサイン |
| `input_work_hours` | 工数入力（自分のみ） |
| `view_work_hours` | 他メンバーの工数閲覧 |
| `view_project_cost` | プロジェクトコスト閲覧（外注費＋工数コスト） |
| `manage_categories` | プロジェクト属性カテゴリ管理 |
| `manage_grade_rates` | グレード別単価管理 |
| `manage_roles` | ロール・権限管理 |
| `manage_users` | ユーザー管理 |

#### 初期ロール別権限マッピング

| function | executive | pmo_admin | pm | member | planner |
|---|---|---|---|---|---|
| view_dashboard | ✓ | ✓ | ✓ | | |
| view_executive_report | ✓ | ✓ | | | |
| view_project_report | | ✓ | ✓（担当PJのみ） | ✓（担当PJのみ） | |
| issue_project_code | | ✓ | | | |
| manage_projects | | ✓ | ✓（担当PJのみ） | | |
| view_project_detail | | ✓ | ✓（担当PJのみ） | ✓（担当PJのみ） | ✓（自起案PJのみ） |
| manage_members | | ✓ | | | |
| assign_project_members | | ✓ | ✓（担当PJのみ） | | |
| input_work_hours | | ✓ | ✓ | ✓ | |
| view_work_hours | | ✓ | ✓（担当PJのみ） | | |
| view_project_cost | | ✓ | ✓（担当PJのみ・単価非表示） | | |
| manage_categories | | ✓ | | | |
| manage_grade_rates | | ✓ | | | |
| manage_roles | | ✓ | | | |
| manage_users | | ✓ | | | |

> スコープ制御（担当PJのみ等）はAPIミドルウェアで実装する。

---

## 機能仕様

### プロジェクト・プログラム管理

**プログラム**（PMOの管理・集計単位）と **プロジェクト**（実作業単位）の2テーブルで管理する。プロジェクトは必ずいずれかのプログラムに属する。

| 概念 | 役割 |
|---|---|
| プログラム | PMOの管理上の集計単位。プロジェクトコードの共通部分（プレフィックス）を発行する名前空間。予算・期間は配下プロジェクトの集計値として表示 |
| プロジェクト | 実作業単位。PM・決済者・委託先・予算・Backlog連携・ステータスを持つ |

- プロジェクトは `program_id`（NOT NULL）で所属プログラムを参照する
- 1プログラムに複数プロジェクトが属する（1プログラム＝1プロジェクトの構成も許容）
- プログラムは **状態を持たない**（承認フロー不要）。PMO管理者がコードプレフィックス確保のために作成する
- プログラムの `budget` は配下プロジェクトの合計、`start_date` / `end_date` は配下の最早開始日・最遅終了日として算出（表示用）。DBには保持しない
- 各プロジェクトにPM（プロジェクトマネージャー）と決済者を設定
- ステータス管理は **プロジェクト単位**：`planning`（起案中） → `active`（進行中） → `completed`（完了） / `cancelled`（中止）
- プロジェクトコードは `planning` 状態で登録後、PMO管理者が審査・承認時に枝番を採番して発行（`active` に遷移）

### プロジェクトコード

PMO管理者が発行するユニークな業務キー。コードはプログラム／プロジェクトの2段で構成し、連番はシステムが自動採番する（人手で番号を管理しない）。

| 概念 | フォーマット例 | 発行タイミング | 備考 |
|---|---|---|---|
| プログラム | `INV-2026-0001` | プログラム作成時 | 種別-会計年度-連番（4桁・自動採番） |
| プロジェクト | `INV-2026-0001-001` | プロジェクト承認時（active遷移） | プログラムコード-枝番（3桁・自動採番） |

- 種別プレフィックス例：`INV`（投資）、`MNT`（保守）、`OPS`（運用）。種別は英大文字2〜5文字
- **プログラムコードは作成時に自動生成**：PMO管理者は種別と会計年度のみ指定し、連番は `(種別, 会計年度)` ごとに `MAX(seq_no)+1` で自動採番する。`programs` は `type` / `fiscal_year` / `seq_no` を保持し `UNIQUE(type, fiscal_year, seq_no)`、`programs.code` は materialized 値で UNIQUE
- プロジェクトコードは承認時に枝番を採番して発行する。`planning` 中は `projects.project_code` = NULL（`projects.project_code` に UNIQUE 制約）
- 枝番はプログラム内で連番採番（`projects.branch_no`、`UNIQUE(program_id, branch_no)`）
- 発行後は変更不可（UPDATE禁止）
- プロジェクトコードは Backlog の Phase 名と対応させる

### 属性管理（EAVパターン）

プロジェクトに紐づくカテゴリ・属性を柔軟に追加・更新・削除できる。

- `project_categories`：カテゴリ定義（例：機能領域・システム・案件種別・開発手法）
- `project_category_values`：カテゴリに属する値（例：機能領域 → 会員管理/決済/通知）
- `project_attribute_assignments`：プロジェクトと値の紐付け（1プロジェクトに同カテゴリ複数値可）
- 値の削除は `is_active=false` による論理削除のみ（過去プロジェクトのアサインを保護）
- カテゴリはプロジェクトコードとは独立した分類・フィルタ用メタデータ

### メンバー・工数管理

#### メンバー管理

- ユーザーをプロジェクトにアサイン（担当開始日・終了日・工数予定割合を設定）
- グレード（`manager` / `staff`）は `users.grade` で管理

#### 単価管理

- グレード別・年度別のフルコスト時間単価を設定
- 単価 = 給与 + 社会保険料 + 福利厚生 + 間接コスト配賦 ÷ 年間稼働時間（基準 1,800h）
- 単価情報は `pmo_admin` のみ参照可能

#### 工数入力

- メンバーが日次で担当プロジェクトへの工数（時間）を入力
- 自分の工数のみ入力・編集可能

#### コスト集計

- プロジェクト全体コスト = 外注コスト（`projects.budget`） + 内部工数コスト
- 内部工数コスト = Σ（work_hours.hours × grade_rates.hourly_rate）※ユーザーのグレード・年度に対応する単価を使用
- PM には合計コストのみ表示（単価の内訳は非表示）

### 進捗管理（Backlog連携）

- n8n の日次ワークフローで Backlog API からタスクデータを収集
- `projects.backlog_project_id` を使って対象プロジェクトを特定
- 取得したタスクを `project_progress` テーブルに蓄積
- 収集後に進捗分析・リスク検知・トレンド算出を実行し `daily_reports` に保存

### レポート

| 種別 | 生成タイミング | 内容 |
|---|---|---|
| 日次レポート | 毎日（n8n） | プロジェクト単位の進捗・リスク・トレンド・AIコメント |
| 週次レポート | 毎週月曜（n8n） | 日次レポートの週次集約・AIコメント |
| エグゼクティブレポート | 毎週月曜（n8n） | 全プロジェクト横断総括・経営視点AIコメント |

---

## DBスキーマ

### 認証・アカウント管理

```sql
-- ユーザー（認証 + 社員情報を統合）
CREATE TABLE users (
  id            INT AUTO_INCREMENT PRIMARY KEY,
  email         VARCHAR(255) NOT NULL UNIQUE,
  name          VARCHAR(100) NOT NULL,
  password_hash VARCHAR(255),                         -- NULL=未アクティベート（招待リンクで設定後に確定）
  grade         ENUM('manager', 'staff') NOT NULL,  -- 内部単価グレード
  is_active     BOOLEAN NOT NULL DEFAULT true,
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- ロール定義
CREATE TABLE roles (
  id          INT AUTO_INCREMENT PRIMARY KEY,
  code        VARCHAR(50) NOT NULL UNIQUE,
  name        VARCHAR(100) NOT NULL,
  description TEXT,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- ユーザーとロールの紐付け（1ユーザーが複数ロールを持てる）
CREATE TABLE user_roles (
  user_id    INT NOT NULL,
  role_id    INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, role_id),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

-- 機能権限定義
CREATE TABLE functions (
  id          INT AUTO_INCREMENT PRIMARY KEY,
  code        VARCHAR(100) NOT NULL UNIQUE,
  name        VARCHAR(100) NOT NULL,
  description TEXT,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ロールと機能権限の紐付け
CREATE TABLE role_functions (
  role_id     INT NOT NULL,
  function_id INT NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (role_id, function_id),
  FOREIGN KEY (role_id)     REFERENCES roles(id)     ON DELETE CASCADE,
  FOREIGN KEY (function_id) REFERENCES functions(id) ON DELETE CASCADE
);

-- パスワード設定トークン（招待・リセット共通 / 単回利用）
CREATE TABLE password_set_tokens (
  id         INT AUTO_INCREMENT PRIMARY KEY,
  user_id    INT NOT NULL,
  token_hash VARCHAR(255) NOT NULL UNIQUE,            -- トークンはハッシュ化して保存
  expires_at TIMESTAMP NOT NULL,                       -- 発行から72時間
  used_at    TIMESTAMP NULL,                           -- 使用済み時刻（単回利用）
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  INDEX idx_user (user_id)
);

-- リフレッシュトークン（ローテーション・失効管理）
CREATE TABLE refresh_tokens (
  id         INT AUTO_INCREMENT PRIMARY KEY,
  user_id    INT NOT NULL,
  token_hash VARCHAR(255) NOT NULL UNIQUE,            -- トークンはハッシュ化して保存
  expires_at TIMESTAMP NOT NULL,                       -- 発行から7日
  revoked_at TIMESTAMP NULL,                           -- 失効時刻（ローテーション/ログアウト/無効化）
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  INDEX idx_user (user_id)
);
```

### プロジェクト管理

```sql
-- プログラムマスタ（PMOの管理・集計単位 / コードプレフィックスの名前空間）
CREATE TABLE programs (
  id          INT AUTO_INCREMENT PRIMARY KEY,
  code        VARCHAR(100) NOT NULL UNIQUE,            -- 種別-年度-連番（例: INV-2026-0001）。materialized・不変
  type        VARCHAR(8) NOT NULL,                     -- 種別プレフィックス（例: INV）
  fiscal_year SMALLINT NOT NULL,                       -- 会計年度（例: 2026）
  seq_no      SMALLINT NOT NULL,                       -- (type, fiscal_year) 内の連番。自動採番
  name        VARCHAR(255) NOT NULL,
  description TEXT,
  created_by  INT NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_type_year_seq (type, fiscal_year, seq_no),
  FOREIGN KEY (created_by) REFERENCES users(id)
);

-- プロジェクトマスタ（実作業単位 / 必ずプログラムに属する）
CREATE TABLE projects (
  id                 INT AUTO_INCREMENT PRIMARY KEY,
  program_id         INT NOT NULL,                        -- 所属プログラム（必須）
  branch_no          SMALLINT,                            -- プログラム内の枝番（承認時に採番。例: 1 → -001）
  project_code       VARCHAR(100) UNIQUE,                 -- 承認時に発行する枝番付き業務キー（発行前はNULL）
  name               VARCHAR(255) NOT NULL,
  description        TEXT,
  pm_id              INT,                                 -- プロジェクトマネージャー
  approver_id        INT,                                 -- 決済者
  vendor             VARCHAR(255),                        -- 委託先
  budget             DECIMAL(15, 0),                      -- 外注予算（円）
  start_date         DATE,
  end_date           DATE,
  status             ENUM('planning','active','completed','cancelled') NOT NULL DEFAULT 'planning',
  backlog_project_id VARCHAR(255),                        -- Backlog Project ID
  ai_review_doc_path TEXT,                                -- AIレビュー結果ドキュメントのパス
  created_by         INT NOT NULL,
  created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_program_branch (program_id, branch_no),
  FOREIGN KEY (program_id)  REFERENCES programs(id) ON DELETE RESTRICT,
  FOREIGN KEY (pm_id)       REFERENCES users(id) ON DELETE SET NULL,
  FOREIGN KEY (approver_id) REFERENCES users(id) ON DELETE SET NULL,
  FOREIGN KEY (created_by)  REFERENCES users(id)
);
```

> **設計注意**: プロジェクトは `program_id`（NOT NULL）で必ずプログラムに属する。プログラムは状態を持たず、予算は配下プロジェクトの合計、期間は配下の最早開始日・最遅終了日として算出し、DB上は保持しない。配下にプロジェクトが存在するプログラムは削除不可（`ON DELETE RESTRICT`）。

### 属性管理（EAV）

```sql
-- カテゴリ定義
CREATE TABLE project_categories (
  id          INT AUTO_INCREMENT PRIMARY KEY,
  code        VARCHAR(50) NOT NULL UNIQUE,
  name        VARCHAR(100) NOT NULL,
  description TEXT,
  is_required BOOLEAN NOT NULL DEFAULT false,
  sort_order  INT NOT NULL DEFAULT 0,
  is_active   BOOLEAN NOT NULL DEFAULT true,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- カテゴリに属する値
CREATE TABLE project_category_values (
  id          INT AUTO_INCREMENT PRIMARY KEY,
  category_id INT NOT NULL,
  code        VARCHAR(50) NOT NULL,
  label       VARCHAR(100) NOT NULL,
  sort_order  INT NOT NULL DEFAULT 0,
  is_active   BOOLEAN NOT NULL DEFAULT true,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_category_code (category_id, code),
  FOREIGN KEY (category_id) REFERENCES project_categories(id) ON DELETE CASCADE
);

-- プロジェクトと属性値の紐付け（同カテゴリに複数値可）
CREATE TABLE project_attribute_assignments (
  id         INT AUTO_INCREMENT PRIMARY KEY,
  project_id INT NOT NULL,
  category_id INT NOT NULL,
  value_id   INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_project_value (project_id, value_id),
  FOREIGN KEY (project_id)  REFERENCES projects(id)               ON DELETE CASCADE,
  FOREIGN KEY (category_id) REFERENCES project_categories(id),
  FOREIGN KEY (value_id)    REFERENCES project_category_values(id)
);
```

### メンバー・工数管理

```sql
-- プロジェクトメンバーアサイン
CREATE TABLE project_members (
  id                 INT AUTO_INCREMENT PRIMARY KEY,
  project_id         INT NOT NULL,
  user_id            INT NOT NULL,
  allocation_percent DECIMAL(5, 2),                    -- 工数予定割合（%）
  start_date         DATE,
  end_date           DATE,
  created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_project_user (project_id, user_id),
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id)    REFERENCES users(id)    ON DELETE RESTRICT   -- ユーザーは論理削除のみ。アサイン履歴を保護
);

-- グレード別・年度別単価
CREATE TABLE grade_rates (
  id          INT AUTO_INCREMENT PRIMARY KEY,
  grade       ENUM('manager', 'staff') NOT NULL,
  fiscal_year INT NOT NULL,                            -- 年度（例：2026）
  hourly_rate DECIMAL(10, 0) NOT NULL,                 -- 時間単価（円）
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_grade_year (grade, fiscal_year)
);

-- 工数実績
CREATE TABLE work_hours (
  id         INT AUTO_INCREMENT PRIMARY KEY,
  user_id    INT NOT NULL,
  project_id INT NOT NULL,
  work_date  DATE NOT NULL,
  hours      DECIMAL(4, 1) NOT NULL,                   -- 工数（時間）
  note       TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id)    REFERENCES users(id)    ON DELETE RESTRICT,  -- ユーザーは論理削除のみ。工数履歴を保護
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
  INDEX idx_user_date    (user_id, work_date),
  INDEX idx_project_date (project_id, work_date)
);
```

### 進捗管理

```sql
-- Backlogから収集したタスク進捗
CREATE TABLE project_progress (
  id           INT AUTO_INCREMENT PRIMARY KEY,
  project_id   INT NOT NULL,
  phase        VARCHAR(100),                           -- Backlog上の管理単位
  task         TEXT,
  status       VARCHAR(50),
  progress     DECIMAL(5, 2),
  deadline     DATE,
  register_day DATE NOT NULL,                          -- 収集日
  created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
  INDEX idx_project_day (project_id, register_day)
);
```

### レポート

```sql
-- 日次レポート
CREATE TABLE daily_reports (
  id               INT AUTO_INCREMENT PRIMARY KEY,
  project_id       INT NOT NULL,
  report_date      DATE NOT NULL,
  overall_progress DECIMAL(5, 2),
  phase_summary    JSON,
  risks            JSON,
  trend            JSON,
  ai_comment       TEXT,
  created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_project_date (project_id, report_date),
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- 週次レポート
CREATE TABLE weekly_reports (
  id              INT AUTO_INCREMENT PRIMARY KEY,
  project_id      INT NOT NULL,
  week_start_date DATE NOT NULL,
  week_end_date   DATE NOT NULL,
  summary         JSON,
  ai_comment      TEXT,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_project_week (project_id, week_start_date),
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- エグゼクティブレポート（全プロジェクト横断）
CREATE TABLE executive_reports (
  id          INT AUTO_INCREMENT PRIMARY KEY,
  report_date DATE NOT NULL,
  report_type VARCHAR(50) NOT NULL DEFAULT 'weekly',
  content     JSON,
  ai_comment  TEXT,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_date_type (report_date, report_type)
);
```

---

## API設計（Go）

全エンドポイントのプレフィックスは `/api`。JWT認証が必要なエンドポイントはミドルウェアで検証し、ACLは `role_functions` テーブルを参照して制御する。

### 認証

| メソッド | パス | 説明 | 認証 |
|---|---|---|---|
| POST | `/auth/login` | ログイン・アクセス/リフレッシュトークン発行 | 不要 |
| POST | `/auth/refresh` | アクセストークン再発行（リフレッシュトークンをローテーション） | リフレッシュトークン |
| POST | `/auth/logout` | ログアウト・リフレッシュトークン失効・Cookie削除 | 必要 |
| GET | `/auth/me` | ログイン中ユーザー情報 | 必要 |
| POST | `/auth/change-password` | ログイン中のパスワード変更（現パスワード検証） | 必要 |
| GET | `/auth/set-password/:token` | トークン検証（有効性・対象メール返却） | 不要 |
| POST | `/auth/set-password` | トークン＋新パスワードでパスワード設定（招待・リセット共通） | 不要 |

### ユーザー管理

| メソッド | パス | 説明 | 必要権限 |
|---|---|---|---|
| GET | `/users` | ユーザー一覧 | `manage_users` |
| POST | `/users` | ユーザー作成（ロール付与・招待リンク発行。レスポンスに設定用URL） | `manage_users` |
| GET | `/users/:id` | ユーザー詳細 | `manage_users` |
| PUT | `/users/:id` | ユーザー更新（ロール変更含む） | `manage_users` |
| DELETE | `/users/:id` | ユーザー削除（論理） | `manage_users` |
| POST | `/users/:id/reissue-link` | 招待/リセット用リンク再発行（レスポンスに設定用URL） | `manage_users` |

### ロール・権限管理

| メソッド | パス | 説明 | 必要権限 |
|---|---|---|---|
| GET | `/roles` | ロール一覧 | `manage_roles` |
| POST | `/roles` | ロール作成 | `manage_roles` |
| PUT | `/roles/:id` | ロール更新 | `manage_roles` |
| DELETE | `/roles/:id` | ロール削除 | `manage_roles` |
| GET | `/roles/:id/functions` | ロールの権限一覧 | `manage_roles` |
| PUT | `/roles/:id/functions` | ロールの権限一括更新 | `manage_roles` |
| GET | `/functions` | 機能権限一覧 | `manage_roles` |

### プログラム管理

| メソッド | パス | 説明 | 必要権限 |
|---|---|---|---|
| GET | `/programs` | プログラム一覧。各プログラムに配下件数・集計予算・集計期間・配下ステータス分布を付与 | `view_project_detail` |
| POST | `/programs` | プログラム作成（種別+会計年度を指定、連番は自動採番） | `issue_project_code` |
| GET | `/programs/:id` | プログラム詳細（配下プロジェクト一覧を含む） | `view_project_detail` |
| PUT | `/programs/:id` | プログラム更新（name / description） | `issue_project_code` |
| DELETE | `/programs/:id` | プログラム削除（配下プロジェクトが存在する場合は409） | `issue_project_code` |
| GET | `/programs/:id/projects` | 配下プロジェクト一覧 | `view_project_detail` |
| POST | `/programs/:id/projects` | プロジェクト作成（`program_id = :id` で登録） | `manage_projects` |

### プロジェクト管理

| メソッド | パス | 説明 | 必要権限 |
|---|---|---|---|
| GET | `/projects` | プロジェクト一覧（スコープ制御あり） | `view_project_detail` |
| GET | `/projects/:id` | プロジェクト詳細 | `view_project_detail` |
| PUT | `/projects/:id` | プロジェクト更新 | `manage_projects` |
| DELETE | `/projects/:id` | プロジェクト削除 | `manage_projects` |
| POST | `/projects/:id/issue-code` | プロジェクトコード発行（枝番採番＋active遷移） | `issue_project_code` |

### プロジェクト属性

| メソッド | パス | 説明 | 必要権限 |
|---|---|---|---|
| GET | `/projects/:id/attributes` | プロジェクトの属性一覧 | `view_project_detail` |
| POST | `/projects/:id/attributes` | 属性値の紐付け追加 | `manage_projects` |
| DELETE | `/projects/:id/attributes/:valueId` | 属性値の紐付け削除 | `manage_projects` |

### プロジェクトメンバー

| メソッド | パス | 説明 | 必要権限 |
|---|---|---|---|
| GET | `/projects/:id/members` | メンバー一覧 | `view_project_detail` |
| POST | `/projects/:id/members` | メンバーアサイン | `assign_project_members` |
| PUT | `/projects/:id/members/:userId` | アサイン情報更新 | `assign_project_members` |
| DELETE | `/projects/:id/members/:userId` | アサイン解除 | `assign_project_members` |

### プロジェクトコスト

| メソッド | パス | 説明 | 必要権限 |
|---|---|---|---|
| GET | `/projects/:id/cost` | コスト集計（外注費＋内部工数コスト） | `view_project_cost` |

### 属性カテゴリ管理

| メソッド | パス | 説明 | 必要権限 |
|---|---|---|---|
| GET | `/categories` | カテゴリ一覧 | 認証のみ |
| POST | `/categories` | カテゴリ作成 | `manage_categories` |
| PUT | `/categories/:id` | カテゴリ更新 | `manage_categories` |
| DELETE | `/categories/:id` | カテゴリ論理削除 | `manage_categories` |
| GET | `/categories/:id/values` | カテゴリ値一覧 | 認証のみ |
| POST | `/categories/:id/values` | カテゴリ値追加 | `manage_categories` |
| PUT | `/categories/:id/values/:valueId` | カテゴリ値更新 | `manage_categories` |
| DELETE | `/categories/:id/values/:valueId` | カテゴリ値論理削除 | `manage_categories` |

### 工数管理

| メソッド | パス | 説明 | 必要権限 |
|---|---|---|---|
| GET | `/work-hours` | 自分の工数一覧 | `input_work_hours` |
| POST | `/work-hours` | 工数入力 | `input_work_hours` |
| PUT | `/work-hours/:id` | 工数更新（自分のみ） | `input_work_hours` |
| DELETE | `/work-hours/:id` | 工数削除（自分のみ） | `input_work_hours` |
| GET | `/projects/:id/work-hours` | プロジェクト別工数一覧 | `view_work_hours` |

### 単価管理

| メソッド | パス | 説明 | 必要権限 |
|---|---|---|---|
| GET | `/grade-rates` | 単価一覧 | `manage_grade_rates` |
| POST | `/grade-rates` | 単価登録 | `manage_grade_rates` |
| PUT | `/grade-rates/:id` | 単価更新 | `manage_grade_rates` |

### レポート

| メソッド | パス | 説明 | 必要権限 |
|---|---|---|---|
| GET | `/dashboard` | ダッシュボード集計データ | `view_dashboard` |
| GET | `/reports/executive` | エグゼクティブレポート一覧 | `view_executive_report` |
| GET | `/reports/:projectId/daily` | 日次レポート（最新30件） | `view_project_report` |
| GET | `/reports/:projectId/weekly` | 週次レポート（全件） | `view_project_report` |

---

## フロントエンド構成

### PMO Dashboard（pmo-dashboard）

PMO管理者・経営層・PM向けの管理画面。

```
/                              ダッシュボード
/programs                      プログラム一覧（配下プロジェクトをツリー表示）
/programs/new                  プログラム新規登録
/programs/:id                  プログラム詳細（配下プロジェクト一覧・集計コスト）
/programs/:id/edit             プログラム編集
/programs/:id/projects/new     プロジェクト新規作成
/projects/:id                  プロジェクト詳細（属性・メンバー・進捗・コスト）
/projects/:id/edit             プロジェクト編集
/reports/executive             エグゼクティブレポート
/reports/:id                   プロジェクト別レポート詳細
/admin/users                   ユーザー管理
/admin/roles                   ロール・権限管理
/admin/categories              属性カテゴリ管理
/admin/grade-rates             単価管理
```

### 工数管理UI（worktrack）

全メンバーが日々の工数を入力するシンプルなUI。PMO Dashboardと同一APIを使用。

```
/                        自分の工数カレンダー（週表示）
/projects                担当プロジェクト一覧
/input                   工数入力フォーム
```

---

## n8nワークフロー

### 日次フロー（毎日 10:00）

1. DBから `status='active'` かつ `backlog_project_id` が設定されたプロジェクト一覧を取得
2. Backlog API から各プロジェクトのIssue一覧を取得
3. 取得データを `project_progress` テーブルに保存（`register_day` = 実行日）
4. 過去7日分のデータで進捗分析・リスク検知・トレンド算出を実行
5. 結果を `daily_reports` テーブルに保存
6. AI Agent Node（GPT-4o-mini）でPMO視点の分析コメントを生成・更新

### 週次フロー（毎週月曜 09:00）

1. 直近7件の日次レポートを集約し `weekly_reports` を生成
2. AI Agent Node で週次PMO視点コメントを生成・更新
3. 全プロジェクトの週次レポートを横断集計し `executive_reports` を生成
4. AI Agent Node で経営視点コメントを生成・更新

---

## フェーズ計画

### フェーズ1（本仕様の対象）

- 認証・ACL基盤
- プロジェクト管理（コード発行・属性・メンバーアサイン）
- 工数入力・コスト集計
- Backlog連携（n8n）
- レポート閲覧（日次・週次・エグゼクティブ）
- PMO Dashboard UI
- 工数管理 UI

### フェーズ2

- PMO Agent MCP Server（MySQL データへの Claude アクセス基盤）
- Cowork 上での自然言語プロジェクト分析・経営レポート生成
- AIレビューSkill（プロジェクト計画書の一次レビュー自動化）

---

## 設計上の注意事項

- **プロジェクトは必ずプログラムに属する**。`projects.program_id` は NOT NULL。プログラム未指定でのプロジェクト作成はAPIで 400 を返す。
- **配下プロジェクトが存在するプログラムは削除不可**。DELETE時に配下プロジェクトの存在チェックを行い、存在する場合は 409 を返す（DB側も `ON DELETE RESTRICT`）。
- **プログラムは状態を持たず、予算・期間はDBに保持しない**。APIのレスポンス生成時に配下プロジェクトの集計値（予算＝合計、期間＝最早開始〜最遅終了）として算出する。
- **プロジェクトコードは承認時に発行し、以後変更禁止**。`planning` 中は NULL。承認（active遷移）時にプログラム内の枝番を採番して発行する。Backlog Phase名との整合性を保つため、発行後の UPDATE を API レベルで禁止する。
- **カテゴリ値の削除は論理削除のみ**（`is_active=false`）。物理削除すると過去プロジェクトのアサインレコードが参照エラーになる。
- **単価情報は `pmo_admin` のみ参照可能**。コスト集計APIは権限によって返却フィールドを制御する（PMには合計額のみ）。
- **スコープ制御**（「担当プロジェクトのみ」）はAPIミドルウェアで実装し、フロントエンドの表示制御に依存しない。
- **工数の他人分編集は禁止**。`work_hours` の更新・削除はリクエストユーザーと `user_id` が一致する場合のみ許可。
- **アカウントはPMO管理者が発行**（セルフサインアップなし）。`password_hash IS NULL` を未アクティベート状態とし、パスワード未設定ユーザーのログインは拒否する。
- **ユーザーは論理削除のみ**（`is_active=false`）。物理削除は禁止し、`work_hours` / `project_members` の `user_id` FK は `ON DELETE RESTRICT` で過去の工数・アサイン履歴を保護する。無効化されたユーザーの過去データは引き続き集計・表示対象とする。
- **無効化時はリフレッシュトークンを全失効**させ、既存セッションを次回 `/auth/refresh` で打ち切る（アクセストークンは最長8時間で自然失効）。
- **リフレッシュトークンはローテーション運用**。`/auth/refresh` のたびに新トークンを発行して旧トークンを失効させ、ログアウト時も失効させる。トークンはハッシュ保存・7日有効。
- **初期管理者はマイグレーションで seed**（`password_hash=NULL` + `pmo_admin` ロール、メールは環境ごとに設定）。初期パスワードやトークンはコミットせず、初回の設定リンクは `make seed-link email=<管理者メール>` で都度発行する。マイグレーションで seed する初期データは roles / functions / role_functions と管理者ユーザーに限定する。
- **招待とリセットは同一のワンタイムトークン機構**。トークンはハッシュ保存・単回利用・72時間有効。再発行時は当該ユーザーの未使用トークンを失効させ、有効トークンは常に最新1件のみとする。
- **パスワード設定リンクは管理者が手動共有**（フェーズ1ではメール送信に依存しない）。`/auth/set-password` 系は未認証でアクセス可能だが、有効なトークンがなければ拒否する。セルフサービスの forgot-password は提供しない（リセットは管理者のリンク再発行に一本化）。
