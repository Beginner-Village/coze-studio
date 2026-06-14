-- ============================================================
-- Skill version snapshots
-- Adds a `version` column to `skill` and a `skill_version` table that
-- stores an immutable snapshot of a skill's content for each version.
-- ============================================================

-- Current content version of a skill; starts at 1 and increments on update.
ALTER TABLE `skill` ADD COLUMN `version` bigint(20) NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS `skill_version` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `skill_id` bigint(20) NOT NULL,
  `version` bigint(20) NOT NULL,
  `name` varchar(255) NOT NULL,
  `description` text DEFAULT NULL,
  `prompt` mediumtext DEFAULT NULL,
  `icon_uri` varchar(1024) NOT NULL DEFAULT '',
  `content_hash` varchar(64) NOT NULL DEFAULT '',
  `created_at` bigint(20) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_skill_version` (`skill_id`, `version`),
  KEY `idx_skill_id` (`skill_id`)
) DEFAULT CHARSET=utf8mb4;
