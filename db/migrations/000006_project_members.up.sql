-- プロジェクトメンバーアサイン。担当PJスコープ制御（pm/member=担当PJ）の解決元。
-- user_id FK は ON DELETE RESTRICT でユーザー論理削除時のアサイン履歴を保護する。

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
  INDEX idx_member_user (user_id),                     -- スコープ解決（担当PJ集合）の逆引き用
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id)    REFERENCES users(id)    ON DELETE RESTRICT
);
