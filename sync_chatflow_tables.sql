-- ChatFlow Database Tables Sync Script
-- 同步ChatFlow相关的数据库表到远程数据库
-- 数据库: opencoze
-- 执行前请确保已连接到正确的数据库

USE opencoze;

-- Create "app_conversation_template_draft" table
CREATE TABLE IF NOT EXISTS `app_conversation_template_draft` (
  `id` bigint unsigned NOT NULL COMMENT "id",
  `app_id` bigint unsigned NOT NULL COMMENT "app id",
  `space_id` bigint unsigned NOT NULL COMMENT "space id",
  `name` varchar(256) NOT NULL COMMENT "conversation name",
  `template_id` bigint unsigned NOT NULL COMMENT "template id",
  `creator_id` bigint unsigned NOT NULL COMMENT "creator id",
  `created_at` bigint unsigned NOT NULL COMMENT "create time in millisecond",
  `updated_at` bigint unsigned NULL COMMENT "update time in millisecond",
  `deleted_at` datetime(3) NULL COMMENT "delete time in millisecond",
  PRIMARY KEY (`id`),
  INDEX `idx_space_id_app_id_template_id` (`space_id`, `app_id`, `template_id`)
) ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT="应用对话模板草稿表";

-- Create "app_conversation_template_online" table
CREATE TABLE IF NOT EXISTS `app_conversation_template_online` (
  `id` bigint unsigned NOT NULL COMMENT "id",
  `app_id` bigint unsigned NOT NULL COMMENT "app id",
  `space_id` bigint unsigned NOT NULL COMMENT "space id",
  `name` varchar(256) NOT NULL COMMENT "conversation name",
  `template_id` bigint unsigned NOT NULL COMMENT "template id",
  `version` varchar(256) NOT NULL COMMENT "version name",
  `creator_id` bigint unsigned NOT NULL COMMENT "creator id",
  `created_at` bigint unsigned NOT NULL COMMENT "create time in millisecond",
  PRIMARY KEY (`id`),
  INDEX `idx_space_id_app_id_template_id_version` (`space_id`, `app_id`, `template_id`, `version`)
) ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT="应用对话模板在线表";

-- Create "app_dynamic_conversation_draft" table
CREATE TABLE IF NOT EXISTS `app_dynamic_conversation_draft` (
  `id` bigint unsigned NOT NULL COMMENT "id",
  `app_id` bigint unsigned NOT NULL COMMENT "app id",
  `name` varchar(256) NOT NULL COMMENT "conversation name",
  `user_id` bigint unsigned NOT NULL COMMENT "user id",
  `connector_id` bigint unsigned NOT NULL COMMENT "connector id",
  `conversation_id` bigint unsigned NOT NULL COMMENT "conversation id",
  `created_at` bigint unsigned NOT NULL COMMENT "create time in millisecond",
  `deleted_at` datetime(3) NULL COMMENT "delete time in millisecond",
  PRIMARY KEY (`id`),
  INDEX `idx_app_id_connector_id_user_id` (`app_id`, `connector_id`, `user_id`),
  INDEX `idx_connector_id_user_id_name` (`connector_id`, `user_id`, `name`)
) ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT="应用动态对话草稿表";

-- Create "app_dynamic_conversation_online" table
CREATE TABLE IF NOT EXISTS `app_dynamic_conversation_online` (
  `id` bigint unsigned NOT NULL COMMENT "id",
  `app_id` bigint unsigned NOT NULL COMMENT "app id",
  `name` varchar(256) NOT NULL COMMENT "conversation name",
  `user_id` bigint unsigned NOT NULL COMMENT "user id",
  `connector_id` bigint unsigned NOT NULL COMMENT "connector id",
  `conversation_id` bigint unsigned NOT NULL COMMENT "conversation id",
  `created_at` bigint unsigned NOT NULL COMMENT "create time in millisecond",
  `deleted_at` datetime(3) NULL COMMENT "delete time in millisecond",
  PRIMARY KEY (`id`),
  INDEX `idx_app_id_connector_id_user_id` (`app_id`, `connector_id`, `user_id`),
  INDEX `idx_connector_id_user_id_name` (`connector_id`, `user_id`, `name`)
) ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT="应用动态对话在线表";

