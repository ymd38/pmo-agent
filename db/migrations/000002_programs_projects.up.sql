-- プログラム（PMO集計単位／コードプレフィックスの名前空間）とプロジェクト（実作業単位）

CREATE TABLE programs (
  id          INT AUTO_INCREMENT PRIMARY KEY,
  code        VARCHAR(100) NOT NULL UNIQUE,            -- 種別-年度-連番（例: INV-2026-0001）。作成時に確定
  name        VARCHAR(255) NOT NULL,
  description TEXT,
  created_by  INT NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (created_by) REFERENCES users(id)
);

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
  backlog_project_id VARCHAR(255),
  ai_review_doc_path TEXT,
  created_by         INT NOT NULL,
  created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_program_branch (program_id, branch_no),
  FOREIGN KEY (program_id)  REFERENCES programs(id) ON DELETE RESTRICT,
  FOREIGN KEY (pm_id)       REFERENCES users(id) ON DELETE SET NULL,
  FOREIGN KEY (approver_id) REFERENCES users(id) ON DELETE SET NULL,
  FOREIGN KEY (created_by)  REFERENCES users(id)
);
