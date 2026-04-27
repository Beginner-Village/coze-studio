-- ============================================================
-- Idempotent migrations for space_sync_mapping
-- Apply on every deploy: existing installs get the schema fix,
-- fresh installs are no-op (the new index is already in 04-ynet-sync.sql).
--
-- DEPLOYMENT NOTE: ynet-docker/deploy-v2/deploy.sh must run this file on
-- every deploy. The block to add (next to the existing 04-ynet-sync.sql
-- conditional import) is:
--
--   if [ -f "$SQL_DIR/04b-ynet-sync-migrate.sql" ]; then
--     import_sql "ynet-studio" "$SQL_DIR/04b-ynet-sync-migrate.sql" \
--       && ok "Sync 迁移已应用" || warn "Sync 迁移失败 (可能已就位)"
--   fi
-- ============================================================

-- 2026-04-27 (B5): drop legacy uk_source (source_space_id, resource_type, source_resource_id)
-- which made the same source resource share a single mapping row across all
-- targets, causing 1-source-N-target imports to overwrite each other instead
-- of recording one row per target. Replace with uk_source_target.

SET @drop_stmt = (
  SELECT IF(
    EXISTS (SELECT 1 FROM information_schema.STATISTICS
            WHERE TABLE_SCHEMA = DATABASE()
              AND TABLE_NAME = 'space_sync_mapping'
              AND INDEX_NAME = 'uk_source'),
    'ALTER TABLE space_sync_mapping DROP INDEX uk_source',
    'SELECT 1'
  )
);
PREPARE _stmt FROM @drop_stmt;
EXECUTE _stmt;
DEALLOCATE PREPARE _stmt;

SET @add_stmt = (
  SELECT IF(
    NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS
                WHERE TABLE_SCHEMA = DATABASE()
                  AND TABLE_NAME = 'space_sync_mapping'
                  AND INDEX_NAME = 'uk_source_target'),
    'ALTER TABLE space_sync_mapping ADD UNIQUE KEY uk_source_target (source_space_id, target_space_id, resource_type, source_resource_id)',
    'SELECT 1'
  )
);
PREPARE _stmt FROM @add_stmt;
EXECUTE _stmt;
DEALLOCATE PREPARE _stmt;
