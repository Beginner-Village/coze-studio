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
  `export_time` bigint NOT NULL COMMENT '导出时间戳',
  `import_time` bigint DEFAULT NULL COMMENT '导入时间戳',
  `statistics` json NOT NULL COMMENT '同步统计',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '0=exported, 1=imported, 2=failed',
  `error_msg` text DEFAULT NULL,
  `package_file_name` varchar(256) DEFAULT NULL,
  `created_at` bigint NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_source_target` (`source_space_id`, `target_space_id`, `export_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='空间同步历史记录';
