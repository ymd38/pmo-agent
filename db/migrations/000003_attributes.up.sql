-- プロジェクト属性マスタ（EAVパターン）

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

CREATE TABLE project_attribute_assignments (
  id          INT AUTO_INCREMENT PRIMARY KEY,
  project_id  INT NOT NULL,
  category_id INT NOT NULL,
  value_id    INT NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_project_value (project_id, value_id),
  FOREIGN KEY (project_id)  REFERENCES projects(id)               ON DELETE CASCADE,
  FOREIGN KEY (category_id) REFERENCES project_categories(id),
  FOREIGN KEY (value_id)    REFERENCES project_category_values(id)
);
