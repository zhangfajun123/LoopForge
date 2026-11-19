-- Support large run artifacts in Huawei Cloud OBS.
--
-- storage_backend = 'db'  : Content holds the full body (legacy behavior).
-- storage_backend = 'obs' : Content is empty and storage_key names the object.
--
-- The defaults keep every historical row on the inline backend, so no
-- backfill is required. MySQL 8 supports ADD COLUMN IF NOT EXISTS only in
-- some forks; this script is written to be re-runnable by operators.

ALTER TABLE run_artifacts
  ADD COLUMN storage_backend VARCHAR(16) NOT NULL DEFAULT 'db',
  ADD COLUMN storage_key     VARCHAR(512) NOT NULL DEFAULT '';