-- Create "app_static_conversation_draft" table
CREATE TABLE IF NOT EXISTS `app_static_conversation_draft` (
  `id` bigint unsigned NOT NULL COMMENT "id",
  `template_id` bigint unsigned NOT NULL COMMENT "template id",
  `user_id` bigint unsigned NOT NULL COMMENT "user id",
  `connector_id` bigint unsigned NOT NULL COMMENT "connector id",
  `conversation_id` bigint unsigned NOT NULL COMMENT "conversation id",
  `created_at` bigint unsigned NOT NULL COMMENT "create time in millisecond",
  `deleted_at` datetime(3) NULL COMMENT "delete time in millisecond",
  PRIMARY KEY (`id`),
  INDEX `idx_connector_id_user_id_template_id` (`connector_id`, `user_id`, `template_id`)
) ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT="应用静态对话草稿表";

-- Create "app_static_conversation_online" table
CREATE TABLE IF NOT EXISTS `app_static_conversation_online` (
  `id` bigint unsigned NOT NULL COMMENT "id",
  `template_id` bigint unsigned NOT NULL COMMENT "template id",
  `user_id` bigint unsigned NOT NULL COMMENT "user id",
  `connector_id` bigint unsigned NOT NULL COMMENT "connector id",
  `conversation_id` bigint unsigned NOT NULL COMMENT "conversation id",
  `created_at` bigint unsigned NOT NULL COMMENT "create time in millisecond",
  PRIMARY KEY (`id`),
  INDEX `idx_connector_id_user_id_template_id` (`connector_id`, `user_id`, `template_id`)
) ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT="应用静态对话在线表";

-- Create "chat_flow_role_config" table
CREATE TABLE IF NOT EXISTS `chat_flow_role_config` (
  `id` bigint unsigned NOT NULL COMMENT "id",
  `workflow_id` bigint unsigned NOT NULL COMMENT "workflow id",
  `connector_id` bigint unsigned NULL COMMENT "connector id",
  `name` varchar(256) NOT NULL COMMENT "role name",
  `description` mediumtext NOT NULL COMMENT "role description",
  `version` varchar(256) NOT NULL COMMENT "version",
  `avatar` varchar(256) NOT NULL COMMENT "avatar uri",
  `background_image_info` mediumtext NOT NULL COMMENT "background image information, object structure",
  `onboarding_info` mediumtext NOT NULL COMMENT "intro information, object structure",
  `suggest_reply_info` mediumtext NOT NULL COMMENT "user suggestions, object structure",
  `audio_config` mediumtext NOT NULL COMMENT "agent audio config, object structure",
  `user_input_config` varchar(256) NOT NULL COMMENT "user input config, object structure",
  `creator_id` bigint unsigned NOT NULL COMMENT "creator id",
  `created_at` bigint unsigned NOT NULL COMMENT "create time in millisecond",
  `updated_at` bigint unsigned NULL COMMENT "update time in millisecond",
  `deleted_at` datetime(3) NULL COMMENT "delete time in millisecond",
  PRIMARY KEY (`id`),
  INDEX `idx_connector_id_version` (`connector_id`, `version`),
  INDEX `idx_workflow_id_version` (`workflow_id`, `version`)
) ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT="ChatFlow角色配置表";

-- 验证表是否创建成功
SELECT 
  TABLE_NAME,
  TABLE_COMMENT,
  CREATE_TIME
FROM 
  INFORMATION_SCHEMA.TABLES 
WHERE 
  TABLE_SCHEMA = 'opencoze' 
  AND TABLE_NAME IN (
    'app_conversation_template_draft',
    'app_conversation_template_online', 
    'app_dynamic_conversation_draft',
    'app_dynamic_conversation_online',
    'app_static_conversation_draft',
    'app_static_conversation_online',
    'chat_flow_role_config'
  )
ORDER BY TABLE_NAME;

-- 显示成功信息
SELECT 'ChatFlow数据库表同步完成！' as message;