-- Support large run artifacts in Huawei Cloud OBS.
--
-- storage_backend = 'db'  : Content holds the full body (legacy behavior).
-- storage_backend = 'obs' : Content is empty and storage_key names the object.
--
-- The defaults keep every historical row on the inline backend, so no
-- backfill is required. The information_schema checks make this manual SQL
-- migration safe to rerun on supported MySQL/MariaDB deployments.

SET @schema_name = DATABASE();

SET @migration_sql = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = @schema_name
     AND TABLE_NAME = 'run_artifacts'
     AND COLUMN_NAME = 'storage_backend') = 0,
  'ALTER TABLE run_artifacts ADD COLUMN storage_backend VARCHAR(16) NOT NULL DEFAULT ''db''',
  'SELECT 1'
);
PREPARE migration_stmt FROM @migration_sql;
EXECUTE migration_stmt;
DEALLOCATE PREPARE migration_stmt;

SET @migration_sql = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = @schema_name
     AND TABLE_NAME = 'run_artifacts'
     AND COLUMN_NAME = 'storage_key') = 0,
  'ALTER TABLE run_artifacts ADD COLUMN storage_key VARCHAR(512) NOT NULL DEFAULT ''''',
  'SELECT 1'
);
PREPARE migration_stmt FROM @migration_sql;
EXECUTE migration_stmt;
DEALLOCATE PREPARE migration_stmt;
