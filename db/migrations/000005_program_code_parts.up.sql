-- プログラムコードを種別+会計年度+連番に分解し、連番を自動採番できるようにする。
-- code は materialized（type-fiscal_year-seq の0埋め4桁）として維持。

ALTER TABLE programs
  ADD COLUMN type        VARCHAR(8) AFTER code,
  ADD COLUMN fiscal_year SMALLINT   AFTER type,
  ADD COLUMN seq_no      SMALLINT   AFTER fiscal_year;

-- 既存行を code から後方互換でバックフィル（TYPE-YEAR-SEQ 形式を分解）
UPDATE programs SET
  type        = SUBSTRING_INDEX(code, '-', 1),
  fiscal_year = CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(code, '-', 2), '-', -1) AS UNSIGNED),
  seq_no      = CAST(SUBSTRING_INDEX(code, '-', -1) AS UNSIGNED);

ALTER TABLE programs
  MODIFY COLUMN type        VARCHAR(8) NOT NULL,
  MODIFY COLUMN fiscal_year SMALLINT   NOT NULL,
  MODIFY COLUMN seq_no      SMALLINT   NOT NULL,
  ADD UNIQUE KEY uq_type_year_seq (type, fiscal_year, seq_no);
