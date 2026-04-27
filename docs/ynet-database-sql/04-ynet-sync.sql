CREATE TABLE IF NOT EXISTS `space_sync_mapping` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_space_id` bigint NOT NULL COMMENT '源空间ID（测试环境）',
  `target_space_id` bigint NOT NULL COMMENT '目标空间ID（生产环境）',
  `resource_type` varchar(32) NOT NULL COMMENT 'agent/plugin/workflow/variable/space_model/knowledge/document/folder/external_knowledge (document is smallest granularity)',
  `source_resource_id` bigint NOT NULL COMMENT '源资源ID',
  `target_resource_id` bigint NOT NULL COMMENT '目标资源ID',
  `source_updated_at` bigint NOT NULL DEFAULT 0 COMMENT '上次同步时源资源的 updated_at',
  `content_hash` varchar(64) DEFAULT NULL COMMENT '预留字段',
  `created_at` bigint NOT NULL,
  `updated_at` bigint NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_source_target` (`source_space_id`, `target_space_id`, `resource_type`, `source_resource_id`),
  KEY `idx_target` (`target_space_id`, `resource_type`, `target_resource_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='跨环境空间同步ID映射';

CREATE TABLE IF NOT EXISTS `space_sync_history` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_space_id` bigint NOT NULL,
  `target_space_id` bigint NOT NULL,
  `sync_type` varchar(16) NOT NULL COMMENT 'full/incremental',
  `version` varchar(32) DEFAULT NULL COMMENT '本次同步关联的 release version (e.g. v1.0.0)',
  `export_time` bigint NOT NULL COMMENT '导出时间戳',
  `import_time` bigint DEFAULT NULL COMMENT '导入时间戳',
  `statistics` json NOT NULL COMMENT '同步统计',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '0=exported, 1=imported, 2=failed',
  `error_msg` text DEFAULT NULL,
  `package_file_name` varchar(256) DEFAULT NULL,
  `snapshot_key` varchar(512) DEFAULT NULL COMMENT '导入前 target 状态的快照 key (用于回滚)',
  `rollback_from_version` varchar(32) DEFAULT NULL COMMENT '当此条记录是回滚事件时，记录回滚自的版本',
  `created_at` bigint NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_source_target` (`source_space_id`, `target_space_id`, `export_time`),
  KEY `idx_version` (`target_space_id`, `version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='空间同步历史记录';

CREATE TABLE IF NOT EXISTS `space_release` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `space_id` bigint NOT NULL,
  `version` varchar(32) NOT NULL COMMENT '语义化版本号 vX.Y.Z',
  `tag` varchar(128) DEFAULT NULL,
  `description` text,
  `sync_type` varchar(16) NOT NULL COMMENT 'full/incremental',
  `parent_version` varchar(32) DEFAULT NULL COMMENT '父版本，incremental 类型用',
  `manifest` json NOT NULL,
  `statistics` json NOT NULL,
  `package_key` varchar(512) NOT NULL COMMENT '对象存储中 release ZIP 的 key',
  `package_size` bigint NOT NULL DEFAULT 0,
  `content_hash` varchar(64) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'draft' COMMENT 'draft/published',
  `created_by` bigint NOT NULL,
  `published_at` bigint DEFAULT NULL,
  `created_at` bigint NOT NULL,
  `updated_at` bigint NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_space_version` (`space_id`, `version`),
  KEY `idx_space_status` (`space_id`, `status`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='空间发布版本';
