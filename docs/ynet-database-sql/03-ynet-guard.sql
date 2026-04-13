-- ============================================================
-- Ynet Guard Database Schema
-- Source: 172.93.101.237:2881 (OceanBase) / guard
-- Exported: 2026-03-17
-- ============================================================
CREATE DATABASE IF NOT EXISTS `guard` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `guard`;

CREATE TABLE IF NOT EXISTS `api_keys` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `user_id` varchar(32) DEFAULT NULL COMMENT '所属用户ID',
  `business_id` varchar(32) DEFAULT NULL COMMENT '关联业务ID',
  `key_hash` varchar(64) NOT NULL COMMENT 'API Key哈希值',
  `name` varchar(100) NOT NULL COMMENT 'API Key名称',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `last_used_at` datetime DEFAULT NULL COMMENT '最后使用时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `key_hash` (`key_hash`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='API密钥表' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `audit_logs` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `request_id` varchar(64) NOT NULL COMMENT '请求ID，唯一标识',
  `trace_id` varchar(64) DEFAULT NULL COMMENT '链路追踪ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `query` text NOT NULL COMMENT '检测的查询内容',
  `session_id` varchar(64) DEFAULT NULL COMMENT '会话ID',
  `user_id` varchar(64) DEFAULT NULL COMMENT '用户ID',
  `scene` varchar(50) DEFAULT NULL COMMENT '检测场景',
  `passed` tinyint(1) NOT NULL COMMENT '是否通过检测',
  `action` varchar(20) NOT NULL COMMENT '处理动作（pass/block/review等）',
  `source` varchar(20) DEFAULT NULL COMMENT '请求来源',
  `risk_level` varchar(20) DEFAULT NULL COMMENT '风险等级（R1/R2/R3）',
  `risk_type` varchar(50) DEFAULT NULL COMMENT '风险类型',
  `reason` text DEFAULT NULL COMMENT '判定原因',
  `matched_rule` varchar(64) DEFAULT NULL COMMENT '匹配的规则ID',
  `confidence` float DEFAULT NULL COMMENT '置信度',
  `answer` text DEFAULT NULL COMMENT '代答内容',
  `total_ms` int(11) DEFAULT NULL COMMENT '总耗时（毫秒）',
  `model_check_ms` int(11) DEFAULT NULL COMMENT 'AI模型检测耗时（毫秒）',
  `kb_match_ms` int(11) DEFAULT NULL COMMENT '知识库匹配耗时（毫秒）',
  `extra` json DEFAULT NULL COMMENT '扩展信息',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `request_id` (`request_id`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_audit_logs_created_at` (`created_at`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_audit_logs_tenant_id` (`tenant_id`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='审计日志表（结构化存储，同时写入ES便于检索）' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `categories` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `name` varchar(50) NOT NULL COMMENT '分类名称',
  `description` text DEFAULT NULL COMMENT '分类描述',
  `color` varchar(20) DEFAULT NULL COMMENT '显示颜色（十六进制，如#ff5733）',
  `parent_id` varchar(32) DEFAULT NULL COMMENT '父分类ID，支持层级结构',
  `sort_order` int(11) NOT NULL COMMENT '排序序号',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`)
) DEFAULT CHARSET = utf8mb4 COMMENT='知识库分类表' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `kb_entries` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `kb_id` varchar(32) NOT NULL COMMENT '所属知识库ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `category_id` varchar(32) DEFAULT NULL COMMENT '关联安全分类ID',
  `keywords` json DEFAULT NULL COMMENT '关键词匹配规则（JSON数组）',
  `semantic_text` text DEFAULT NULL COMMENT '语义匹配文本',
  `risk_type` varchar(50) DEFAULT NULL COMMENT '风险类型',
  `risk_level` varchar(20) NOT NULL COMMENT '风险等级（R1/R2/R3）',
  `answer_template` text DEFAULT NULL COMMENT '代答模板内容',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `version` int(11) NOT NULL COMMENT '版本号',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`)
) DEFAULT CHARSET = utf8mb4 COMMENT='知识库条目表（匹配规则、风险配置、代答内容）' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `kb_vectors` (
  `id` varchar(64) NOT NULL COMMENT '主键ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `kb_id` varchar(32) NOT NULL COMMENT '所属知识库ID',
  `entry_id` varchar(32) NOT NULL COMMENT '关联知识库条目ID',
  `vector` VECTOR(1024) DEFAULT NULL COMMENT '向量嵌入（1024维）',
  PRIMARY KEY (`id`),
  KEY `idx_tenant_id` (`tenant_id`) BLOCK_SIZE 16384 LOCAL,
  KEY `idx_kb_id` (`kb_id`) BLOCK_SIZE 16384 LOCAL,
  KEY `idx_entry_id` (`entry_id`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='知识库向量表（语义匹配用）' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `knowledge_bases` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `name` varchar(100) NOT NULL COMMENT '知识库名称',
  `description` text DEFAULT NULL COMMENT '知识库描述',
  `is_system` tinyint(1) NOT NULL COMMENT '是否系统知识库（所有用户可见且不可删除）',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `ix_knowledge_bases_is_system` (`is_system`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='知识库表' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `operation_logs` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `user_id` varchar(32) DEFAULT NULL COMMENT '操作用户ID',
  `action` varchar(20) NOT NULL COMMENT '操作类型（create/update/delete等）',
  `resource` varchar(20) NOT NULL COMMENT '操作资源类型',
  `resource_id` varchar(32) DEFAULT NULL COMMENT '操作资源ID',
  `detail` text DEFAULT NULL COMMENT '操作详情',
  `ip_address` varchar(45) DEFAULT NULL COMMENT '操作者IP地址',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`)
) DEFAULT CHARSET = utf8mb4 COMMENT='操作日志表（用户行为审计）' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `shield_audit_logs` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `request_id` varchar(64) NOT NULL COMMENT '请求ID，唯一标识',
  `trace_id` varchar(64) DEFAULT NULL COMMENT '链路追踪ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `product_id` varchar(32) NOT NULL COMMENT '产品ID',
  `product_code` varchar(50) NOT NULL COMMENT '产品编码',
  `business_id` varchar(32) NOT NULL COMMENT '业务ID',
  `business_code` varchar(50) NOT NULL COMMENT '业务编码',
  `content` text NOT NULL COMMENT '检测内容',
  `content_type` varchar(20) NOT NULL COMMENT '内容类型（text/image等）',
  `client_ip` varchar(50) DEFAULT NULL COMMENT '客户端IP',
  `user_id` varchar(64) DEFAULT NULL COMMENT '终端用户ID',
  `device_id` varchar(64) DEFAULT NULL COMMENT '设备ID',
  `passed` tinyint(1) NOT NULL COMMENT '是否通过检测',
  `action` varchar(20) NOT NULL COMMENT '处理动作（pass/block/review等）',
  `hit_categories` json DEFAULT NULL COMMENT '命中的安全分类列表',
  `hit_blocklist_id` varchar(32) DEFAULT NULL COMMENT '命中的名单库ID',
  `hit_blocklist_value` varchar(500) DEFAULT NULL COMMENT '命中的名单库值',
  `ai_model_triggered` tinyint(1) NOT NULL COMMENT '是否触发AI模型检测',
  `ai_risk_type` varchar(50) DEFAULT NULL COMMENT 'AI判定风险类型',
  `ai_risk_level` varchar(20) DEFAULT NULL COMMENT 'AI判定风险等级',
  `ai_confidence` float DEFAULT NULL COMMENT 'AI判定置信度',
  `total_ms` int(11) DEFAULT NULL COMMENT '总耗时（毫秒）',
  `ai_check_ms` int(11) DEFAULT NULL COMMENT 'AI检测耗时（毫秒）',
  `kb_check_ms` int(11) DEFAULT NULL COMMENT '知识库检测耗时（毫秒）',
  `policy_check_ms` int(11) DEFAULT NULL COMMENT '策略检测耗时（毫秒）',
  `blocklist_check_ms` int(11) DEFAULT NULL COMMENT '名单库检测耗时（毫秒）',
  `extra` json DEFAULT NULL COMMENT '扩展信息',
  `review_status` varchar(20) DEFAULT NULL COMMENT '人工审核状态',
  `review_user_id` varchar(32) DEFAULT NULL COMMENT '审核人员ID',
  `review_at` datetime DEFAULT NULL COMMENT '审核时间',
  `review_remark` text DEFAULT NULL COMMENT '审核备注',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `request_id` (`request_id`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_shield_audit_logs_action` (`action`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_shield_audit_logs_business_id` (`business_id`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_shield_audit_logs_created_at` (`created_at`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_shield_audit_logs_product_id` (`product_id`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_shield_audit_logs_tenant_id` (`tenant_id`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='鹰盾检测审计日志（记录每次检测的详细信息，支持运营分析）' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `shield_blocklist_entries` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `blocklist_id` varchar(32) NOT NULL COMMENT '所属名单库ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `value` varchar(500) NOT NULL COMMENT '名单值（关键词/用户ID/IP/设备ID）',
  `remark` text DEFAULT NULL COMMENT '备注说明',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `ix_shield_blocklist_entries_blocklist_id` (`blocklist_id`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_shield_blocklist_entries_tenant_id` (`tenant_id`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_shield_blocklist_entries_value` (`value`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='名单库条目表' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `shield_blocklists` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `name` varchar(100) NOT NULL COMMENT '名单库名称',
  `description` text DEFAULT NULL COMMENT '名单库描述',
  `list_type` varchar(20) NOT NULL COMMENT '名单类型（keyword/user/ip/device）',
  `mode` varchar(20) NOT NULL COMMENT '名单模式（block黑名单/allow白名单）',
  `category_id` varchar(32) DEFAULT NULL COMMENT '关联安全分类ID',
  `risk_level` varchar(10) DEFAULT NULL COMMENT '风险等级（R1/R2/R3）',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `ix_shield_blocklists_tenant_id` (`tenant_id`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='名单库表（支持关键词/用户/IP/设备黑白名单）' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `shield_business_blocklists` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `business_id` varchar(32) NOT NULL COMMENT '业务ID',
  `blocklist_id` varchar(32) NOT NULL COMMENT '名单库ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `ix_shield_business_blocklists_blocklist_id` (`blocklist_id`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_shield_business_blocklists_business_id` (`business_id`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='业务与名单库关联表（多对多）' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `shield_businesses` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `product_id` varchar(32) NOT NULL COMMENT '所属产品ID',
  `name` varchar(100) NOT NULL COMMENT '业务名称',
  `code` varchar(50) NOT NULL COMMENT '业务编码（API调用标识）',
  `description` text DEFAULT NULL COMMENT '业务描述',
  `business_type` varchar(20) NOT NULL COMMENT '业务类型（input输入检测/output输出检测/both双向）',
  `enable_ai_model` tinyint(1) NOT NULL COMMENT '是否启用AI模型检测',
  `enable_kb` tinyint(1) NOT NULL COMMENT '是否启用知识库匹配',
  `kb_category_ids` json DEFAULT NULL COMMENT '关联的知识库分类ID列表',
  `enable_blocklist` tinyint(1) NOT NULL COMMENT '是否启用名单库过滤',
  `default_action` varchar(20) NOT NULL COMMENT '默认处理动作',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `ix_shield_businesses_product_id` (`product_id`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_shield_businesses_tenant_id` (`tenant_id`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='业务配置表（产品下的检测场景，如输入检测、输出检测）' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `shield_categories` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `code` varchar(50) NOT NULL COMMENT '分类编码',
  `name` varchar(100) NOT NULL COMMENT '分类名称',
  `description` text DEFAULT NULL COMMENT '分类描述',
  `level` int(11) NOT NULL COMMENT '层级（1=一级分类/2=二级分类/3=三级分类）',
  `parent_id` varchar(32) DEFAULT NULL COMMENT '父分类ID',
  `path` varchar(500) NOT NULL COMMENT '分类路径（如/广告/引流广告/公众号）',
  `default_risk_level` varchar(10) NOT NULL COMMENT '默认风险等级（R1低/R2中/R3高）',
  `sort_order` int(11) NOT NULL COMMENT '排序序号',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `ix_shield_categories_tenant_id` (`tenant_id`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='鹰盾安全分类表（三级分类体系：一级9个/二级128个/三级595个）' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `shield_policy_configs` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `business_id` varchar(32) NOT NULL COMMENT '业务ID',
  `category_id` varchar(32) NOT NULL COMMENT '安全分类ID',
  `action_r1` varchar(20) NOT NULL COMMENT '低风险(R1)处理动作',
  `action_r2` varchar(20) NOT NULL COMMENT '中风险(R2)处理动作',
  `action_r3` varchar(20) NOT NULL COMMENT '高风险(R3)处理动作',
  `action_default` varchar(20) NOT NULL COMMENT '默认处理动作',
  `priority` int(11) NOT NULL COMMENT '策略优先级',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `ix_shield_policy_configs_business_id` (`business_id`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_shield_policy_configs_category_id` (`category_id`) BLOCK_SIZE 16384 LOCAL,
  KEY `ix_shield_policy_configs_tenant_id` (`tenant_id`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='策略配置表（定义业务对各分类的处理方式）' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `shield_products` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `name` varchar(100) NOT NULL COMMENT '产品名称',
  `code` varchar(50) NOT NULL COMMENT '产品编码（API调用标识）',
  `description` text DEFAULT NULL COMMENT '产品描述',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `ix_shield_products_tenant_id` (`tenant_id`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='产品表（最顶层资源，如个人手机、企业手机、平板设备等）' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `tenants` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `name` varchar(100) NOT NULL COMMENT '租户名称',
  `description` text DEFAULT NULL COMMENT '租户描述',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`)
) DEFAULT CHARSET = utf8mb4 COMMENT='租户表（多租户隔离）' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

CREATE TABLE IF NOT EXISTS `users` (
  `id` varchar(32) NOT NULL COMMENT '主键ID',
  `tenant_id` varchar(32) NOT NULL COMMENT '租户ID',
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `email` varchar(100) NOT NULL COMMENT '邮箱',
  `password_hash` varchar(128) NOT NULL COMMENT '密码哈希值',
  `role` varchar(20) NOT NULL COMMENT '角色（admin/operator/viewer）',
  `enabled` tinyint(1) NOT NULL COMMENT '是否启用',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  `last_login_at` datetime DEFAULT NULL COMMENT '最后登录时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_users_tenant_username` (`tenant_id`, `username`) BLOCK_SIZE 16384 LOCAL
) DEFAULT CHARSET = utf8mb4 COMMENT='用户表' ROW_FORMAT = DYNAMIC COMPRESSION = 'zstd_1.3.8' REPLICA_NUM = 1 BLOCK_SIZE = 16384 USE_BLOOM_FILTER = FALSE TABLET_SIZE = 134217728 PCTFREE = 0;

-- Guard 系统配置表（模型配置热更新）
CREATE TABLE IF NOT EXISTS `system_config` (
  `key` VARCHAR(100) PRIMARY KEY,
  `value` TEXT,
  `description` VARCHAR(255) DEFAULT '',
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_by` VARCHAR(50) DEFAULT 'system'
);
