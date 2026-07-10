ALTER TABLE programs
  DROP KEY uq_type_year_seq,
  DROP COLUMN seq_no,
  DROP COLUMN fiscal_year,
  DROP COLUMN type;
