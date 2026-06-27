-- ============================================================
-- Skill publish scope and marketplace visibility
-- Adds publish metadata to skill so a standard skill package can be
-- published inside one space or globally to all spaces.
-- ============================================================

ALTER TABLE `skill` ADD COLUMN `publish_scope` tinyint(4) NOT NULL DEFAULT 1 COMMENT '1 private, 2 space, 3 global';
ALTER TABLE `skill` ADD COLUMN `published_version` bigint(20) NOT NULL DEFAULT 0 COMMENT 'skill version pinned at publish time';
ALTER TABLE `skill` ADD COLUMN `published_at` bigint(20) NOT NULL DEFAULT 0 COMMENT 'publish timestamp in milliseconds';
ALTER TABLE `skill` ADD COLUMN `published_by` bigint(20) NOT NULL DEFAULT 0 COMMENT 'publisher user id';

CREATE INDEX `idx_skill_marketplace_scope` ON `skill` (`publish_scope`, `published_at`);
CREATE INDEX `idx_skill_marketplace_space_scope` ON `skill` (`space_id`, `publish_scope`, `published_at`);
