-- 添加缺失的ChatFlow对话历史表
-- 执行前请确保连接到正确的数据库

USE opencoze;

-- Create "chatflow_conversation_history" table
CREATE TABLE IF NOT EXISTS `chatflow_conversation_history` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary Key ID',
  `conversation_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Conversation identifier',
  `workflow_id` bigint unsigned NOT NULL COMMENT 'Chatflow workflow ID',
  `space_id` bigint unsigned NOT NULL COMMENT 'Space ID',
  `user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'User ID who initiated the conversation',
  `role` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Message role: user, assistant, system',
  `content` longtext COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Message content',
  `execution_id` bigint unsigned DEFAULT NULL COMMENT 'Workflow execution ID that generated this message',
  `node_id` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Node ID that generated this message',
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

-- 验证表创建
SELECT 'chatflow_conversation_history表创建完成' as message;