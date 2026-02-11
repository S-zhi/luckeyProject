-- Add file_name columns for file-name-driven path resolution.
SET @stmt := (
  SELECT IF(
    (
      SELECT COUNT(*)
      FROM information_schema.COLUMNS
      WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = 'models'
        AND COLUMN_NAME = 'file_name'
    ) = 0,
    "ALTER TABLE models ADD COLUMN file_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'standard file name (without dir)' AFTER model_path",
    'SELECT 1'
  )
);
PREPARE s1 FROM @stmt;
EXECUTE s1;
DEALLOCATE PREPARE s1;

SET @stmt := (
  SELECT IF(
    (
      SELECT COUNT(*)
      FROM information_schema.COLUMNS
      WHERE TABLE_SCHEMA = DATABASE()
        AND TABLE_NAME = 'datasets'
        AND COLUMN_NAME = 'file_name'
    ) = 0,
    "ALTER TABLE datasets ADD COLUMN file_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'standard file name (without dir)' AFTER dataset_path",
    'SELECT 1'
  )
);
PREPARE s2 FROM @stmt;
EXECUTE s2;
DEALLOCATE PREPARE s2;

-- Backfill from legacy path columns.
UPDATE models
SET file_name = SUBSTRING_INDEX(REPLACE(model_path, '\\\\', '/'), '/', -1)
WHERE TRIM(COALESCE(file_name, '')) = ''
  AND TRIM(COALESCE(model_path, '')) <> '';

UPDATE datasets
SET file_name = SUBSTRING_INDEX(REPLACE(dataset_path, '\\\\', '/'), '/', -1)
WHERE TRIM(COALESCE(file_name, '')) = ''
  AND TRIM(COALESCE(dataset_path, '')) <> '';

-- Secondary fallback for records without valid file_name.
UPDATE models
SET file_name = CONCAT(REPLACE(TRIM(name), ' ', '_'), '.bin')
WHERE TRIM(COALESCE(file_name, '')) = '';

UPDATE datasets
SET file_name = CONCAT(REPLACE(TRIM(name), ' ', '_'), '.zip')
WHERE TRIM(COALESCE(file_name, '')) = '';
