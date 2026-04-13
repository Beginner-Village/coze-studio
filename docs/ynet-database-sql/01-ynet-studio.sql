-- ============================================================
-- Ynet Studio Database Schema
-- Source: 10.10.10.220:3306 / opencoze -> openynet
-- Exported: 2026-03-17
-- ============================================================
CREATE DATABASE IF NOT EXISTS `openynet` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `openynet`;

CREATE TABLE IF NOT EXISTS `admin_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` bigint unsigned NOT NULL COMMENT '关联的用户ID',
  `role` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'admin' COMMENT '角色: super_admin/admin',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '创建时间（毫秒）',
  `created_by` bigint unsigned NOT NULL DEFAULT '0' COMMENT '创建者ID',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '更新时间（毫秒）',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间（毫秒）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_id` (`user_id`,`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='管理员用户表';

CREATE TABLE IF NOT EXISTS `agent_conversation_mapping` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `chatflow_conversation_id` bigint NOT NULL COMMENT 'ChatFlow会话ID',
  `node_id` varchar(64) NOT NULL COMMENT '工作流节点ID',
  `platform` varchar(32) NOT NULL COMMENT 'Agent平台类型: hiagent, singleagent, dify, ynet',
  `agent_conversation_id` varchar(128) NOT NULL COMMENT 'Agent平台的会话ID',
  `agent_id` bigint DEFAULT NULL COMMENT 'Agent ID (for singleagent platform)',
  `agent_config` json DEFAULT NULL COMMENT 'Agent配置快照',
  `message_count` int DEFAULT '0' COMMENT '消息数量',
  `last_message_id` varchar(128) DEFAULT NULL COMMENT '最后一条消息ID',
  `last_message_time` timestamp NULL DEFAULT NULL COMMENT '最后消息时间',
  `status` tinyint DEFAULT '1' COMMENT '状态: 1-活跃, 0-已结束',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_chatflow_node` (`chatflow_conversation_id`,`node_id`,`deleted_at`),
  KEY `idx_agent_conv` (`platform`,`agent_conversation_id`),
  KEY `idx_chatflow_id` (`chatflow_conversation_id`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_status_updated` (`status`,`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Agent会话映射表，管理ChatFlow与各Agent平台的会话关系';

CREATE TABLE IF NOT EXISTS `agent_to_database` (
  `id` bigint unsigned NOT NULL COMMENT 'ID',
  `agent_id` bigint unsigned NOT NULL COMMENT 'Agent ID',
  `database_id` bigint unsigned NOT NULL COMMENT 'ID of database_info',
  `is_draft` tinyint(1) NOT NULL COMMENT 'Is draft',
  `prompt_disable` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Support prompt calls: 1 not supported, 0 supported',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_agent_db_draft` (`agent_id`,`database_id`,`is_draft`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='agent_to_database info';

CREATE TABLE IF NOT EXISTS `agent_tool_draft` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Primary Key ID',
  `agent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Agent ID',
  `plugin_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Plugin ID',
  `tool_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tool ID',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `sub_url` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Sub URL Path',
  `method` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'HTTP Request Method',
  `tool_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Tool Name',
  `tool_version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Tool Version, e.g. v1.0.0',
  `operation` json DEFAULT NULL COMMENT 'Tool Openapi Operation Schema',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_idx_agent_tool_id` (`agent_id`,`tool_id`),
  UNIQUE KEY `uniq_idx_agent_tool_name` (`agent_id`,`tool_name`),
  KEY `idx_agent_plugin_tool` (`agent_id`,`plugin_id`,`tool_id`),
  KEY `idx_agent_tool_bind` (`agent_id`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Draft Agent Tool';

CREATE TABLE IF NOT EXISTS `agent_tool_version` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Primary Key ID',
  `agent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Agent ID',
  `plugin_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Plugin ID',
  `tool_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tool ID',
  `agent_version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Agent Tool Version',
  `tool_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Tool Name',
  `tool_version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Tool Version, e.g. v1.0.0',
  `sub_url` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Sub URL Path',
  `method` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'HTTP Request Method',
  `operation` json DEFAULT NULL COMMENT 'Tool Openapi Operation Schema',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_idx_agent_tool_id_agent_version` (`agent_id`,`tool_id`,`agent_version`),
  UNIQUE KEY `uniq_idx_agent_tool_name_agent_version` (`agent_id`,`tool_name`,`agent_version`),
  KEY `idx_agent_tool_id_created_at` (`agent_id`,`tool_id`,`created_at`),
  KEY `idx_agent_tool_name_created_at` (`agent_id`,`tool_name`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent Tool Version';

CREATE TABLE IF NOT EXISTS `api_key` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary Key ID',
  `api_key` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'API Key hash',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'API Key Name',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '0 normal, 1 deleted',
  `user_id` bigint NOT NULL DEFAULT '0' COMMENT 'API Key Owner',
  `expired_at` bigint NOT NULL DEFAULT '0' COMMENT 'API Key Expired Time',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `last_used_at` bigint NOT NULL DEFAULT '0' COMMENT 'Used Time in Milliseconds',
  `ak_type` tinyint NOT NULL DEFAULT '0' COMMENT 'api key type ',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=7605050360977162241 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='api key table';

CREATE TABLE IF NOT EXISTS `api_token` (
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `tenant_id` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `token` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `dialog_id` varchar(32) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `source` varchar(16) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `beta` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`tenant_id`,`token`),
  KEY `apitoken_create_time` (`create_time`),
  KEY `apitoken_create_date` (`create_date`),
  KEY `apitoken_update_time` (`update_time`),
  KEY `apitoken_update_date` (`update_date`),
  KEY `apitoken_tenant_id` (`tenant_id`),
  KEY `apitoken_token` (`token`),
  KEY `apitoken_dialog_id` (`dialog_id`),
  KEY `apitoken_source` (`source`),
  KEY `apitoken_beta` (`beta`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `app_connector_release_ref` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Primary Key',
  `record_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Publish Record ID',
  `connector_id` bigint unsigned DEFAULT NULL COMMENT 'Publish Connector ID',
  `publish_config` json DEFAULT NULL COMMENT 'Publish Configuration',
  `publish_status` tinyint NOT NULL DEFAULT '0' COMMENT 'Publish Status',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_record_connector` (`record_id`,`connector_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Connector Release Record Reference';

CREATE TABLE IF NOT EXISTS `app_conversation_template_draft` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `app_id` bigint unsigned NOT NULL COMMENT 'app id',
  `space_id` bigint unsigned NOT NULL COMMENT 'space id',
  `name` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'conversation name',
  `template_id` bigint unsigned NOT NULL COMMENT 'template id',
  `creator_id` bigint unsigned NOT NULL COMMENT 'creator id',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  `updated_at` bigint unsigned DEFAULT NULL COMMENT 'update time in millisecond',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'delete time in millisecond',
  PRIMARY KEY (`id`),
  KEY `idx_space_id_app_id_template_id` (`space_id`,`app_id`,`template_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `app_conversation_template_online` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `app_id` bigint unsigned NOT NULL COMMENT 'app id',
  `space_id` bigint unsigned NOT NULL COMMENT 'space id',
  `name` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'conversation name',
  `template_id` bigint unsigned NOT NULL COMMENT 'template id',
  `version` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'version name',
  `creator_id` bigint unsigned NOT NULL COMMENT 'creator id',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  PRIMARY KEY (`id`),
  KEY `idx_space_id_app_id_template_id_version` (`space_id`,`app_id`,`template_id`,`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `app_draft` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'APP ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Space ID',
  `owner_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Owner ID',
  `icon_uri` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Icon URI',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Application Name',
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Application Description',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Draft Application';

CREATE TABLE IF NOT EXISTS `app_dynamic_conversation_draft` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `app_id` bigint unsigned NOT NULL COMMENT 'app id',
  `name` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'conversation name',
  `user_id` bigint unsigned NOT NULL COMMENT 'user id',
  `connector_id` bigint unsigned NOT NULL COMMENT 'connector id',
  `conversation_id` bigint unsigned NOT NULL COMMENT 'conversation id',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'delete time in millisecond',
  PRIMARY KEY (`id`),
  KEY `idx_app_id_connector_id_user_id` (`app_id`,`connector_id`,`user_id`),
  KEY `idx_connector_id_user_id_name` (`connector_id`,`user_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `app_dynamic_conversation_online` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `app_id` bigint unsigned NOT NULL COMMENT 'app id',
  `name` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'conversation name',
  `user_id` bigint unsigned NOT NULL COMMENT 'user id',
  `connector_id` bigint unsigned NOT NULL COMMENT 'connector id',
  `conversation_id` bigint unsigned NOT NULL COMMENT 'conversation id',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'delete time in millisecond',
  PRIMARY KEY (`id`),
  KEY `idx_app_id_connector_id_user_id` (`app_id`,`connector_id`,`user_id`),
  KEY `idx_connector_id_user_id_name` (`connector_id`,`user_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `app_release_record` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Publish Record ID',
  `app_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Application ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Space ID',
  `owner_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Owner ID',
  `icon_uri` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Icon URI',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Application Name',
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Application Description',
  `connector_ids` json DEFAULT NULL COMMENT 'Publish Connector IDs',
  `extra_info` json DEFAULT NULL COMMENT 'Publish Extra Info',
  `version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Release Version',
  `version_desc` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Version Description',
  `publish_status` tinyint NOT NULL DEFAULT '0' COMMENT 'Publish Status',
  `publish_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Publish Time in Milliseconds',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_idx_app_version_connector` (`app_id`,`version`),
  KEY `idx_app_publish_at` (`app_id`,`publish_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Application Release Record';

CREATE TABLE IF NOT EXISTS `app_static_conversation_draft` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `template_id` bigint unsigned NOT NULL COMMENT 'template id',
  `user_id` bigint unsigned NOT NULL COMMENT 'user id',
  `connector_id` bigint unsigned NOT NULL COMMENT 'connector id',
  `conversation_id` bigint unsigned NOT NULL COMMENT 'conversation id',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'delete time in millisecond',
  PRIMARY KEY (`id`),
  KEY `idx_connector_id_user_id_template_id` (`connector_id`,`user_id`,`template_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `app_static_conversation_online` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `template_id` bigint unsigned NOT NULL COMMENT 'template id',
  `user_id` bigint unsigned NOT NULL COMMENT 'user id',
  `connector_id` bigint unsigned NOT NULL COMMENT 'connector id',
  `conversation_id` bigint unsigned NOT NULL COMMENT 'conversation id',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  PRIMARY KEY (`id`),
  KEY `idx_connector_id_user_id_template_id` (`connector_id`,`user_id`,`template_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `chat_flow_role_config` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `workflow_id` bigint unsigned NOT NULL COMMENT 'workflow id',
  `connector_id` bigint unsigned DEFAULT NULL COMMENT 'connector id',
  `name` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'role name',
  `description` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'role description',
  `version` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'version',
  `avatar` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'avatar uri',
  `background_image_info` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'background image information, object structure',
  `onboarding_info` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'intro information, object structure',
  `suggest_reply_info` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'user suggestions, object structure',
  `audio_config` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'agent audio config, object structure',
  `user_input_config` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'user input config, object structure',
  `creator_id` bigint unsigned NOT NULL COMMENT 'creator id',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  `updated_at` bigint unsigned DEFAULT NULL COMMENT 'update time in millisecond',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'delete time in millisecond',
  PRIMARY KEY (`id`),
  KEY `idx_connector_id_version` (`connector_id`,`version`),
  KEY `idx_workflow_id_version` (`workflow_id`,`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `chatflow_conversation_history` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary Key ID',
  `conversation_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Conversation identifier',
  `workflow_id` bigint unsigned NOT NULL COMMENT 'Chatflow workflow ID',
  `space_id` bigint unsigned NOT NULL COMMENT 'Space ID',
  `user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'User ID who initiated the conversation',
  `role` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Message role: user, assistant, system',
  `content` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Message content',
  `execution_id` bigint unsigned DEFAULT NULL COMMENT 'Workflow execution ID that generated this message',
  `node_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Node ID that generated this message',
  `message_order` int unsigned NOT NULL DEFAULT '0' COMMENT 'Message order in conversation',
  `metadata` json DEFAULT NULL COMMENT 'Additional metadata like tokens, model info, etc.',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  PRIMARY KEY (`id`),
  KEY `idx_conversation_name_workflow_order` (`conversation_name`,`workflow_id`,`message_order`),
  KEY `idx_conversation_name_created_at` (`conversation_name`,`created_at`),
  KEY `idx_workflow_id_conversation_name` (`workflow_id`,`conversation_name`),
  KEY `idx_space_id_created_at` (`space_id`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Chatflow conversation history table';

CREATE TABLE IF NOT EXISTS `connector_workflow_version` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `app_id` bigint unsigned NOT NULL COMMENT 'app id',
  `connector_id` bigint unsigned NOT NULL COMMENT 'connector id',
  `workflow_id` bigint unsigned NOT NULL COMMENT 'workflow id',
  `version` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'version',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_connector_id_workflow_id_version` (`connector_id`,`workflow_id`,`version`),
  KEY `idx_connector_id_workflow_id_create_at` (`connector_id`,`workflow_id`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='connector workflow version';

CREATE TABLE IF NOT EXISTS `conversation` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '' COMMENT 'conversation name',
  `connector_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Publish Connector ID',
  `agent_id` bigint NOT NULL DEFAULT '0' COMMENT 'agent_id',
  `scene` tinyint NOT NULL DEFAULT '0' COMMENT 'conversation scene',
  `section_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'section_id',
  `creator_id` bigint unsigned DEFAULT '0' COMMENT 'creator_id',
  `ext` text COLLATE utf8mb4_unicode_ci COMMENT 'ext',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT 'status: 1-normal 2-deleted',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  PRIMARY KEY (`id`),
  KEY `idx_connector_bot_status` (`connector_id`,`agent_id`,`creator_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7605050399304712193 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='conversation info record';

CREATE TABLE IF NOT EXISTS `data_copy_task` (
  `master_task_id` varchar(128) COLLATE utf8mb4_general_ci DEFAULT '' COMMENT 'task id',
  `origin_data_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'origin data id',
  `target_data_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'target data id',
  `origin_space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'origin space id',
  `target_space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'target space id',
  `origin_user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'origin user id',
  `target_user_id` bigint unsigned DEFAULT '0' COMMENT 'target user id',
  `origin_app_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'origin app id',
  `target_app_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'target app id',
  `data_type` tinyint unsigned NOT NULL DEFAULT '0' COMMENT 'data type 1:knowledge, 2:database',
  `ext_info` varchar(255) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'ext',
  `start_time` bigint DEFAULT '0' COMMENT 'task start time',
  `finish_time` bigint DEFAULT NULL COMMENT 'task finish time',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '1: Create 2: Running 3: Success 4: Failure',
  `error_msg` varchar(128) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'error msg',
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_master_task_id_origin_data_id_data_type` (`master_task_id`,`origin_data_id`,`data_type`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='data copy task record';

CREATE TABLE IF NOT EXISTS `draft_database_info` (
  `id` bigint unsigned NOT NULL COMMENT 'ID',
  `app_id` bigint unsigned DEFAULT NULL COMMENT 'App ID',
  `space_id` bigint unsigned NOT NULL COMMENT 'Space ID',
  `related_online_id` bigint unsigned NOT NULL COMMENT 'The primary key ID of online_database_info table',
  `is_visible` tinyint NOT NULL DEFAULT '1' COMMENT 'Visibility: 0 invisible, 1 visible',
  `prompt_disabled` tinyint NOT NULL DEFAULT '0' COMMENT 'Support prompt calls: 1 not supported, 0 supported',
  `table_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'Table name',
  `table_desc` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'Table description',
  `table_field` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'Table field info',
  `creator_id` bigint NOT NULL DEFAULT '0' COMMENT 'Creator ID',
  `icon_uri` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'Icon Uri',
  `physical_table_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'The name of the real physical table',
  `rw_mode` bigint NOT NULL DEFAULT '1' COMMENT 'Read and write permission modes: 1. Limited read and write mode 2. Read-only mode 3. Full read and write mode',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time',
  PRIMARY KEY (`id`),
  KEY `idx_space_app_creator_deleted` (`space_id`,`app_id`,`creator_id`,`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='draft database info';

CREATE TABLE IF NOT EXISTS `external_agent_config` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `space_id` bigint NOT NULL COMMENT '空间ID',
  `name` varchar(255) NOT NULL COMMENT '智能体名称',
  `description` text COMMENT '智能体描述',
  `platform` varchar(32) NOT NULL COMMENT '平台类型: hiagent, dify, ynet',
  `agent_url` varchar(512) NOT NULL COMMENT 'API接口地址',
  `agent_key` varchar(512) DEFAULT NULL COMMENT 'API密钥(加密存储)',
  `agent_id` varchar(128) DEFAULT NULL COMMENT '外部平台的Agent ID',
  `app_id` varchar(128) DEFAULT NULL COMMENT 'Dify应用ID',
  `icon` varchar(255) DEFAULT 'robot' COMMENT '图标',
  `category` varchar(64) DEFAULT 'external' COMMENT '分类',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态: 0-禁用, 1-启用',
  `metadata` json DEFAULT NULL COMMENT '其他扩展配置',
  `created_by` bigint NOT NULL COMMENT '创建者用户ID',
  `updated_by` bigint DEFAULT NULL COMMENT '最后更新者用户ID',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_space_name_platform` (`space_id`,`name`,`platform`,`deleted_at`),
  KEY `idx_space_id` (`space_id`),
  KEY `idx_platform` (`platform`),
  KEY `idx_status` (`status`),
  KEY `idx_created_by` (`created_by`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='外部智能体配置表';

CREATE TABLE IF NOT EXISTS `external_knowledge_binding` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '用户ID，全局唯一标识',
  `binding_key` varchar(500) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '绑定密钥，用于连接外部知识库',
  `binding_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '绑定名称，用户自定义名称',
  `binding_type` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'default' COMMENT '绑定类型，预留字段用于支持多种知识库类型',
  `extra_config` json DEFAULT NULL COMMENT '额外配置信息，JSON格式存储',
  `status` tinyint(1) NOT NULL DEFAULT '1' COMMENT '状态，0=禁用，1=启用',
  `last_sync_at` timestamp NULL DEFAULT NULL COMMENT '最后同步时间',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_binding_key` (`user_id`,`binding_key`) COMMENT '用户和绑定密钥组合唯一索引',
  KEY `idx_user_id` (`user_id`) COMMENT '用户ID索引',
  KEY `idx_status` (`status`) COMMENT '状态索引',
  KEY `idx_binding_type` (`binding_type`) COMMENT '绑定类型索引',
  KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引'
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='外部知识库绑定表，存储用户的外部知识库连接信息';

CREATE TABLE IF NOT EXISTS `files` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'file name',
  `file_size` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'file size',
  `tos_uri` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'TOS URI',
  `status` tinyint unsigned NOT NULL DEFAULT '0' COMMENT 'status，0invalid，1valid',
  `comment` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'file comment',
  `source` tinyint unsigned NOT NULL DEFAULT '0' COMMENT 'source：1 from API,',
  `creator_id` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'creator id',
  `content_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'content type',
  `ynet_account_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'ynet account id',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'Delete Time',
  PRIMARY KEY (`id`),
  KEY `idx_creator_id` (`creator_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='file resource table';

CREATE TABLE IF NOT EXISTS `folder` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Folder ID',
  `space_id` bigint unsigned NOT NULL COMMENT 'Space ID',
  `parent_id` bigint unsigned DEFAULT NULL COMMENT 'Parent Folder ID, NULL for root folder',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Folder Name',
  `description` varchar(1000) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Folder Description',
  `creator_id` bigint unsigned NOT NULL COMMENT 'Creator ID',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'Delete Time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_space_parent_name_deleted` (`space_id`,`parent_id`,`name`,`deleted_at`),
  KEY `idx_space_id_parent_id_deleted_at` (`space_id`,`parent_id`,`deleted_at`),
  KEY `idx_creator_id` (`creator_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Folder Table for Library Resource Organization';

CREATE TABLE IF NOT EXISTS `hi_agent` (
  `agent_id` varchar(64) NOT NULL COMMENT '智能体ID',
  `space_id` bigint NOT NULL COMMENT '空间ID',
  `name` varchar(255) NOT NULL COMMENT '智能体名称',
  `description` text COMMENT '描述',
  `icon_url` varchar(500) DEFAULT NULL COMMENT '图标URL',
  `endpoint` varchar(500) NOT NULL COMMENT 'API端点',
  `auth_type` enum('bearer','api_key') NOT NULL COMMENT '认证类型',
  `api_key` text COMMENT 'API密钥（加密存储）',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：0-禁用，1-启用',
  `meta` json DEFAULT NULL COMMENT '额外元数据',
  `created_at` bigint NOT NULL COMMENT '创建时间',
  `updated_at` bigint NOT NULL COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`agent_id`),
  KEY `idx_space_id` (`space_id`),
  KEY `idx_deleted_at` (`deleted_at`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='HiAgent外部智能体表';

CREATE TABLE IF NOT EXISTS `knowledge` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `name` varchar(150) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'knowledge_s name',
  `app_id` bigint NOT NULL DEFAULT '0' COMMENT 'app id',
  `creator_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'creator id',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'space id',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'Delete Time',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '0 initialization, 1 effective, 2 invalid',
  `description` text COLLATE utf8mb4_unicode_ci COMMENT 'description',
  `icon_uri` varchar(150) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'icon uri',
  `format_type` tinyint NOT NULL DEFAULT '0' COMMENT '0: Text 1: Table 2: Images',
  PRIMARY KEY (`id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_creator_id` (`creator_id`),
  KEY `idx_space_id_deleted_at_updated_at` (`space_id`,`deleted_at`,`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='knowledge tabke';

CREATE TABLE IF NOT EXISTS `knowledge_backup_20250812` (
  `id` bigint unsigned NOT NULL COMMENT '主键ID',
  `name` varchar(150) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '名称',
  `app_id` bigint NOT NULL DEFAULT '0' COMMENT '项目ID，标识该资源是否是项目独有',
  `creator_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间ID',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'Delete Time in Milliseconds',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '0 初始化, 1 生效 2 失效',
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '描述',
  `icon_uri` varchar(150) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '头像uri',
  `format_type` tinyint NOT NULL DEFAULT '0' COMMENT '0:文本 1:表格 2:图片'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `knowledge_document` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `knowledge_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'knowledge id',
  `name` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'document name',
  `file_extension` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '0' COMMENT 'Document type, txt/pdf/csv etc..',
  `document_type` int NOT NULL DEFAULT '0' COMMENT 'Document type: 0: Text 1: Table 2: Image',
  `uri` text COLLATE utf8mb4_unicode_ci COMMENT 'uri',
  `size` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'document size',
  `slice_count` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'slice count',
  `char_count` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'number of characters',
  `creator_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'creator id',
  `space_id` bigint NOT NULL DEFAULT '0' COMMENT 'space id',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'Delete Time',
  `source_type` int DEFAULT '0' COMMENT '0: Local file upload, 2: Custom text, 103: Feishu 104: Lark',
  `status` int NOT NULL DEFAULT '0' COMMENT 'status',
  `fail_reason` text COLLATE utf8mb4_unicode_ci COMMENT 'fail reason',
  `parse_rule` json DEFAULT NULL COMMENT 'parse rule',
  `table_info` json DEFAULT NULL COMMENT 'table info',
  PRIMARY KEY (`id`),
  KEY `idx_creator_id` (`creator_id`),
  KEY `idx_knowledge_id_deleted_at_updated_at` (`knowledge_id`,`deleted_at`,`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='knowledge document info';

CREATE TABLE IF NOT EXISTS `knowledge_document_review` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `knowledge_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'knowledge id',
  `space_id` bigint NOT NULL DEFAULT '0' COMMENT 'space id',
  `name` varchar(150) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'name',
  `type` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '0' COMMENT 'document type',
  `uri` text COLLATE utf8mb4_unicode_ci COMMENT 'uri',
  `format_type` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '0 text, 1 table, 2 images',
  `status` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '0 Processing 1 Completed 2 Failed 3 Expired',
  `chunk_resp_uri` text COLLATE utf8mb4_unicode_ci COMMENT 'pre-sliced uri',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'Delete Time',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `creator_id` bigint NOT NULL DEFAULT '0' COMMENT 'creator id',
  PRIMARY KEY (`id`),
  KEY `idx_dataset_id` (`knowledge_id`,`status`,`updated_at`),
  KEY `idx_uri` (`uri`(100))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Document slice preview info';

CREATE TABLE IF NOT EXISTS `knowledge_document_slice` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `knowledge_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'knowledge id',
  `document_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'document_id',
  `content` text COLLATE utf8mb4_unicode_ci COMMENT 'content',
  `answer` text COLLATE utf8mb4_unicode_ci COMMENT 'answer content for QA format',
  `sequence` decimal(20,5) NOT NULL COMMENT 'slice sequence number, starting from 1',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'Delete Time',
  `creator_id` bigint NOT NULL DEFAULT '0' COMMENT 'creator id',
  `space_id` bigint NOT NULL DEFAULT '0' COMMENT 'space id',
  `status` int NOT NULL DEFAULT '0' COMMENT 'status',
  `fail_reason` text COLLATE utf8mb4_unicode_ci COMMENT 'fail reason',
  `hit` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'hit counts ',
  PRIMARY KEY (`id`),
  KEY `idx_document_id_deleted_at_sequence` (`document_id`,`deleted_at`,`sequence`),
  KEY `idx_knowledge_id_document_id` (`knowledge_id`,`document_id`),
  KEY `idx_sequence` (`sequence`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='knowledge document slice';

CREATE TABLE IF NOT EXISTS `marketplace_bot` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `bot_id` bigint NOT NULL COMMENT '智能体ID',
  `bot_version` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '智能体版本',
  `category_id` int NOT NULL COMMENT '分类ID: 1-效率工具,2-商业服务,3-文本创作,4-学习教育,5-代码助手,6-生活方式,7-游戏,8-图像与音视频,9-角色',
  `category_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '分类名称',
  `title` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '商店显示标题',
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '商店显示描述',
  `icon_url` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '图标URL',
  `publisher_id` bigint NOT NULL COMMENT '发布者用户ID',
  `publisher_name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '发布者名称',
  `space_id` bigint NOT NULL COMMENT '所属空间ID',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态: 0-下架,1-上架,2-审核中',
  `view_count` int DEFAULT '0' COMMENT '查看次数',
  `use_count` int DEFAULT '0' COMMENT '使用次数',
  `favorite_count` int DEFAULT '0' COMMENT '收藏次数',
  `is_featured` tinyint DEFAULT '0' COMMENT '是否精选推荐',
  `is_official` tinyint DEFAULT '0' COMMENT '是否官方认证',
  `tags` json DEFAULT NULL COMMENT '标签列表',
  `extra_info` json DEFAULT NULL COMMENT '扩展信息',
  `publish_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '发布时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_bot_id` (`bot_id`),
  KEY `idx_category_status` (`category_id`,`status`),
  KEY `idx_publisher_id` (`publisher_id`),
  KEY `idx_space_id` (`space_id`),
  KEY `idx_publish_time` (`publish_time`),
  KEY `idx_view_count` (`view_count`),
  KEY `idx_use_count` (`use_count`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='智能体商店表';

CREATE TABLE IF NOT EXISTS `marketplace_bot_favorite` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `bot_id` bigint NOT NULL COMMENT '智能体ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '收藏时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_bot` (`user_id`,`bot_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_bot_id` (`bot_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户收藏表';

CREATE TABLE IF NOT EXISTS `marketplace_plugin_tools` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `plugin_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Marketplace Plugin ID (关联 plugin_marketplace.id)',
  `stable_tool_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '稳定的工具ID (100000-999999范围)',
  `operation_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'OpenAPI operationId',
  `tool_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '工具名称',
  `method` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'HTTP Request Method',
  `sub_url` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Sub URL Path',
  `operation` json DEFAULT NULL COMMENT 'Tool OpenAPI Operation Schema (与tool表格式一致)',
  `pkg_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '外部插件包名',
  `pkg_version` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '外部插件版本',
  `plugin_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '外部插件名称',
  `external_tool_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '外部工具名称',
  `activated_status` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '0:activated; 1:deactivated',
  `version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'v1.0.0' COMMENT 'Tool Version',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `operation_json` text COLLATE utf8mb4_unicode_ci COMMENT '完整的OpenAPI操作JSON',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_stable_tool_id` (`stable_tool_id`),
  UNIQUE KEY `uk_plugin_operation` (`plugin_id`,`operation_id`),
  KEY `idx_plugin_id` (`plugin_id`),
  KEY `idx_tool_name` (`tool_name`),
  KEY `idx_activated_status` (`activated_status`),
  KEY `idx_pkg_info` (`pkg_name`,`pkg_version`,`plugin_name`),
  KEY `idx_external_tool` (`external_tool_name`),
  KEY `idx_stable_tool_id` (`stable_tool_id`),
  KEY `idx_plugin_operation` (`plugin_id`,`operation_id`)
) ENGINE=InnoDB AUTO_INCREMENT=206 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Marketplace Plugin Tools Mapping Table';

CREATE TABLE IF NOT EXISTS `marketplace_sync_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `plugin_id` bigint unsigned NOT NULL COMMENT '插件ID',
  `plugin_name` varchar(255) NOT NULL COMMENT '插件名称',
  `action` varchar(50) NOT NULL COMMENT '操作类型',
  `status` varchar(50) NOT NULL COMMENT '状态',
  `message` text COMMENT '消息',
  `created_at` bigint unsigned NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_plugin_id` (`plugin_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='插件同步日志表';

CREATE TABLE IF NOT EXISTS `message` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `run_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'run_id',
  `conversation_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'conversation id',
  `user_id` varchar(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'user id',
  `agent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'agent_id',
  `role` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'role: user、assistant、system',
  `content_type` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'content type 1 text',
  `content` mediumtext COLLATE utf8mb4_unicode_ci COMMENT 'content',
  `message_type` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'message_type',
  `display_content` text COLLATE utf8mb4_unicode_ci COMMENT 'display content',
  `ext` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'message ext',
  `section_id` bigint unsigned DEFAULT NULL COMMENT 'section_id',
  `broken_position` int DEFAULT '-1' COMMENT 'broken position',
  `status` tinyint unsigned NOT NULL DEFAULT '0' COMMENT 'message status: 1 Available 2 Deleted 3 Replaced 4 Broken 5 Failed 6 Streaming 7 Pending',
  `model_content` mediumtext COLLATE utf8mb4_unicode_ci COMMENT 'model content',
  `meta_info` text COLLATE utf8mb4_unicode_ci COMMENT 'text tagging information such as citation and highlighting',
  `reasoning_content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'reasoning content',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  PRIMARY KEY (`id`),
  KEY `idx_conversation_id` (`conversation_id`),
  KEY `idx_run_id` (`run_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7600000000000000000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='message record';

CREATE TABLE IF NOT EXISTS `model_entity` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `meta_id` bigint unsigned NOT NULL COMMENT 'model metadata id',
  `name` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'name',
  `description` text COLLATE utf8mb4_unicode_ci COMMENT 'description',
  `default_params` json DEFAULT NULL COMMENT 'default params',
  `scenario` bigint unsigned NOT NULL COMMENT 'scenario',
  `status` int NOT NULL DEFAULT '1' COMMENT 'model status',
  `is_public` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否为公共模型: 0=私有, 1=公共',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT 'Delete Time',
  PRIMARY KEY (`id`),
  KEY `idx_scenario` (`scenario`),
  KEY `idx_status` (`status`),
  KEY `idx_is_public` (`is_public`)
) ENGINE=InnoDB AUTO_INCREMENT=257066926260359169 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Model information';

CREATE TABLE IF NOT EXISTS `model_meta` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `model_name` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'model name',
  `protocol` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'model protocol',
  `icon_uri` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Icon URI',
  `capability` json DEFAULT NULL COMMENT 'capability',
  `conn_config` json DEFAULT NULL COMMENT 'model conn config',
  `status` int NOT NULL DEFAULT '1' COMMENT 'model status',
  `is_public` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否为公共模型: 0=私有, 1=公共',
  `description` varchar(2048) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'description',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT 'Delete Time',
  `icon_url` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Icon URL',
  PRIMARY KEY (`id`),
  KEY `idx_status` (`status`),
  KEY `idx_is_public` (`is_public`)
) ENGINE=InnoDB AUTO_INCREMENT=257066926163890177 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Model metadata';

CREATE TABLE IF NOT EXISTS `model_template` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `provider` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '模型提供商',
  `model_name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '模型名称',
  `model_type` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '模型类型(llm/embedding/rerank/..)',
  `template` json NOT NULL COMMENT '模型模版，json格式',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '创建时间',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_provider` (`provider`)
) ENGINE=InnoDB AUTO_INCREMENT=1758269967783240 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='model_template';

CREATE TABLE IF NOT EXISTS `node_execution` (
  `id` bigint unsigned NOT NULL COMMENT 'node execution id',
  `execute_id` bigint unsigned NOT NULL COMMENT 'the workflow execute id this node execution belongs to',
  `node_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'node key',
  `node_name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'name of the node',
  `node_type` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'the type of the node, in string',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  `status` tinyint unsigned NOT NULL COMMENT '1=waiting 2=running 3=success 4=fail',
  `duration` bigint unsigned DEFAULT NULL COMMENT 'execution duration in millisecond',
  `input` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'actual input of the node',
  `output` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'actual output of the node',
  `raw_output` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'the original output of the node',
  `error_info` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'error info',
  `error_level` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'level of the error',
  `input_tokens` bigint unsigned DEFAULT NULL COMMENT 'number of input tokens',
  `output_tokens` bigint unsigned DEFAULT NULL COMMENT 'number of output tokens',
  `updated_at` bigint unsigned DEFAULT NULL COMMENT 'update time in millisecond',
  `composite_node_index` bigint unsigned DEFAULT NULL COMMENT 'loop or batch_s execution index',
  `composite_node_items` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'the items extracted from parent composite node for this index',
  `parent_node_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'when as inner node for loop or batch, this is the parent node_s key',
  `sub_execute_id` bigint unsigned DEFAULT NULL COMMENT 'if this node is sub_workflow, the exe id of the sub workflow',
  `extra` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'extra info',
  PRIMARY KEY (`id`),
  KEY `idx_execute_id_node_id` (`execute_id`,`node_id`),
  KEY `idx_execute_id_parent_node_id` (`execute_id`,`parent_node_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Node run record, used to record the status information of each node during each workflow execution';

CREATE TABLE IF NOT EXISTS `online_database_info` (
  `id` bigint unsigned NOT NULL COMMENT 'ID',
  `app_id` bigint unsigned DEFAULT NULL COMMENT 'App ID',
  `space_id` bigint unsigned NOT NULL COMMENT 'Space ID',
  `related_draft_id` bigint unsigned NOT NULL COMMENT 'The primary key ID of draft_database_info table',
  `is_visible` tinyint NOT NULL DEFAULT '1' COMMENT 'Visibility: 0 invisible, 1 visible',
  `prompt_disabled` tinyint NOT NULL DEFAULT '0' COMMENT 'Support prompt calls: 1 not supported, 0 supported',
  `table_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'Table name',
  `table_desc` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'Table description',
  `table_field` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'Table field info',
  `creator_id` bigint NOT NULL DEFAULT '0' COMMENT 'Creator ID',
  `icon_uri` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'Icon Uri',
  `physical_table_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'The name of the real physical table',
  `rw_mode` bigint NOT NULL DEFAULT '1' COMMENT 'Read and write permission modes: 1. Limited read and write mode 2. Read-only mode 3. Full read and write mode',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time',
  PRIMARY KEY (`id`),
  KEY `idx_space_app_creator_deleted` (`space_id`,`app_id`,`creator_id`,`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='online database info';

CREATE TABLE IF NOT EXISTS `plugin` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Plugin ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Space ID',
  `developer_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Developer ID',
  `app_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Application ID',
  `icon_uri` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Icon URI',
  `server_url` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Server URL',
  `plugin_type` tinyint NOT NULL DEFAULT '0' COMMENT 'Plugin Type, 1:http, 6:local',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Plugin Version, e.g. v1.0.0',
  `version_desc` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Plugin Version Description',
  `manifest` json DEFAULT NULL COMMENT 'Plugin Manifest',
  `openapi_doc` json DEFAULT NULL COMMENT 'OpenAPI Document, only stores the root',
  PRIMARY KEY (`id`),
  KEY `idx_space_created_at` (`space_id`,`created_at`),
  KEY `idx_space_updated_at` (`space_id`,`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Latest Plugin';

CREATE TABLE IF NOT EXISTS `plugin_backup_20250812` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Plugin ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Space ID',
  `developer_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Developer ID',
  `app_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Application ID',
  `icon_uri` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Icon URI',
  `server_url` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Server URL',
  `plugin_type` tinyint NOT NULL DEFAULT '0' COMMENT 'Plugin Type, 1:http, 6:local',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Plugin Version, e.g. v1.0.0',
  `version_desc` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Plugin Version Description',
  `manifest` json DEFAULT NULL COMMENT 'Plugin Manifest',
  `openapi_doc` json DEFAULT NULL COMMENT 'OpenAPI Document, only stores the root'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plugin_draft` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Plugin ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Space ID',
  `developer_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Developer ID',
  `app_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Application ID',
  `icon_uri` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Icon URI',
  `server_url` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Server URL',
  `plugin_type` tinyint NOT NULL DEFAULT '0' COMMENT 'Plugin Type, 1:http, 6:local',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time',
  `manifest` json DEFAULT NULL COMMENT 'Plugin Manifest',
  `openapi_doc` json DEFAULT NULL COMMENT 'OpenAPI Document, only stores the root',
  PRIMARY KEY (`id`),
  KEY `idx_app_id` (`app_id`,`id`),
  KEY `idx_space_app_created_at` (`space_id`,`app_id`,`created_at`),
  KEY `idx_space_app_updated_at` (`space_id`,`app_id`,`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Draft Plugin';

CREATE TABLE IF NOT EXISTS `plugin_marketplace` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '商店插件ID',
  `template_plugin_id` bigint unsigned DEFAULT NULL COMMENT '关联的模板插件ID（可选）',
  `product_id` bigint unsigned NOT NULL COMMENT '商店产品ID（唯一标识）',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '插件名称',
  `description` text COLLATE utf8mb4_unicode_ci COMMENT '插件描述',
  `icon_url` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '图标URL',
  `server_url` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '服务器URL',
  `version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '版本号 e.g. v1.0.0',
  `version_desc` text COLLATE utf8mb4_unicode_ci COMMENT '版本描述',
  `plugin_type` tinyint NOT NULL DEFAULT '1' COMMENT '插件类型：1=HTTP API, 6=本地插件',
  `manifest` json DEFAULT NULL COMMENT '插件清单（JSON格式）',
  `openapi_doc` json DEFAULT NULL COMMENT 'OpenAPI文档（JSON格式）',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：0=下架, 1=上架',
  `is_official` tinyint NOT NULL DEFAULT '0' COMMENT '是否官方插件：0=否, 1=是',
  `is_free` tinyint NOT NULL DEFAULT '1' COMMENT '是否免费：0=付费, 1=免费',
  `heat_score` int NOT NULL DEFAULT '0' COMMENT '热度评分',
  `favorite_count` int NOT NULL DEFAULT '0' COMMENT '收藏数量',
  `usage_count` int NOT NULL DEFAULT '0' COMMENT '使用次数',
  `listed_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '上架时间（毫秒时间戳）',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '创建时间（毫秒时间戳）',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '更新时间（毫秒时间戳）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_product_id` (`product_id`),
  KEY `idx_template_plugin_id` (`template_plugin_id`),
  KEY `idx_status_official` (`status`,`is_official`),
  KEY `idx_listed_at` (`listed_at`),
  KEY `idx_heat_score` (`heat_score`),
  KEY `idx_usage_count` (`usage_count`)
) ENGINE=InnoDB AUTO_INCREMENT=48 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='插件商店主表';

CREATE TABLE IF NOT EXISTS `plugin_oauth_auth` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Primary Key',
  `user_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'User ID',
  `plugin_id` bigint NOT NULL DEFAULT '0' COMMENT 'Plugin ID',
  `is_draft` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Is Draft Plugin',
  `oauth_config` json DEFAULT NULL COMMENT 'Authorization Code OAuth Config',
  `access_token` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Access Token',
  `refresh_token` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Refresh Token',
  `token_expired_at` bigint DEFAULT NULL COMMENT 'Token Expired in Milliseconds',
  `next_token_refresh_at` bigint DEFAULT NULL COMMENT 'Next Token Refresh Time in Milliseconds',
  `last_active_at` bigint DEFAULT NULL COMMENT 'Last active time in Milliseconds',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_idx_user_plugin_is_draft` (`user_id`,`plugin_id`,`is_draft`),
  KEY `idx_last_active_at` (`last_active_at`),
  KEY `idx_last_token_expired_at` (`token_expired_at`),
  KEY `idx_next_token_refresh_at` (`next_token_refresh_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Plugin OAuth Authorization Code Info';

CREATE TABLE IF NOT EXISTS `plugin_version` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Primary Key ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Space ID',
  `developer_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Developer ID',
  `plugin_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Plugin ID',
  `app_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Application ID',
  `icon_uri` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Icon URI',
  `server_url` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Server URL',
  `plugin_type` tinyint NOT NULL DEFAULT '0' COMMENT 'Plugin Type, 1:http, 6:local',
  `version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Plugin Version, e.g. v1.0.0',
  `version_desc` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Plugin Version Description',
  `manifest` json DEFAULT NULL COMMENT 'Plugin Manifest',
  `openapi_doc` json DEFAULT NULL COMMENT 'OpenAPI Document, only stores the root',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_idx_plugin_version` (`plugin_id`,`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Plugin Version';

CREATE TABLE IF NOT EXISTS `prompt_resource` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `space_id` bigint NOT NULL COMMENT 'space id',
  `name` varchar(255) NOT NULL COMMENT 'name',
  `description` varchar(255) NOT NULL COMMENT 'description',
  `prompt_text` mediumtext COMMENT 'prompt text',
  `status` int NOT NULL COMMENT 'status, 0 is invalid, 1 is valid',
  `creator_id` bigint NOT NULL COMMENT 'creator id',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  PRIMARY KEY (`id`),
  KEY `idx_creator_id` (`creator_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7539813868940296203 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='prompt_resource';

CREATE TABLE IF NOT EXISTS `ragflow_api_4_conversation` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `dialog_id` varchar(32) NOT NULL,
  `user_id` varchar(255) NOT NULL,
  `message` longtext,
  `reference` longtext,
  `tokens` int NOT NULL,
  `source` varchar(16) DEFAULT NULL,
  `dsl` longtext,
  `duration` float NOT NULL,
  `round` int NOT NULL,
  `thumb_up` int NOT NULL,
  `errors` text,
  PRIMARY KEY (`id`),
  KEY `api4conversation_create_time` (`create_time`),
  KEY `api4conversation_create_date` (`create_date`),
  KEY `api4conversation_update_time` (`update_time`),
  KEY `api4conversation_update_date` (`update_date`),
  KEY `api4conversation_dialog_id` (`dialog_id`),
  KEY `api4conversation_user_id` (`user_id`),
  KEY `api4conversation_source` (`source`),
  KEY `api4conversation_duration` (`duration`),
  KEY `api4conversation_round` (`round`),
  KEY `api4conversation_thumb_up` (`thumb_up`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_canvas_template` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `avatar` text,
  `title` longtext,
  `description` longtext,
  `canvas_type` varchar(32) DEFAULT NULL,
  `dsl` longtext,
  `canvas_category` varchar(32) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `canvastemplate_create_time` (`create_time`),
  KEY `canvastemplate_create_date` (`create_date`),
  KEY `canvastemplate_update_time` (`update_time`),
  KEY `canvastemplate_update_date` (`update_date`),
  KEY `canvastemplate_canvas_type` (`canvas_type`),
  KEY `canvas_template_canvas_category` (`canvas_category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_conversation` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `dialog_id` varchar(32) NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `message` longtext,
  `reference` longtext,
  `user_id` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `conversation_create_time` (`create_time`),
  KEY `conversation_create_date` (`create_date`),
  KEY `conversation_update_time` (`update_time`),
  KEY `conversation_update_date` (`update_date`),
  KEY `conversation_dialog_id` (`dialog_id`),
  KEY `conversation_name` (`name`),
  KEY `conversation_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_dialog` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `tenant_id` varchar(32) NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `description` text,
  `icon` text,
  `language` varchar(32) DEFAULT NULL,
  `llm_id` varchar(128) NOT NULL,
  `llm_setting` longtext NOT NULL,
  `prompt_type` varchar(16) NOT NULL,
  `prompt_config` longtext NOT NULL,
  `meta_data_filter` longtext,
  `similarity_threshold` float NOT NULL,
  `vector_similarity_weight` float NOT NULL,
  `top_n` int NOT NULL,
  `top_k` int NOT NULL,
  `do_refer` varchar(1) NOT NULL,
  `rerank_id` varchar(128) NOT NULL,
  `kb_ids` longtext NOT NULL,
  `status` varchar(1) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `dialog_create_time` (`create_time`),
  KEY `dialog_create_date` (`create_date`),
  KEY `dialog_update_time` (`update_time`),
  KEY `dialog_update_date` (`update_date`),
  KEY `dialog_tenant_id` (`tenant_id`),
  KEY `dialog_name` (`name`),
  KEY `dialog_language` (`language`),
  KEY `dialog_prompt_type` (`prompt_type`),
  KEY `dialog_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_document` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `thumbnail` text,
  `kb_id` varchar(256) NOT NULL,
  `parser_id` varchar(32) NOT NULL,
  `parser_config` longtext NOT NULL,
  `source_type` varchar(128) NOT NULL,
  `type` varchar(32) NOT NULL,
  `created_by` varchar(32) NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `location` varchar(255) DEFAULT NULL,
  `size` int NOT NULL,
  `token_num` int NOT NULL,
  `chunk_num` int NOT NULL,
  `progress` float NOT NULL,
  `progress_msg` text,
  `process_begin_at` datetime DEFAULT NULL,
  `process_duration` float NOT NULL,
  `meta_fields` longtext,
  `suffix` varchar(32) NOT NULL,
  `run` varchar(1) DEFAULT NULL,
  `status` varchar(1) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `document_create_time` (`create_time`),
  KEY `document_create_date` (`create_date`),
  KEY `document_update_time` (`update_time`),
  KEY `document_update_date` (`update_date`),
  KEY `document_kb_id` (`kb_id`),
  KEY `document_parser_id` (`parser_id`),
  KEY `document_source_type` (`source_type`),
  KEY `document_type` (`type`),
  KEY `document_created_by` (`created_by`),
  KEY `document_name` (`name`),
  KEY `document_location` (`location`),
  KEY `document_size` (`size`),
  KEY `document_token_num` (`token_num`),
  KEY `document_chunk_num` (`chunk_num`),
  KEY `document_progress` (`progress`),
  KEY `document_process_begin_at` (`process_begin_at`),
  KEY `document_suffix` (`suffix`),
  KEY `document_run` (`run`),
  KEY `document_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_file` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `parent_id` varchar(32) NOT NULL,
  `tenant_id` varchar(32) NOT NULL,
  `created_by` varchar(32) NOT NULL,
  `name` varchar(255) NOT NULL,
  `location` varchar(255) DEFAULT NULL,
  `size` int NOT NULL,
  `type` varchar(32) NOT NULL,
  `source_type` varchar(128) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `file_create_time` (`create_time`),
  KEY `file_create_date` (`create_date`),
  KEY `file_update_time` (`update_time`),
  KEY `file_update_date` (`update_date`),
  KEY `file_parent_id` (`parent_id`),
  KEY `file_tenant_id` (`tenant_id`),
  KEY `file_created_by` (`created_by`),
  KEY `file_name` (`name`),
  KEY `file_location` (`location`),
  KEY `file_size` (`size`),
  KEY `file_type` (`type`),
  KEY `file_source_type` (`source_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_file2document` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `file_id` varchar(32) DEFAULT NULL,
  `document_id` varchar(32) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `file2document_create_time` (`create_time`),
  KEY `file2document_create_date` (`create_date`),
  KEY `file2document_update_time` (`update_time`),
  KEY `file2document_update_date` (`update_date`),
  KEY `file2document_file_id` (`file_id`),
  KEY `file2document_document_id` (`document_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_invitation_code` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `code` varchar(32) NOT NULL,
  `visit_time` datetime DEFAULT NULL,
  `user_id` varchar(32) DEFAULT NULL,
  `tenant_id` varchar(32) DEFAULT NULL,
  `status` varchar(1) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `invitationcode_create_time` (`create_time`),
  KEY `invitationcode_create_date` (`create_date`),
  KEY `invitationcode_update_time` (`update_time`),
  KEY `invitationcode_update_date` (`update_date`),
  KEY `invitationcode_code` (`code`),
  KEY `invitationcode_visit_time` (`visit_time`),
  KEY `invitationcode_user_id` (`user_id`),
  KEY `invitationcode_tenant_id` (`tenant_id`),
  KEY `invitationcode_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_kb_api_tokens` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `kb_id` varchar(32) NOT NULL,
  `tenant_id` varchar(32) NOT NULL,
  `token` varchar(255) NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `description` text,
  `permissions` longtext NOT NULL,
  `status` varchar(16) NOT NULL,
  `expires_at` datetime DEFAULT NULL,
  `created_by` varchar(32) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `knowledgebaseapitoken_token` (`token`),
  KEY `knowledgebaseapitoken_create_time` (`create_time`),
  KEY `knowledgebaseapitoken_create_date` (`create_date`),
  KEY `knowledgebaseapitoken_update_time` (`update_time`),
  KEY `knowledgebaseapitoken_update_date` (`update_date`),
  KEY `knowledgebaseapitoken_kb_id` (`kb_id`),
  KEY `knowledgebaseapitoken_tenant_id` (`tenant_id`),
  KEY `knowledgebaseapitoken_kb_id_token` (`kb_id`,`token`),
  KEY `knowledgebaseapitoken_tenant_id_token` (`tenant_id`,`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_knowledgebase` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `avatar` text,
  `tenant_id` varchar(32) NOT NULL,
  `name` varchar(128) NOT NULL,
  `language` varchar(32) DEFAULT NULL,
  `description` text,
  `embd_id` varchar(128) NOT NULL,
  `permission` varchar(16) NOT NULL,
  `created_by` varchar(32) NOT NULL,
  `doc_num` int NOT NULL,
  `token_num` int NOT NULL,
  `chunk_num` int NOT NULL,
  `similarity_threshold` float NOT NULL,
  `vector_similarity_weight` float NOT NULL,
  `parser_id` varchar(32) NOT NULL,
  `parser_config` longtext NOT NULL,
  `pagerank` int NOT NULL,
  `status` varchar(1) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `knowledgebase_create_time` (`create_time`),
  KEY `knowledgebase_create_date` (`create_date`),
  KEY `knowledgebase_update_time` (`update_time`),
  KEY `knowledgebase_update_date` (`update_date`),
  KEY `knowledgebase_tenant_id` (`tenant_id`),
  KEY `knowledgebase_name` (`name`),
  KEY `knowledgebase_language` (`language`),
  KEY `knowledgebase_embd_id` (`embd_id`),
  KEY `knowledgebase_permission` (`permission`),
  KEY `knowledgebase_created_by` (`created_by`),
  KEY `knowledgebase_doc_num` (`doc_num`),
  KEY `knowledgebase_token_num` (`token_num`),
  KEY `knowledgebase_chunk_num` (`chunk_num`),
  KEY `knowledgebase_similarity_threshold` (`similarity_threshold`),
  KEY `knowledgebase_vector_similarity_weight` (`vector_similarity_weight`),
  KEY `knowledgebase_parser_id` (`parser_id`),
  KEY `knowledgebase_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_llm` (
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `llm_name` varchar(128) NOT NULL,
  `model_type` varchar(128) NOT NULL,
  `fid` varchar(128) NOT NULL,
  `max_tokens` int NOT NULL,
  `tags` varchar(255) NOT NULL,
  `is_tools` tinyint(1) NOT NULL,
  `status` varchar(1) DEFAULT NULL,
  PRIMARY KEY (`fid`,`llm_name`),
  KEY `llm_create_time` (`create_time`),
  KEY `llm_create_date` (`create_date`),
  KEY `llm_update_time` (`update_time`),
  KEY `llm_update_date` (`update_date`),
  KEY `llm_llm_name` (`llm_name`),
  KEY `llm_model_type` (`model_type`),
  KEY `llm_fid` (`fid`),
  KEY `llm_tags` (`tags`),
  KEY `llm_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_llm_factories` (
  `name` varchar(128) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `logo` text,
  `tags` varchar(255) NOT NULL,
  `status` varchar(1) DEFAULT NULL,
  PRIMARY KEY (`name`),
  KEY `llmfactories_create_time` (`create_time`),
  KEY `llmfactories_create_date` (`create_date`),
  KEY `llmfactories_update_time` (`update_time`),
  KEY `llmfactories_update_date` (`update_date`),
  KEY `llmfactories_tags` (`tags`),
  KEY `llmfactories_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_mcp_server` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `name` varchar(255) NOT NULL,
  `tenant_id` varchar(32) NOT NULL,
  `url` varchar(2048) NOT NULL,
  `server_type` varchar(32) NOT NULL,
  `description` text,
  `variables` longtext,
  `headers` longtext,
  PRIMARY KEY (`id`),
  KEY `mcpserver_create_time` (`create_time`),
  KEY `mcpserver_create_date` (`create_date`),
  KEY `mcpserver_update_time` (`update_time`),
  KEY `mcpserver_update_date` (`update_date`),
  KEY `mcpserver_tenant_id` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_search` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `avatar` text,
  `tenant_id` varchar(32) NOT NULL,
  `name` varchar(128) NOT NULL,
  `description` text,
  `created_by` varchar(32) NOT NULL,
  `search_config` longtext NOT NULL,
  `status` varchar(1) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `search_create_time` (`create_time`),
  KEY `search_create_date` (`create_date`),
  KEY `search_update_time` (`update_time`),
  KEY `search_update_date` (`update_date`),
  KEY `search_tenant_id` (`tenant_id`),
  KEY `search_name` (`name`),
  KEY `search_created_by` (`created_by`),
  KEY `search_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_session_map` (
  `id` int NOT NULL AUTO_INCREMENT,
  `ynet_user_id` bigint NOT NULL,
  `ragflow_user_id` varchar(32) NOT NULL,
  `session_key` varchar(255) NOT NULL,
  `ragflow_token` varchar(255) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `expires_at` timestamp NULL DEFAULT NULL,
  `is_active` tinyint(1) DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_session` (`session_key`),
  KEY `idx_ynet_user` (`ynet_user_id`),
  KEY `idx_ragflow_user` (`ragflow_user_id`),
  KEY `idx_session_key` (`session_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Session mapping between RAGFlow and Ynet';

CREATE TABLE IF NOT EXISTS `ragflow_task` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `doc_id` varchar(32) NOT NULL,
  `from_page` int NOT NULL,
  `to_page` int NOT NULL,
  `task_type` varchar(32) NOT NULL,
  `priority` int NOT NULL,
  `begin_at` datetime DEFAULT NULL,
  `process_duration` float NOT NULL,
  `progress` float NOT NULL,
  `progress_msg` text,
  `retry_count` int NOT NULL,
  `digest` text,
  `chunk_ids` longtext,
  PRIMARY KEY (`id`),
  KEY `task_create_time` (`create_time`),
  KEY `task_create_date` (`create_date`),
  KEY `task_update_time` (`update_time`),
  KEY `task_update_date` (`update_date`),
  KEY `task_doc_id` (`doc_id`),
  KEY `task_begin_at` (`begin_at`),
  KEY `task_progress` (`progress`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_tenant_config` (
  `tenant_id` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '租户ID，对应user表的MD5(id)',
  `name` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '租户名称',
  `llm_id` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '默认LLM模型ID',
  `embd_id` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '默认嵌入模型ID',
  `asr_id` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '默认ASR模型ID',
  `img2txt_id` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '默认图像识别模型ID',
  `rerank_id` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '默认重排序模型ID',
  `tts_id` varchar(256) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '默认TTS模型ID',
  `parser_ids` varchar(256) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '文档处理器',
  `description` text COLLATE utf8mb4_unicode_ci COMMENT '租户描述',
  `settings` text COLLATE utf8mb4_unicode_ci COMMENT '其他配置信息(JSON格式)',
  `create_time` bigint DEFAULT NULL COMMENT '创建时间戳',
  `create_date` datetime DEFAULT NULL COMMENT '创建日期',
  `update_time` bigint DEFAULT NULL COMMENT '更新时间戳',
  `update_date` datetime DEFAULT NULL COMMENT '更新日期',
  PRIMARY KEY (`tenant_id`),
  KEY `idx_name` (`name`),
  KEY `idx_llm_id` (`llm_id`),
  KEY `idx_embd_id` (`embd_id`),
  KEY `idx_asr_id` (`asr_id`),
  KEY `idx_img2txt_id` (`img2txt_id`),
  KEY `idx_rerank_id` (`rerank_id`),
  KEY `idx_tts_id` (`tts_id`),
  KEY `idx_parser_ids` (`parser_ids`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='租户配置表';

CREATE TABLE IF NOT EXISTS `ragflow_tenant_langfuse` (
  `tenant_id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `secret_key` varchar(2048) NOT NULL,
  `public_key` varchar(2048) NOT NULL,
  `host` varchar(128) NOT NULL,
  PRIMARY KEY (`tenant_id`),
  KEY `tenantlangfuse_create_time` (`create_time`),
  KEY `tenantlangfuse_create_date` (`create_date`),
  KEY `tenantlangfuse_update_time` (`update_time`),
  KEY `tenantlangfuse_update_date` (`update_date`),
  KEY `tenantlangfuse_secret_key` (`secret_key`(768)),
  KEY `tenantlangfuse_public_key` (`public_key`(768)),
  KEY `tenantlangfuse_host` (`host`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_tenant_llm` (
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `tenant_id` varchar(32) NOT NULL,
  `llm_factory` varchar(128) NOT NULL,
  `model_type` varchar(128) DEFAULT NULL,
  `llm_name` varchar(128) NOT NULL,
  `api_key` varchar(2048) DEFAULT NULL,
  `api_base` varchar(255) DEFAULT NULL,
  `max_tokens` int NOT NULL,
  `used_tokens` int NOT NULL,
  PRIMARY KEY (`tenant_id`,`llm_factory`,`llm_name`),
  KEY `tenantllm_create_time` (`create_time`),
  KEY `tenantllm_create_date` (`create_date`),
  KEY `tenantllm_update_time` (`update_time`),
  KEY `tenantllm_update_date` (`update_date`),
  KEY `tenantllm_tenant_id` (`tenant_id`),
  KEY `tenantllm_llm_factory` (`llm_factory`),
  KEY `tenantllm_model_type` (`model_type`),
  KEY `tenantllm_llm_name` (`llm_name`),
  KEY `tenantllm_api_key` (`api_key`(768)),
  KEY `tenantllm_max_tokens` (`max_tokens`),
  KEY `tenantllm_used_tokens` (`used_tokens`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_user_canvas` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `avatar` text,
  `user_id` varchar(255) NOT NULL,
  `title` varchar(255) DEFAULT NULL,
  `permission` varchar(16) NOT NULL,
  `description` text,
  `canvas_type` varchar(32) DEFAULT NULL,
  `dsl` longtext,
  `canvas_category` varchar(32) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `usercanvas_create_time` (`create_time`),
  KEY `usercanvas_create_date` (`create_date`),
  KEY `usercanvas_update_time` (`update_time`),
  KEY `usercanvas_update_date` (`update_date`),
  KEY `usercanvas_user_id` (`user_id`),
  KEY `usercanvas_permission` (`permission`),
  KEY `usercanvas_canvas_type` (`canvas_type`),
  KEY `user_canvas_canvas_category` (`canvas_category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_user_canvas_version` (
  `id` varchar(32) NOT NULL,
  `create_time` bigint DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  `user_canvas_id` varchar(255) NOT NULL,
  `title` varchar(255) DEFAULT NULL,
  `description` text,
  `dsl` longtext,
  PRIMARY KEY (`id`),
  KEY `usercanvasversion_create_time` (`create_time`),
  KEY `usercanvasversion_create_date` (`create_date`),
  KEY `usercanvasversion_update_time` (`update_time`),
  KEY `usercanvasversion_update_date` (`update_date`),
  KEY `usercanvasversion_user_canvas_id` (`user_canvas_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS `ragflow_user_tenant_real` (
  `id` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `tenant_id` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `role` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'normal',
  `invited_by` varchar(32) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(1) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '1',
  `create_time` bigint unsigned DEFAULT NULL,
  `create_date` datetime DEFAULT NULL,
  `update_time` bigint unsigned DEFAULT NULL,
  `update_date` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_user_tenant` (`user_id`,`tenant_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `resource_folder_mapping` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary Key ID',
  `space_id` bigint unsigned NOT NULL COMMENT 'Space ID',
  `resource_id` bigint unsigned NOT NULL COMMENT 'Resource ID (workflow_id, knowledge_id, etc.)',
  `resource_type` tinyint unsigned NOT NULL COMMENT 'Resource Type: 1=agent, 2=workflow, 3=knowledge, 4=database, 5=plugin',
  `folder_id` bigint unsigned NOT NULL COMMENT 'Folder ID',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_resource_mapping` (`resource_id`,`resource_type`,`space_id`),
  KEY `idx_space_id_folder_id` (`space_id`,`folder_id`),
  KEY `idx_resource_id_type` (`resource_id`,`resource_type`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Resource to Folder Mapping Table';

CREATE TABLE IF NOT EXISTS `run_record` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `conversation_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'conversation id',
  `section_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'section ID',
  `agent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'agent_id',
  `user_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'user id',
  `source` tinyint unsigned NOT NULL DEFAULT '0' COMMENT 'Execute source 0 API',
  `status` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'status,0 Unknown, 1-Created,2-InProgress,3-Completed,4-Failed,5-Expired,6-Cancelled,7-RequiresAction',
  `creator_id` bigint NOT NULL DEFAULT '0' COMMENT 'creator id',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `failed_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Fail Time in Milliseconds',
  `last_error` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'error message',
  `completed_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Finish Time in Milliseconds',
  `chat_request` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'Original request field',
  `ext` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'ext',
  `usage` json DEFAULT NULL COMMENT 'usage',
  PRIMARY KEY (`id`),
  KEY `idx_c_s` (`conversation_id`,`section_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='run record';

CREATE TABLE IF NOT EXISTS `shortcut_command` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `object_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Entity ID, this command can be used for this entity',
  `command_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'command id',
  `command_name` varchar(255) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'command name',
  `shortcut_command` varchar(255) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'shortcut command',
  `description` varchar(2000) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'description',
  `send_type` tinyint unsigned NOT NULL DEFAULT '0' COMMENT 'send type 0:query 1:panel',
  `tool_type` tinyint unsigned NOT NULL DEFAULT '0' COMMENT 'Type 1 of tool used: WorkFlow 2: Plugin',
  `work_flow_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'workflow id',
  `plugin_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'plugin id',
  `plugin_tool_name` varchar(255) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'plugin tool name',
  `template_query` text COLLATE utf8mb4_general_ci COMMENT 'template query',
  `components` json DEFAULT NULL COMMENT 'Panel parameters',
  `card_schema` text COLLATE utf8mb4_general_ci COMMENT 'card schema',
  `tool_info` json DEFAULT NULL COMMENT 'Tool information includes name+variable list',
  `status` tinyint unsigned NOT NULL DEFAULT '0' COMMENT 'Status, 0 is invalid, 1 is valid',
  `creator_id` bigint unsigned DEFAULT '0' COMMENT 'creator id',
  `is_online` tinyint unsigned NOT NULL DEFAULT '0' COMMENT 'Is online information: 0 draft 1 online',
  `created_at` bigint NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `agent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'When executing a multi instruction, which node executes the instruction',
  `shortcut_icon` json DEFAULT NULL COMMENT 'shortcut icon',
  `plugin_tool_id` bigint NOT NULL DEFAULT '0' COMMENT 'tool_id',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_object_command_id_type` (`object_id`,`command_id`,`is_online`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='bot shortcut command table';

CREATE TABLE IF NOT EXISTS `single_agent_draft` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary Key ID',
  `agent_id` bigint NOT NULL DEFAULT '0' COMMENT 'Agent ID',
  `creator_id` bigint NOT NULL DEFAULT '0' COMMENT 'Creator ID',
  `space_id` bigint NOT NULL DEFAULT '0' COMMENT 'Space ID',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Agent Name',
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Agent Description',
  `icon_uri` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Icon URI',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'delete time in millisecond',
  `variables_meta_id` bigint DEFAULT NULL COMMENT 'variables meta table ID',
  `model_info` json DEFAULT NULL COMMENT 'Model Configuration Information',
  `onboarding_info` json DEFAULT NULL COMMENT 'Onboarding Information',
  `prompt` json DEFAULT NULL COMMENT 'Agent Prompt Configuration',
  `plugin` json DEFAULT NULL COMMENT 'Agent Plugin Base Configuration',
  `knowledge` json DEFAULT NULL COMMENT 'Agent Knowledge Base Configuration',
  `workflow` json DEFAULT NULL COMMENT 'Agent Workflow Configuration',
  `suggest_reply` json DEFAULT NULL COMMENT 'Suggested Replies',
  `jump_config` json DEFAULT NULL COMMENT 'Jump Configuration',
  `background_image_info_list` json DEFAULT NULL COMMENT 'Background image',
  `database_config` json DEFAULT NULL COMMENT 'Agent Database Base Configuration',
  `external_knowledge` json DEFAULT NULL COMMENT 'External knowledge base configuration',
  `bot_mode` tinyint NOT NULL DEFAULT '0' COMMENT 'bot mode,0:single mode 2:chatflow mode',
  `shortcut_command` json DEFAULT NULL COMMENT 'shortcut command',
  `layout_info` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'chatflow layout info',
  `memory_tool_config` json DEFAULT NULL COMMENT 'Memory Tool Configuration',
  `bound_cards` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Bound Cards for Prompt Injection',
  `skill_info_list` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Skill references bound to this agent',
  `force_tool_return` tinyint(1) DEFAULT NULL COMMENT 'Force all tool results to return to model',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_agent_id` (`agent_id`),
  KEY `idx_creator_id` (`creator_id`)
) ENGINE=InnoDB AUTO_INCREMENT=3579 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Single Agent Draft Copy Table';

CREATE TABLE IF NOT EXISTS `single_agent_publish` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `agent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'agent_id',
  `publish_id` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'publish id',
  `connector_ids` json DEFAULT NULL COMMENT 'connector_ids',
  `version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Agent Version',
  `publish_info` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'publish info',
  `publish_time` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'publish time',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `creator_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'creator id',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT 'Status 0: In use 1: Delete 3: Disabled',
  `extra` json DEFAULT NULL COMMENT 'extra',
  PRIMARY KEY (`id`),
  KEY `idx_agent_id_version` (`agent_id`,`version`),
  KEY `idx_creator_id` (`creator_id`),
  KEY `idx_publish_id` (`publish_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7582817375850856449 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Bot connector and release version info';

CREATE TABLE IF NOT EXISTS `single_agent_version` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary Key ID',
  `agent_id` bigint NOT NULL DEFAULT '0' COMMENT 'Agent ID',
  `creator_id` bigint NOT NULL DEFAULT '0' COMMENT 'Creator ID',
  `space_id` bigint NOT NULL DEFAULT '0' COMMENT 'Space ID',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Agent Name',
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Agent Description',
  `icon_uri` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Icon URI',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'delete time in millisecond',
  `variables_meta_id` bigint DEFAULT NULL COMMENT 'variables meta table ID',
  `model_info` json DEFAULT NULL COMMENT 'Model Configuration Information',
  `onboarding_info` json DEFAULT NULL COMMENT 'Onboarding Information',
  `prompt` json DEFAULT NULL COMMENT 'Agent Prompt Configuration',
  `plugin` json DEFAULT NULL COMMENT 'Agent Plugin Base Configuration',
  `knowledge` json DEFAULT NULL COMMENT 'Agent Knowledge Base Configuration',
  `workflow` json DEFAULT NULL COMMENT 'Agent Workflow Configuration',
  `suggest_reply` json DEFAULT NULL COMMENT 'Suggested Replies',
  `jump_config` json DEFAULT NULL COMMENT 'Jump Configuration',
  `connector_id` bigint unsigned NOT NULL COMMENT 'Connector ID',
  `version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Agent Version',
  `background_image_info_list` json DEFAULT NULL COMMENT 'Background image',
  `database_config` json DEFAULT NULL COMMENT 'Agent Database Base Configuration',
  `external_knowledge` json DEFAULT NULL COMMENT 'External knowledge base configuration',
  `bot_mode` tinyint NOT NULL DEFAULT '0' COMMENT 'bot mode,0:single mode 2:chatflow mode',
  `shortcut_command` json DEFAULT NULL COMMENT 'shortcut command',
  `layout_info` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'chatflow layout info',
  `memory_tool_config` json DEFAULT NULL COMMENT 'Memory Tool Configuration',
  `bound_cards` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Bound Cards for Prompt Injection',
  `skill_info_list` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Skill references bound to this agent',
  `force_tool_return` tinyint(1) DEFAULT NULL COMMENT 'Force all tool results to return to model',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_agent_id_and_version_connector_id` (`agent_id`,`version`,`connector_id`),
  KEY `idx_creator_id` (`creator_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7582817379067904001 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Single Agent Version Copy Table';

CREATE TABLE IF NOT EXISTS `space` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary Key ID, Space ID',
  `owner_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Owner ID',
  `name` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Space Name',
  `description` varchar(2000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Space Description',
  `icon_uri` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Icon URI',
  `creator_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Creator ID',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Creation Time (Milliseconds)',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time (Milliseconds)',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT 'Deletion Time (Milliseconds)',
  PRIMARY KEY (`id`),
  KEY `idx_creator_id` (`creator_id`),
  KEY `idx_owner_id` (`owner_id`)
) ENGINE=InnoDB AUTO_INCREMENT=8000000000000000002 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Space Table';

CREATE TABLE IF NOT EXISTS `space_embedding` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `space_id` bigint unsigned NOT NULL COMMENT '空间ID',
  `user_id` bigint unsigned NOT NULL COMMENT '创建者ID',
  `name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '配置名称',
  `description` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT '' COMMENT '配置描述',
  `embedding_type` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Embedding类型: openai, ark, ollama, http',
  `config` json NOT NULL COMMENT 'Embedding配置(JSON格式)',
  `max_batch_size` int NOT NULL DEFAULT '100' COMMENT '最大批处理大小',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态: 1启用 2禁用',
  `is_default` tinyint NOT NULL DEFAULT '0' COMMENT '是否为默认配置: 0否 1是',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '创建时间(毫秒)',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '更新时间(毫秒)',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间(毫秒)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_space_embedding_name` (`space_id`,`name`,`deleted_at`),
  KEY `idx_space_id_status` (`space_id`,`status`),
  KEY `idx_space_id_default` (`space_id`,`is_default`),
  KEY `idx_creator_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='空间Embedding配置表';

CREATE TABLE IF NOT EXISTS `space_model` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `space_id` bigint unsigned NOT NULL COMMENT '空间ID',
  `model_entity_id` bigint unsigned NOT NULL COMMENT '模型实体ID',
  `user_id` bigint unsigned NOT NULL COMMENT '创建者ID',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态: 1启用 2禁用',
  `custom_config` json DEFAULT NULL COMMENT '空间自定义配置(覆盖默认配置)',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '创建时间',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_space_model` (`space_id`,`model_entity_id`),
  KEY `idx_space_id_status` (`space_id`,`status`),
  KEY `idx_model_entity_id` (`model_entity_id`),
  KEY `idx_creator_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7579550703480209409 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='空间模型关联表';

CREATE TABLE IF NOT EXISTS `space_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary Key ID, Auto Increment',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Space ID',
  `user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'User ID',
  `role_type` int NOT NULL DEFAULT '3' COMMENT 'Role Type: 1.owner 2.admin 3.member',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Creation Time (Milliseconds)',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time (Milliseconds)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_space_user` (`space_id`,`user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=203 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Space Member Table';

CREATE TABLE IF NOT EXISTS `statistics_export_file` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'primary key',
  `agent_id` bigint NOT NULL COMMENT 'agent id',
  `export_task_id` varchar(64) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'export task id',
  `file_name` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'exported file name',
  `object_key` varchar(512) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'object storage key',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT 'created time',
  `expire_at` datetime(3) NOT NULL COMMENT 'expire time',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT 'upload status',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_export_task_id` (`export_task_id`),
  KEY `idx_agent_expire` (`agent_id`,`expire_at`),
  KEY `idx_expire_at` (`expire_at`)
) ENGINE=InnoDB AUTO_INCREMENT=22 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Conversation statistics export files';

CREATE TABLE IF NOT EXISTS `table_7537715689126100992` (
  `c_7537715689121906688` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `c_7537715689121923072` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `c_7537715689121939456` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `c_7537715689121955840` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `c_7537715689121972224` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `c_7537715689121988608` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `c_7537715689122004992` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `c_7537715689122021376` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `_knowledge_document_slice_id` bigint NOT NULL,
  PRIMARY KEY (`_knowledge_document_slice_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `table_7539733783084269568` (
  `f_1` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `bstudio_id` bigint NOT NULL AUTO_INCREMENT,
  `bstudio_connector_uid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `bstudio_connector_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `bstudio_create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `f_2` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `f_3` bigint DEFAULT NULL,
  PRIMARY KEY (`bstudio_id`),
  KEY `idx_uid` (`bstudio_connector_uid`,`bstudio_connector_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7539734630887325697 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `table_7539733783759552512` (
  `f_1` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `bstudio_id` bigint NOT NULL AUTO_INCREMENT,
  `bstudio_connector_uid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `bstudio_connector_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `bstudio_create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `f_2` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `f_3` bigint DEFAULT NULL,
  PRIMARY KEY (`bstudio_id`),
  KEY `idx_uid` (`bstudio_connector_uid`,`bstudio_connector_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7539735202998812673 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `table_7542341359437348864` (
  `c_7542341359399600128` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `c_7542341359399616512` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `c_7542341359399632896` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `c_7542341359399649280` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `c_7542341359399665664` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `c_7542341359399682048` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `c_7542341359399698432` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `c_7542341359399714816` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `c_7542341359399731200` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `_knowledge_document_slice_id` bigint NOT NULL,
  PRIMARY KEY (`_knowledge_document_slice_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `table_7544717731438788608` (
  `c_7544717731434594304` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434610688` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `c_7544717731434627072` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434643456` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434659840` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434676224` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434692608` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434708992` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434725376` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434741760` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434758144` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434774528` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434790912` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434807296` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434823680` text COLLATE utf8mb4_unicode_ci,
  `c_7544717731434840064` text COLLATE utf8mb4_unicode_ci,
  `_knowledge_document_slice_id` bigint NOT NULL,
  PRIMARY KEY (`_knowledge_document_slice_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `template` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary Key ID',
  `agent_id` bigint NOT NULL DEFAULT '0' COMMENT 'Agent ID',
  `workflow_id` bigint NOT NULL DEFAULT '0' COMMENT 'Workflow ID',
  `space_id` bigint NOT NULL DEFAULT '0' COMMENT 'Space ID',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `heat` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Heat',
  `product_entity_type` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Product Entity Type',
  `meta_info` json DEFAULT NULL COMMENT 'Meta Info',
  `plugin_extra` json DEFAULT NULL COMMENT 'Plugin Extra Info',
  `agent_extra` json DEFAULT NULL COMMENT 'Agent Extra Info',
  `workflow_extra` json DEFAULT NULL COMMENT 'Workflow Extra Info',
  `project_extra` json DEFAULT NULL COMMENT 'Project Extra Info',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_agent_id_space_id` (`agent_id`,`space_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7550135146683301917 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Template Info Table';

CREATE TABLE IF NOT EXISTS `tool` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tool ID',
  `plugin_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Plugin ID',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Tool Version, e.g. v1.0.0',
  `sub_url` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Sub URL Path',
  `method` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'HTTP Request Method',
  `operation` json DEFAULT NULL COMMENT 'Tool Openapi Operation Schema',
  `activated_status` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '0:activated; 1:deactivated',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_idx_plugin_sub_url_method` (`plugin_id`,`sub_url`,`method`),
  KEY `idx_plugin_activated_status` (`plugin_id`,`activated_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Latest Tool';

CREATE TABLE IF NOT EXISTS `tool_draft` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tool ID',
  `plugin_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Plugin ID',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time in Milliseconds',
  `sub_url` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Sub URL Path',
  `method` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'HTTP Request Method',
  `operation` json DEFAULT NULL COMMENT 'Tool Openapi Operation Schema',
  `debug_status` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '0:not pass; 1:pass',
  `activated_status` tinyint unsigned NOT NULL DEFAULT '0' COMMENT '0:activated; 1:deactivated',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_idx_plugin_sub_url_method` (`plugin_id`,`sub_url`,`method`),
  KEY `idx_plugin_created_at_id` (`plugin_id`,`created_at`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Draft Tool';

CREATE TABLE IF NOT EXISTS `tool_version` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Primary Key ID',
  `tool_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tool ID',
  `plugin_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Plugin ID',
  `version` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Tool Version, e.g. v1.0.0',
  `sub_url` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Sub URL Path',
  `method` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'HTTP Request Method',
  `operation` json DEFAULT NULL COMMENT 'Tool Openapi Operation Schema',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time in Milliseconds',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_idx_tool_version` (`tool_id`,`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Tool Version';

CREATE TABLE IF NOT EXISTS `user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary Key ID',
  `name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'User Nickname',
  `unique_name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'User Unique Name',
  `email` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Email',
  `password` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Password (Encrypted)',
  `description` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'User Description',
  `icon_uri` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Avatar URI',
  `user_verified` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'User Verification Status',
  `locale` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Locale',
  `session_key` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Session Key',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Creation Time (Milliseconds)',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time (Milliseconds)',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT 'Deletion Time (Milliseconds)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_email` (`email`),
  UNIQUE KEY `uniq_unique_name` (`unique_name`),
  KEY `idx_session_key` (`session_key`)
) ENGINE=InnoDB AUTO_INCREMENT=7589282949053284353 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='User Table';

CREATE TABLE IF NOT EXISTS `user_memory_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '用户ID',
  `connector_id` bigint NOT NULL DEFAULT '0' COMMENT '连接器ID',
  `memory_enabled` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否启用记忆功能，0=禁用，1=启用',
  `auto_learn` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否自动学习，0=不自动学习，1=自动学习用户偏好',
  `search_context_lines` int NOT NULL DEFAULT '10' COMMENT '搜索上下文行数，默认前后10行',
  `max_document_lines` int NOT NULL DEFAULT '10000' COMMENT '文档最大行数限制',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_connector_config` (`user_id`,`connector_id`) COMMENT '用户和连接器配置唯一索引',
  KEY `idx_user_id_config` (`user_id`) COMMENT '用户ID索引',
  KEY `idx_memory_enabled` (`memory_enabled`) COMMENT '记忆启用状态索引'
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户记忆配置表，存储用户记忆功能的配置信息';

CREATE TABLE IF NOT EXISTS `user_memory_document` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '用户ID，全局唯一标识',
  `connector_id` bigint NOT NULL DEFAULT '0' COMMENT '连接器ID',
  `document_content` longtext COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '记忆文档内容，完整的文本记录用户的所有记忆',
  `line_count` int NOT NULL DEFAULT '0' COMMENT '文档行数，用于上下文检索',
  `version` int NOT NULL DEFAULT '1' COMMENT '文档版本号，每次更新递增',
  `enabled` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否启用记忆功能，0=禁用，1=启用',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_connector` (`user_id`,`connector_id`) COMMENT '用户和连接器组合唯一索引',
  KEY `idx_user_id` (`user_id`) COMMENT '用户ID索引',
  KEY `idx_enabled` (`enabled`) COMMENT '启用状态索引',
  KEY `idx_updated_at` (`updated_at`) COMMENT '更新时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户记忆文档表，存储每个用户的完整记忆文档';

CREATE TABLE IF NOT EXISTS `variable_instance` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '主键ID',
  `biz_type` tinyint unsigned NOT NULL COMMENT '1 for agent，2 for app',
  `biz_id` varchar(128) NOT NULL DEFAULT '' COMMENT '1 for agent_id，2 for app_id',
  `version` varchar(255) NOT NULL COMMENT 'agent or project 版本,为空代表草稿态',
  `keyword` varchar(255) NOT NULL COMMENT '记忆的KEY',
  `type` tinyint NOT NULL COMMENT '记忆类型 1 KV 2 list',
  `content` text COMMENT '记忆内容',
  `connector_uid` varchar(255) NOT NULL COMMENT '二方用户ID',
  `connector_id` bigint NOT NULL COMMENT '二方id, e.g. ynet = 10000010',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '创建时间',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_connector_key` (`biz_id`,`biz_type`,`version`,`connector_uid`,`connector_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='KV Memory';

CREATE TABLE IF NOT EXISTS `variables_meta` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '主键ID',
  `creator_id` bigint unsigned NOT NULL COMMENT '创建者ID',
  `biz_type` tinyint unsigned NOT NULL COMMENT '1 for agent，2 for app',
  `biz_id` varchar(128) NOT NULL DEFAULT '' COMMENT '1 for agent_id，2 for app_id',
  `variable_list` json DEFAULT NULL COMMENT '变量配置的json数据',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'create time',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'update time',
  `version` varchar(255) NOT NULL COMMENT 'project版本,为空代表草稿态',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_project_key` (`biz_id`,`biz_type`,`version`),
  KEY `idx_user_key` (`creator_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='KV Memory meta';

CREATE TABLE IF NOT EXISTS `workflow_draft` (
  `id` bigint unsigned NOT NULL COMMENT 'workflow ID',
  `canvas` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Front end schema',
  `input_params` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT ' 入参 schema',
  `output_params` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT ' 出参 schema',
  `test_run_success` tinyint(1) NOT NULL DEFAULT '0' COMMENT '0 未运行, 1 运行成功',
  `modified` tinyint(1) NOT NULL DEFAULT '0' COMMENT '0 未被修改, 1 已被修改',
  `updated_at` bigint unsigned DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `commit_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'used to uniquely identify a draft snapshot',
  PRIMARY KEY (`id`),
  KEY `idx_updated_at` (`updated_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='workflow 画布草稿表，用于记录workflow最新的草稿画布信息';

CREATE TABLE IF NOT EXISTS `workflow_execution` (
  `id` bigint unsigned NOT NULL COMMENT 'execute id',
  `workflow_id` bigint unsigned NOT NULL COMMENT 'workflow_id',
  `version` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'workflow version. empty if is draft',
  `space_id` bigint unsigned NOT NULL COMMENT 'the space id the workflow belongs to',
  `mode` tinyint unsigned NOT NULL COMMENT 'the execution mode: 1. debug run 2. release run 3. node debug',
  `operator_id` bigint unsigned NOT NULL COMMENT 'the user id that runs this workflow',
  `connector_id` bigint unsigned DEFAULT NULL COMMENT 'the connector on which this execution happened',
  `connector_uid` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'user id of the connector',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  `log_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'log id',
  `status` tinyint unsigned DEFAULT NULL COMMENT '1=running 2=success 3=fail 4=interrupted',
  `duration` bigint unsigned DEFAULT NULL COMMENT 'execution duration in millisecond',
  `input` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'actual input of this execution',
  `output` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'the actual output of this execution',
  `error_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'error code if any',
  `fail_reason` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'the reason for failure',
  `input_tokens` bigint unsigned DEFAULT NULL COMMENT 'number of input tokens',
  `output_tokens` bigint unsigned DEFAULT NULL COMMENT 'number of output tokens',
  `updated_at` bigint unsigned DEFAULT NULL COMMENT 'update time in millisecond',
  `root_execution_id` bigint unsigned DEFAULT NULL COMMENT 'the top level execution id. Null if this is the root',
  `parent_node_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'the node key for the sub_workflow node that executes this workflow',
  `app_id` bigint unsigned DEFAULT NULL COMMENT 'app id this workflow execution belongs to',
  `node_count` mediumint unsigned DEFAULT NULL COMMENT 'the total node count of the workflow',
  `resume_event_id` bigint unsigned DEFAULT NULL COMMENT 'the current event ID which is resuming',
  `agent_id` bigint unsigned DEFAULT NULL COMMENT 'the agent that this execution binds to',
  `sync_pattern` tinyint unsigned DEFAULT NULL COMMENT 'the sync pattern 1. sync 2. async 3. stream',
  `commit_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'draft commit id this execution belongs to',
  PRIMARY KEY (`id`),
  KEY `idx_workflow_id_version_mode_created_at` (`workflow_id`,`version`,`mode`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='workflow 执行记录表，用于记录每次workflow执行时的状态';

CREATE TABLE IF NOT EXISTS `workflow_meta` (
  `id` bigint unsigned NOT NULL COMMENT 'workflow id',
  `name` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'workflow name',
  `description` varchar(2000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'workflow description',
  `icon_uri` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'icon uri',
  `status` tinyint unsigned NOT NULL COMMENT '0:未发布过, 1:已发布过',
  `content_type` tinyint unsigned NOT NULL COMMENT '0用户 1官方',
  `mode` tinyint unsigned NOT NULL COMMENT '0:workflow, 3:chat_flow',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  `updated_at` bigint unsigned DEFAULT NULL COMMENT 'update time in millisecond',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'delete time in millisecond',
  `creator_id` bigint unsigned NOT NULL COMMENT 'user id for creator',
  `tag` tinyint unsigned DEFAULT NULL COMMENT 'template tag: Tag: 1=All, 2=Hot, 3=Information, 4=Music, 5=Picture, 6=UtilityTool, 7=Life, 8=Traval, 9=Network, 10=System, 11=Movie, 12=Office, 13=Shopping, 14=Education, 15=Health, 16=Social, 17=Entertainment, 18=Finance, 100=Hidden',
  `author_id` bigint unsigned NOT NULL COMMENT '原作者用户 ID',
  `space_id` bigint unsigned NOT NULL COMMENT ' 空间 ID',
  `updater_id` bigint unsigned DEFAULT NULL COMMENT ' 更新元信息的用户 ID',
  `source_id` bigint unsigned DEFAULT NULL COMMENT ' 复制来源的 workflow ID',
  `app_id` bigint unsigned DEFAULT NULL COMMENT '应用 ID',
  `latest_version` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'the version of the most recent publish',
  `latest_version_ts` bigint unsigned DEFAULT NULL COMMENT 'create time of latest version',
  PRIMARY KEY (`id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_latest_version_ts` (`latest_version_ts` DESC),
  KEY `idx_space_id_app_id_status_latest_version_ts` (`space_id`,`app_id`,`status`,`latest_version_ts`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='workflow 元信息表，用于记录workflow基本的元信息';

CREATE TABLE IF NOT EXISTS `workflow_meta_backup_20250812` (
  `id` bigint unsigned NOT NULL COMMENT 'workflow id',
  `name` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'workflow name',
  `description` varchar(2000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'workflow description',
  `icon_uri` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'icon uri',
  `status` tinyint unsigned NOT NULL COMMENT '0:未发布过, 1:已发布过',
  `content_type` tinyint unsigned NOT NULL COMMENT '0用户 1官方',
  `mode` tinyint unsigned NOT NULL COMMENT '0:workflow, 3:chat_flow',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  `updated_at` bigint unsigned DEFAULT NULL COMMENT 'update time in millisecond',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT 'delete time in millisecond',
  `creator_id` bigint unsigned NOT NULL COMMENT 'user id for creator',
  `tag` tinyint unsigned DEFAULT NULL COMMENT 'template tag: Tag: 1=All, 2=Hot, 3=Information, 4=Music, 5=Picture, 6=UtilityTool, 7=Life, 8=Traval, 9=Network, 10=System, 11=Movie, 12=Office, 13=Shopping, 14=Education, 15=Health, 16=Social, 17=Entertainment, 18=Finance, 100=Hidden',
  `author_id` bigint unsigned NOT NULL COMMENT '原作者用户 ID',
  `space_id` bigint unsigned NOT NULL COMMENT ' 空间 ID',
  `updater_id` bigint unsigned DEFAULT NULL COMMENT ' 更新元信息的用户 ID',
  `source_id` bigint unsigned DEFAULT NULL COMMENT ' 复制来源的 workflow ID',
  `app_id` bigint unsigned DEFAULT NULL COMMENT '应用 ID',
  `latest_version` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'the version of the most recent publish',
  `latest_version_ts` bigint unsigned DEFAULT NULL COMMENT 'create time of latest version'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `workflow_reference` (
  `id` bigint unsigned NOT NULL COMMENT 'workflow id',
  `referred_id` bigint unsigned NOT NULL COMMENT 'the id of the workflow that is referred by other entities',
  `referring_id` bigint unsigned NOT NULL COMMENT 'the entity id that refers this workflow',
  `refer_type` tinyint unsigned NOT NULL COMMENT '1 subworkflow 2 tool',
  `referring_biz_type` tinyint unsigned NOT NULL COMMENT 'the biz type the referring entity belongs to: 1. workflow 2. agent',
  `created_at` bigint unsigned NOT NULL COMMENT 'create time in millisecond',
  `status` tinyint unsigned NOT NULL COMMENT 'whether this reference currently takes effect. 0: disabled 1: enabled',
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_referred_id_referring_id_refer_type` (`referred_id`,`referring_id`,`refer_type`),
  KEY `idx_referred_id_referring_biz_type_status` (`referred_id`,`referring_biz_type`,`status`),
  KEY `idx_referring_id_status` (`referring_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='workflow 关联关系表，用于记录workflow 直接互相引用关系';

CREATE TABLE IF NOT EXISTS `workflow_snapshot` (
  `workflow_id` bigint unsigned NOT NULL COMMENT 'workflow id this snapshot belongs to',
  `commit_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'the commit id of the workflow draft',
  `canvas` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'frontend schema for this snapshot',
  `input_params` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'input parameter info',
  `output_params` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'output parameter info',
  `created_at` bigint unsigned NOT NULL,
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_workflow_id_commit_id` (`workflow_id`,`commit_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1866 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='snapshot for executed workflow draft';

CREATE TABLE IF NOT EXISTS `workflow_version` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `workflow_id` bigint unsigned NOT NULL COMMENT 'workflow id',
  `version` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '发布版本',
  `version_description` varchar(2000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '版本描述',
  `canvas` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT 'Front end schema',
  `input_params` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `output_params` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `creator_id` bigint unsigned NOT NULL COMMENT '发布用户 ID',
  `created_at` bigint unsigned NOT NULL COMMENT '创建时间毫秒时间戳',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT '删除毫秒时间戳',
  `commit_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'the commit id corresponding to this version',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_workflow_id_version` (`workflow_id`,`version`),
  KEY `idx_id_created_at` (`workflow_id`,`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1862 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='workflow 画布版本信息表，用于记录不同版本的画布信息';


-- ============================================================
-- New tables from production (2026-03-27)
-- ============================================================

CREATE TABLE IF NOT EXISTS `skill` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `skill_id` bigint(20) NOT NULL,
  `space_id` bigint(20) NOT NULL,
  `name` varchar(255) NOT NULL,
  `description` text DEFAULT NULL,
  `prompt` mediumtext DEFAULT NULL,
  `icon_uri` varchar(1024) NOT NULL DEFAULT '',
  `creator_id` bigint(20) NOT NULL DEFAULT '0',
  `status` tinyint(4) NOT NULL DEFAULT '1',
  `created_at` bigint(20) NOT NULL DEFAULT '0',
  `updated_at` bigint(20) NOT NULL DEFAULT '0',
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_skill_id` (`skill_id`),
  UNIQUE KEY `uniq_space_name_deleted` (`space_id`, `name`, `deleted_at`),
  KEY `idx_space_id` (`space_id`)
) DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `testcase_data` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '用例名称',
  `description` text DEFAULT NULL,
  `input` text DEFAULT NULL COMMENT 'JSON格式的输入数据',
  `is_default` tinyint(1) DEFAULT '0',
  `space_id` bigint(20) NOT NULL,
  `component_type` int(11) NOT NULL,
  `component_id` varchar(64) NOT NULL,
  `parent_component_type` int(11) DEFAULT NULL,
  `parent_component_id` varchar(64) DEFAULT NULL,
  `creator_id` bigint(20) NOT NULL,
  `updater_id` bigint(20) DEFAULT NULL,
  `created_at` bigint(20) DEFAULT NULL,
  `updated_at` bigint(20) DEFAULT NULL,
  `deleted_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_space_component` (`space_id`, `parent_component_id`, `component_id`),
  KEY `idx_name` (`name`),
  KEY `idx_deleted_at` (`deleted_at`)
) DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `plugin_categories` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL DEFAULT '',
  `description` text DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT '0',
  `status` tinyint(4) NOT NULL DEFAULT '1',
  `created_at` bigint(20) unsigned NOT NULL DEFAULT '0',
  `updated_at` bigint(20) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `idx_status_sort` (`status`, `sort_order`)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plugin_marketplace_categories` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `marketplace_id` bigint(20) unsigned NOT NULL,
  `category_id` bigint(20) unsigned NOT NULL,
  `created_at` bigint(20) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_marketplace_category` (`marketplace_id`, `category_id`),
  KEY `idx_category_id` (`category_id`)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plugin_marketplace_tools` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `marketplace_id` bigint(20) unsigned NOT NULL,
  `tool_id` bigint(20) unsigned NOT NULL,
  `display_name` varchar(255) NOT NULL DEFAULT '',
  `display_description` text DEFAULT NULL,
  `example_usage` text DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT '0',
  `status` tinyint(4) NOT NULL DEFAULT '1',
  `created_at` bigint(20) unsigned NOT NULL DEFAULT '0',
  `updated_at` bigint(20) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_marketplace_tool` (`marketplace_id`, `tool_id`),
  KEY `idx_tool_id` (`tool_id`)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `plugin_usage_stats` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `marketplace_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned DEFAULT NULL,
  `space_id` bigint(20) unsigned DEFAULT NULL,
  `usage_type` tinyint(4) DEFAULT NULL,
  `usage_date` date NOT NULL,
  `usage_count` int(11) NOT NULL DEFAULT '1',
  `created_at` bigint(20) unsigned NOT NULL DEFAULT '0',
  `updated_at` bigint(20) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_marketplace_user_space_date` (`marketplace_id`, `user_id`, `space_id`, `usage_date`),
  KEY `idx_usage_date` (`usage_date`),
  KEY `idx_user_space` (`user_id`, `space_id`)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `user_plugin_favorites` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `marketplace_id` bigint(20) unsigned NOT NULL,
  `created_at` bigint(20) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_plugin` (`user_id`, `marketplace_id`),
  KEY `idx_marketplace_id` (`marketplace_id`)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
