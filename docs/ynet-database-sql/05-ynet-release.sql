CREATE TABLE IF NOT EXISTS `space_release` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `space_id` bigint NOT NULL COMMENT '所属空间ID',
  `version` varchar(32) NOT NULL COMMENT '版本号 (semver: v1.0.0)',
  `tag` varchar(128) DEFAULT NULL COMMENT '用户自定义标签',
  `description` text DEFAULT NULL COMMENT '发布说明',
  `sync_type` varchar(16) NOT NULL COMMENT 'full/incremental',
  `parent_version` varchar(32) DEFAULT NULL COMMENT '上一个版本号（用于版本链）',
  `manifest` json NOT NULL COMMENT '包内 manifest 快照',
  `statistics` json NOT NULL COMMENT '资源统计',
  `package_key` varchar(512) NOT NULL COMMENT '对象存储永久key',
  `package_size` bigint NOT NULL DEFAULT 0 COMMENT '包大小(bytes)',
  `content_hash` varchar(64) NOT NULL COMMENT '整包 SHA256',
  `status` varchar(16) NOT NULL DEFAULT 'draft' COMMENT 'draft/published/deprecated',
  `created_by` bigint NOT NULL COMMENT '创建人ID',
  `published_at` bigint DEFAULT NULL COMMENT '发布时间戳',
  `created_at` bigint NOT NULL,
  `updated_at` bigint NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_space_version` (`space_id`, `version`),
  KEY `idx_space_status` (`space_id`, `status`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='空间发布版本';

-- ALTER space_sync_history for version management
-- Run each statement separately; skip if column already exists
ALTER TABLE `space_sync_history` ADD COLUMN `version` varchar(32) DEFAULT NULL COMMENT '关联的版本号' AFTER `sync_type`;
ALTER TABLE `space_sync_history` ADD COLUMN `snapshot_key` varchar(512) DEFAULT NULL COMMENT '导入前快照的对象存储key' AFTER `package_file_name`;
ALTER TABLE `space_sync_history` ADD COLUMN `rollback_from_version` varchar(32) DEFAULT NULL COMMENT '回滚前的版本号' AFTER `snapshot_key`;
ALTER TABLE `space_sync_history` ADD KEY `idx_version` (`target_space_id`, `version`);
