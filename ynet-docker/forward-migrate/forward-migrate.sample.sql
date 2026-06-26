-- =============================================================================
-- forward-migrate.sample.sql — B 的"样例 / 当前 release 已核对产物"
--   针对 2026-06-26 实测差集（生产 224 v0211 → 当前代码）：缺 12 表 + 5 列。
--   本文件【守卫式列补丁】部分为精确手写、可直接运行、幂等；
--   【缺表】部分由 gen-forward-migrate.sh 自动内联（这里用 SOURCE 引用权威 DDL，便于审阅）。
--   现场用法：mysql -h<host> -u<user> -p <目标库> < forward-migrate.sample.sql
-- =============================================================================
SET NAMES utf8mb4;
SET @OLD_FK := @@FOREIGN_KEY_CHECKS; SET FOREIGN_KEY_CHECKS=0;

-- ============================ 缺表（CREATE TABLE IF NOT EXISTS）================
-- 以下权威 DDL 文件本就以 IF NOT EXISTS 编写，幂等。生成器会把它们内联进自包含产物；
-- 在仓库根目录直接跑本文件时，下列 SOURCE 会就地加载（路径相对当前工作目录）。
--   space_sync_mapping / space_sync_history
SOURCE docs/ynet-database-sql/04-ynet-sync.sql;
--   space_release
SOURCE docs/ynet-database-sql/05-ynet-release.sql;
--   skill_version
SOURCE docs/ynet-database-sql/99-skill-version.sql;
--   skill_publish_marketplace
SOURCE docs/ynet-database-sql/100-skill-publish-marketplace.sql;
--   super_agent_user_memory
SOURCE docs/ynet-database-sql/101-super-agent-user-memory.sql;
--   skill_review_status
SOURCE docs/ynet-database-sql/103-skill-review-status.sql;
--   ai_product / ai_product_version / ai_product_installation / ai_product_audit_log
--   + super_agent_session_runtime_config
SOURCE docs/ynet-database-sql/104-ai-product-foundation.sql;

-- ============================ 缺列（守卫式 ADD COLUMN）========================
-- MySQL 8.4 / OceanBase 均不支持 ADD COLUMN IF NOT EXISTS，故用存储过程守卫。幂等。
DROP PROCEDURE IF EXISTS __add_col;
DELIMITER //
CREATE PROCEDURE __add_col(IN t VARCHAR(128), IN c VARCHAR(128), IN def TEXT)
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                 WHERE table_schema=DATABASE() AND table_name=t AND column_name=c) THEN
    SET @ddl=CONCAT('ALTER TABLE `',t,'` ADD COLUMN `',c,'` ',def);
    PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;
  END IF;
END //
DELIMITER ;

-- 5 个已实测缺列（精确定义，与 docs/ynet-database-sql/102、105 及 docker/migrations 一致）
CALL __add_col('single_agent_draft',  'agent_type',             'VARCHAR(64) DEFAULT NULL COMMENT ''Agent Type for Runtime Routing''');
CALL __add_col('single_agent_version','agent_type',             'VARCHAR(64) DEFAULT NULL COMMENT ''Agent Type for Runtime Routing''');
CALL __add_col('single_agent_draft',  'super_agent_tool_config','json DEFAULT NULL COMMENT ''Super-agent capability switches''');
CALL __add_col('single_agent_draft',  'source_product_id',      'bigint NOT NULL DEFAULT 0 COMMENT ''agent_app product id; 0=normal''');
CALL __add_col('single_agent_draft',  'source_product_version', 'varchar(64) NOT NULL DEFAULT '''' COMMENT ''pinned agent_app product version''');

DROP PROCEDURE __add_col;

-- ============================ known-fixups（站点特有）=========================
-- 行内 CDRCB 等站点的改名/改类型在 known-fixups.sql 维护（守卫幂等），生成器会自动追加。

SET FOREIGN_KEY_CHECKS=@OLD_FK;
-- 校验：SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE(); -- 应含上述 12 张新表
