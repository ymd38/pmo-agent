-- 初期データ（roles / functions / role_functions / 管理者ユーザー / 属性マスタ）
-- 環境非依存の参照データ＋管理者。管理者のメールは環境に合わせて調整すること。

-- ロール
INSERT INTO roles (code, name, description) VALUES
  ('executive', '経営層', 'エグゼクティブレポート・ダッシュボード閲覧専用'),
  ('pmo_admin', 'PMO管理者', '全機能へのアクセス'),
  ('pm', 'プロジェクトマネージャー', '担当プロジェクトの管理・レポート閲覧'),
  ('member', 'メンバー', '担当プロジェクトの工数入力・確認'),
  ('planner', '担当者（起案者）', 'プロジェクト起案・AIレビュー実行');

-- 機能権限
INSERT INTO functions (code, name) VALUES
  ('view_dashboard', 'ダッシュボード閲覧'),
  ('view_executive_report', 'エグゼクティブレポート閲覧'),
  ('view_project_report', 'プロジェクト別レポート閲覧'),
  ('issue_project_code', 'プログラム作成・プロジェクトコード発行'),
  ('manage_projects', 'プロジェクト作成・編集・削除'),
  ('view_project_detail', 'プロジェクト詳細閲覧'),
  ('manage_members', 'ユーザー・単価管理'),
  ('assign_project_members', 'プロジェクトへのメンバーアサイン'),
  ('input_work_hours', '工数入力（自分のみ）'),
  ('view_work_hours', '他メンバーの工数閲覧'),
  ('view_project_cost', 'プロジェクトコスト閲覧'),
  ('manage_categories', 'プロジェクト属性カテゴリ管理'),
  ('manage_grade_rates', 'グレード別単価管理'),
  ('manage_roles', 'ロール・権限管理'),
  ('manage_users', 'ユーザー管理');

-- pmo_admin = 全機能
INSERT INTO role_functions (role_id, function_id)
  SELECT r.id, f.id FROM roles r CROSS JOIN functions f WHERE r.code = 'pmo_admin';

-- executive
INSERT INTO role_functions (role_id, function_id)
  SELECT r.id, f.id FROM roles r JOIN functions f
    ON f.code IN ('view_dashboard', 'view_executive_report')
  WHERE r.code = 'executive';

-- pm（スコープ「担当PJのみ」はミドルウェアで別途強制）
INSERT INTO role_functions (role_id, function_id)
  SELECT r.id, f.id FROM roles r JOIN functions f
    ON f.code IN ('view_dashboard', 'view_project_report', 'manage_projects',
                  'view_project_detail', 'assign_project_members',
                  'input_work_hours', 'view_work_hours', 'view_project_cost')
  WHERE r.code = 'pm';

-- member
INSERT INTO role_functions (role_id, function_id)
  SELECT r.id, f.id FROM roles r JOIN functions f
    ON f.code IN ('view_project_report', 'view_project_detail', 'input_work_hours')
  WHERE r.code = 'member';

-- planner
INSERT INTO role_functions (role_id, function_id)
  SELECT r.id, f.id FROM roles r JOIN functions f
    ON f.code IN ('view_project_detail')
  WHERE r.code = 'planner';

-- 初期管理者（password_hash=NULL の未アクティベート。`make seed-link email=...` で初回設定）
INSERT INTO users (email, name, grade, is_active) VALUES
  ('admin@example.com', '管理者', 'manager', true);

INSERT INTO user_roles (user_id, role_id)
  SELECT u.id, r.id FROM users u JOIN roles r ON r.code = 'pmo_admin'
  WHERE u.email = 'admin@example.com';

-- 属性マスタ初期データ
INSERT INTO project_categories (code, name, description, is_required, sort_order) VALUES
  ('function_area', '機能領域', '会員管理・決済・通知などの機能領域', false, 1),
  ('system', 'システム', '対象システム', false, 2),
  ('project_type', '案件種別', '投資/保守/運用など', true, 3),
  ('dev_method', '開発手法', 'ウォーターフォール/アジャイル等', false, 4);

INSERT INTO project_category_values (category_id, code, label, sort_order)
  SELECT c.id, v.code, v.label, v.sort_order
  FROM project_categories c
  JOIN (
            SELECT 'function_area' AS cat, 'membership'  AS code, '会員管理'         AS label, 1 AS sort_order
  UNION ALL SELECT 'function_area',        'payment',             '決済',               2
  UNION ALL SELECT 'function_area',        'notification',        '通知',               3
  UNION ALL SELECT 'project_type',         'investment',          '投資',               1
  UNION ALL SELECT 'project_type',         'maintenance',         '保守',               2
  UNION ALL SELECT 'project_type',         'operation',           '運用',               3
  UNION ALL SELECT 'dev_method',           'waterfall',           'ウォーターフォール', 1
  UNION ALL SELECT 'dev_method',           'agile',               'アジャイル',         2
  ) v ON v.cat = c.code;
