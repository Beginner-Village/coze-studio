-- known-fixups.sql — 站点特有、纯"只增"表达不了的迁移（改名/改类型）
-- 规则：每条都用 information_schema 守卫，幂等、可重复执行；绝不无条件 DROP。
-- 由 gen-forward-migrate.sh 追加到 forward-migrate-<ver>.sql 末尾。

DROP PROCEDURE IF EXISTS __rename_col;
DELIMITER //
-- 仅当 old 列在、new 列不在时，才把 old 改名为 new（带新定义）。否则跳过。
CREATE PROCEDURE __rename_col(IN t VARCHAR(128), IN oldc VARCHAR(128), IN newc VARCHAR(128), IN def TEXT)
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns
            WHERE table_schema=DATABASE() AND table_name=t AND column_name=oldc)
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns
            WHERE table_schema=DATABASE() AND table_name=t AND column_name=newc) THEN
    SET @ddl=CONCAT('ALTER TABLE `',t,'` CHANGE COLUMN `',oldc,'` `',newc,'` ',def);
    PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;
  END IF;
END //
DELIMITER ;

-- 例：行内 CDRCB 历史漂移 system_setting.`key` → setting_key（避 GaussDB/OB 保留字）
-- CALL __rename_col('system_setting','key','setting_key','VARCHAR(128) NOT NULL');

DROP PROCEDURE __rename_col;
