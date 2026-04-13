-- ============================================================
-- Ynet Loop Database Schema
-- Source: 10.10.10.226:3306 / cozeloop-mysql -> ynetloop-mysql
-- Exported: 2026-03-18 (re-exported with correct encoding)
-- ============================================================
CREATE DATABASE IF NOT EXISTS `ynetloop-mysql` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE `ynetloop-mysql`;

CREATE TABLE IF NOT EXISTS `annotate_record` (
  `id` bigint unsigned NOT NULL COMMENT 'idgen record id',
  `space_id` bigint unsigned NOT NULL COMMENT '空间id，分片键',
  `tag_key_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '标签 id',
  `experiment_id` bigint unsigned NOT NULL COMMENT '实验id',
  `score` decimal(10,4) DEFAULT NULL COMMENT '得分结果',
  `text_value` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '文本结果',
  `annotate_data` mediumblob COMMENT '标注结果, json',
  `created_at` bigint NOT NULL DEFAULT '0' COMMENT '创建时间',
  `updated_at` bigint NOT NULL DEFAULT '0' COMMENT '更新时间',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT '软删除时间',
  `created_by` bigint NOT NULL DEFAULT '0' COMMENT '创建人userID',
  `tag_value_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '标签值 id',
  PRIMARY KEY (`id`),
  KEY `idx_space_id_experiment_id_tag_key_id` (`space_id`,`experiment_id`,`tag_key_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='annotate_record';

CREATE TABLE IF NOT EXISTS `api_key` (
  `id` bigint unsigned NOT NULL COMMENT 'Primary Key ID',
  `key` varchar(255) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'API Key hash',
  `name` varchar(255) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'API Key Name',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '0 normal, 1 deleted',
  `user_id` bigint NOT NULL DEFAULT '0' COMMENT 'API Key Owner',
  `expired_at` bigint NOT NULL DEFAULT '0' COMMENT 'API Key Expired Time',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created Time',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated Time',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT 'Deleted Time',
  `last_used_at` bigint NOT NULL DEFAULT '0' COMMENT 'Last Used Time',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='api key table';

CREATE TABLE IF NOT EXISTS `auto_task_run` (
  `id` bigint unsigned NOT NULL COMMENT 'TaskRun ID',
  `workspace_id` bigint unsigned NOT NULL COMMENT '空间ID',
  `task_id` bigint unsigned NOT NULL COMMENT 'Task ID',
  `task_type` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'Task类型',
  `run_status` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'Task Run状态',
  `run_detail` json DEFAULT NULL COMMENT 'Task Run运行状态详情',
  `backfill_detail` json DEFAULT NULL COMMENT '历史回溯Task Run运行状态详情',
  `run_start_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '任务开始时间',
  `run_end_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '任务结束时间',
  `run_config` json DEFAULT NULL COMMENT '相关Run的配置信息',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_task_id_status` (`task_id`,`run_status`),
  KEY `idx_workspace_task` (`workspace_id`,`task_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Task Run信息';

CREATE TABLE IF NOT EXISTS `dataset` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `app_id` int unsigned NOT NULL DEFAULT '0' COMMENT '应用 ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 ID',
  `schema_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Schema ID',
  `name` varchar(255) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '数据集名称',
  `description` varchar(2048) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '数据集描述',
  `category` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '业务场景分类',
  `biz_category` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '业务场景下自定义分类',
  `status` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '状态',
  `security_level` varchar(32) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '安全等级',
  `visibility` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '可见性',
  `spec` json DEFAULT NULL COMMENT '规格配置',
  `features` json DEFAULT NULL COMMENT '功能开关',
  `latest_version` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '最新版本号',
  `next_version_num` bigint unsigned NOT NULL DEFAULT '1' COMMENT '下一个版本的数字版本号',
  `last_operation` varchar(255) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '最新操作',
  `created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '修改人',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT '删除时间',
  `expired_at` timestamp NULL DEFAULT NULL COMMENT '过期时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_space_id_category_name` (`space_id`,`category`,`name`,`deleted_at`),
  KEY `idx_space_id_category_updated_at_id` (`space_id`,`category`,`updated_at`,`id`)
) ENGINE=InnoDB AUTO_INCREMENT=7544955570130780162 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;数据集';

CREATE TABLE IF NOT EXISTS `dataset_io_job` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `app_id` int unsigned NOT NULL DEFAULT '0' COMMENT '应用 ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 ID',
  `dataset_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '数据集 ID',
  `job_type` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '任务类型',
  `source_file` json DEFAULT NULL COMMENT '源文件信息',
  `source_dataset` json DEFAULT NULL COMMENT '源数据集信息',
  `target_file` json DEFAULT NULL COMMENT '目标文件信息',
  `target_dataset` json DEFAULT NULL COMMENT '目标数据集信息',
  `field_mappings` json DEFAULT NULL COMMENT '字段映射',
  `option` json DEFAULT NULL COMMENT '任务选项',
  `status` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '状态',
  `progress_total` bigint unsigned NOT NULL DEFAULT '0' COMMENT '总数',
  `progress_processed` bigint unsigned NOT NULL DEFAULT '0' COMMENT '已处理的数量',
  `progress_added` bigint unsigned NOT NULL DEFAULT '0' COMMENT '已写入的数量',
  `sub_progresses` json DEFAULT NULL COMMENT '进度信息',
  `errors` json DEFAULT NULL COMMENT '错误信息',
  `created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '修改人',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `started_at` timestamp NULL DEFAULT NULL COMMENT '开始时间',
  `ended_at` timestamp NULL DEFAULT NULL COMMENT '结束时间',
  PRIMARY KEY (`id`),
  KEY `idx_space_dataset` (`space_id`,`dataset_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='数据集导入导出任务';

CREATE TABLE IF NOT EXISTS `dataset_item` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `app_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '应用 ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 ID',
  `dataset_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '数据集 ID',
  `schema_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Schema ID',
  `item_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '条目 ID',
  `item_key` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '幂等 key',
  `data` json DEFAULT NULL COMMENT '数据内容',
  `repeated_data` json DEFAULT NULL COMMENT '多轮数据内容',
  `data_properties` json DEFAULT NULL COMMENT '内容属性',
  `add_vn` bigint unsigned NOT NULL DEFAULT '0' COMMENT '添加版本号',
  `del_vn` bigint unsigned NOT NULL DEFAULT '0' COMMENT '删除版本号',
  `created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '修改人',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT '删除时间',
  `update_version` bigint unsigned NOT NULL DEFAULT '0' COMMENT '更新版本号，用于乐观锁',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_dataset_add_vn_item_id_deleted_at` (`dataset_id`,`add_vn`,`item_id`,`deleted_at`),
  UNIQUE KEY `uk_dataset_add_vn_item_key_deleted_at` (`dataset_id`,`add_vn`,`item_key`,`deleted_at`),
  KEY `idx_dataset_del_vn_created_at_item` (`dataset_id`,`del_vn`,`created_at`,`item_id`),
  KEY `idx_dataset_del_vn_updated_at_item` (`dataset_id`,`del_vn`,`updated_at`,`item_id`),
  KEY `idx_dataset_add_vn_del_vn_item` (`dataset_id`,`add_vn`,`del_vn`,`item_id`),
  KEY `idx_dataset_del_vn_item_id` (`dataset_id`,`del_vn`,`item_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7540962802681249794 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;数据集条目';

CREATE TABLE IF NOT EXISTS `dataset_item_snapshot` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `app_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '应用 ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 ID',
  `dataset_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '数据集 ID',
  `schema_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Schema ID',
  `version_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Version ID',
  `item_primary_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '条目主键 ID',
  `item_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '条目 ID',
  `item_key` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '条目幂等 key',
  `data` json DEFAULT NULL COMMENT '数据内容',
  `repeated_data` json DEFAULT NULL COMMENT '多轮数据内容',
  `data_properties` json DEFAULT NULL COMMENT '内容属性',
  `add_vn` bigint unsigned NOT NULL DEFAULT '0' COMMENT '添加版本号',
  `del_vn` bigint unsigned NOT NULL DEFAULT '0' COMMENT '删除版本号',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'snapshot 创建时间',
  `item_created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'item 创建人',
  `item_created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'item 创建时间',
  `item_updated_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'item 修改人',
  `item_updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'item 修改时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_version_item` (`version_id`,`item_id`),
  KEY `idx_version_item_created_at_item` (`version_id`,`item_created_at`,`item_id`),
  KEY `idx_version_item_updated_at_item` (`version_id`,`item_updated_at`,`item_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7540962953021882370 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;数据集条目快照';

CREATE TABLE IF NOT EXISTS `dataset_schema` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `app_id` int unsigned NOT NULL DEFAULT '0' COMMENT '应用 ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 ID',
  `dataset_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '数据集 ID',
  `fields` json NOT NULL COMMENT '字段格式',
  `immutable` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否不允许编辑',
  `created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '修改人',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `update_version` bigint unsigned NOT NULL DEFAULT '0' COMMENT '更新版本号',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=7544955570130796546 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;数据集 Schema';

CREATE TABLE IF NOT EXISTS `dataset_version` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `app_id` int unsigned NOT NULL DEFAULT '0' COMMENT '应用 ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 ID',
  `dataset_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '数据集 ID',
  `schema_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Schema ID',
  `dataset_brief` json DEFAULT NULL COMMENT '数据集元信息备份',
  `version` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '版本号，SemVer2 三段式',
  `version_num` bigint unsigned NOT NULL DEFAULT '1' COMMENT '数字版本号，从1开始递增',
  `description` varchar(2048) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '版本描述',
  `item_count` bigint unsigned NOT NULL DEFAULT '0' COMMENT '条数',
  `snapshot_status` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '快照状态',
  `snapshot_progress` json DEFAULT NULL COMMENT '快照进度详情',
  `update_version` bigint unsigned NOT NULL DEFAULT '0' COMMENT '更新版本号',
  `created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `disabled_at` timestamp NULL DEFAULT NULL COMMENT '版本禁用时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_dataset_id_version` (`dataset_id`,`version`),
  KEY `idx_dataset_id_created_at_id` (`dataset_id`,`created_at`,`id`)
) ENGINE=InnoDB AUTO_INCREMENT=7540962952791195650 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;数据集版本';

CREATE TABLE IF NOT EXISTS `eval_target` (
  `id` bigint unsigned NOT NULL COMMENT 'idgen id',
  `space_id` bigint unsigned NOT NULL COMMENT '空间id',
  `source_target_id` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT '来源的对象的ID，比如promptID',
  `target_type` int unsigned NOT NULL COMMENT '评估对象类型',
  `created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '创建人',
  `updated_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '更新人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_space_id_source_target_id_target_type` (`space_id`,`source_target_id`,`target_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;评估对象信息';

CREATE TABLE IF NOT EXISTS `eval_target_record` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `space_id` bigint unsigned NOT NULL COMMENT '空间id',
  `target_id` bigint unsigned NOT NULL COMMENT '评测对象id',
  `target_version_id` bigint unsigned NOT NULL COMMENT '版本ID',
  `experiment_run_id` bigint unsigned NOT NULL COMMENT '实验执行id',
  `item_id` bigint unsigned NOT NULL COMMENT '评测集行id',
  `turn_id` bigint unsigned NOT NULL COMMENT '评测集行轮次id',
  `log_id` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'log id',
  `trace_id` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'trace id',
  `input_data` mediumblob COMMENT '输入, json',
  `output_data` mediumblob COMMENT '输出, json',
  `status` int NOT NULL COMMENT '执行状态',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;评估对象记录信息';

CREATE TABLE IF NOT EXISTS `eval_target_version` (
  `id` bigint unsigned NOT NULL COMMENT 'target version id',
  `space_id` bigint unsigned NOT NULL COMMENT '空间id',
  `target_id` bigint unsigned NOT NULL COMMENT 'target id',
  `source_target_version` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'source target version',
  `target_meta` blob COMMENT '具体内容, 每种静态规则类型对应一个解析方式, json',
  `input_schema` blob COMMENT '评估器输入结构信息, json',
  `output_schema` blob COMMENT '评估器输出结构信息, json',
  `created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '创建人',
  `updated_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '更新人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_space_id_target_id_source_target_version` (`space_id`,`target_id`,`source_target_version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;评估对象版本信息';

CREATE TABLE IF NOT EXISTS `evaluator` (
  `id` bigint unsigned NOT NULL COMMENT 'idgen id',
  `space_id` bigint unsigned NOT NULL COMMENT '空间id',
  `evaluator_type` int unsigned NOT NULL COMMENT '评估器类型',
  `name` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '名称',
  `description` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '描述',
  `draft_submitted` tinyint(1) DEFAULT '0' COMMENT '草稿是否已提交',
  `created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '创建人',
  `updated_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '更新人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  `latest_version` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '最新版本号',
  `evaluator_info` blob COMMENT '评估器补充信息, json',
  `builtin` int unsigned NOT NULL DEFAULT '2' COMMENT '是否预置，1:是；2:否',
  `box_type` int unsigned NOT NULL DEFAULT '1' COMMENT '黑白盒类型，1:白盒；2:黑盒',
  `builtin_visible_version` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '预置评估器最新可见版本号',
  PRIMARY KEY (`id`),
  KEY `idx_space_id_evaluator_type` (`space_id`,`evaluator_type`),
  KEY `idx_space_id_created_by` (`space_id`,`created_by`),
  KEY `idx_space_id_created_at` (`space_id`,`created_at`),
  KEY `idx_space_id_updated_at` (`space_id`,`updated_at`),
  KEY `idx_space_id_name` (`space_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;评估器信息';

CREATE TABLE IF NOT EXISTS `evaluator_record` (
  `id` bigint unsigned NOT NULL COMMENT 'idgen id',
  `space_id` bigint unsigned NOT NULL COMMENT '空间id',
  `evaluator_version_id` bigint unsigned NOT NULL COMMENT '评估器版本id',
  `experiment_id` bigint unsigned DEFAULT NULL COMMENT '实验id',
  `experiment_run_id` bigint unsigned NOT NULL COMMENT '实验执行id',
  `item_id` bigint unsigned NOT NULL COMMENT '评估集行id',
  `turn_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评估集行轮次id',
  `log_id` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'log id',
  `trace_id` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'trace id',
  `score` decimal(10,4) DEFAULT NULL COMMENT '得分',
  `status` int NOT NULL COMMENT '执行状态',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  `input_data` mediumblob COMMENT '输入, json',
  `output_data` mediumblob COMMENT '执行结果, json',
  `created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '创建人',
  `updated_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '更新人',
  `ext` mediumblob COMMENT '补充信息, json',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;评估器执行结果';

CREATE TABLE IF NOT EXISTS `evaluator_tag` (
  `id` bigint unsigned NOT NULL COMMENT 'idgen id',
  `source_id` bigint unsigned NOT NULL COMMENT '资源id',
  `tag_type` int unsigned NOT NULL COMMENT 'tag类型，1:评估器；2:模板',
  `tag_key` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT 'tag键',
  `tag_value` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT 'tag值',
  `lang_type` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'zh' COMMENT '语言类型',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '创建人',
  `updated_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '更新人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_source_id` (`source_id`),
  KEY `idx_tag_type_tag_key_tag_value` (`tag_type`,`tag_key`,`tag_value`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;评估器tag';

CREATE TABLE IF NOT EXISTS `evaluator_template` (
  `id` bigint unsigned NOT NULL COMMENT 'idgen id',
  `space_id` bigint unsigned DEFAULT NULL COMMENT '空间id',
  `evaluator_type` int unsigned DEFAULT NULL COMMENT '评估器类型',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '名称',
  `description` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '版本描述',
  `metainfo` blob COMMENT '具体内容, 每种静态规则类型对应一个解析方式, json',
  `receive_chat_history` tinyint(1) DEFAULT '0' COMMENT '是否需求传递上下文',
  `input_schema` blob COMMENT '评估器结构信息, json',
  `output_schema` blob COMMENT '评估器结构信息, json',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '创建人',
  `updated_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '更新人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  `popularity` bigint unsigned NOT NULL DEFAULT '0' COMMENT '热度',
  `evaluator_info` blob COMMENT '评估器补充信息, json',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;评估器模板';

CREATE TABLE IF NOT EXISTS `evaluator_version` (
  `id` bigint unsigned NOT NULL COMMENT 'idgen id',
  `space_id` bigint unsigned NOT NULL COMMENT '空间id',
  `evaluator_type` int unsigned DEFAULT NULL COMMENT '评估器类型',
  `evaluator_id` bigint unsigned NOT NULL COMMENT '评估器id',
  `version` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '版本号',
  `description` varchar(500) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '版本描述',
  `metainfo` blob COMMENT '具体内容, 每种静态规则类型对应一个解析方式, json',
  `receive_chat_history` tinyint(1) DEFAULT '0' COMMENT '是否需求传递上下文',
  `input_schema` blob COMMENT '评估器结构信息, json',
  `created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '创建人',
  `updated_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '更新人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  `output_schema` blob COMMENT '评估器输出schema, json',
  PRIMARY KEY (`id`),
  KEY `idx_evaluator_id_version` (`evaluator_id`,`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='NDB_SHARE_TABLE;评估器版本信息';

CREATE TABLE IF NOT EXISTS `experiment` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 id',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建者 id',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '实验名称',
  `description` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '实验描述',
  `eval_set_version_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评测集版本 id',
  `target_type` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评估对象类型',
  `target_version_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评估对象版本 id',
  `eval_conf` blob COMMENT '实验评估流程配置',
  `status` int unsigned NOT NULL DEFAULT '0' COMMENT '状态',
  `status_message` blob COMMENT '状态提示信息',
  `start_at` timestamp NULL DEFAULT NULL COMMENT '开始执行时间',
  `end_at` timestamp NULL DEFAULT NULL COMMENT '结束执行时间',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  `latest_run_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '最后运行id',
  `target_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评估对象 id',
  `eval_set_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评测集 id',
  `expt_template_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '实验模板 id',
  `credit_cost` int NOT NULL DEFAULT '0' COMMENT '权益消耗模式',
  `source_type` int unsigned NOT NULL DEFAULT '1' COMMENT '实验来源类型，评测:1,自动化任务:2...',
  `source_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '实验来源id',
  `expt_type` int unsigned NOT NULL DEFAULT '1' COMMENT '实验类型，offline:1,online:2...',
  `max_alive_time` bigint unsigned DEFAULT NULL COMMENT '最大存活时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_expt_item_idx` (`space_id`,`name`,`deleted_at`),
  KEY `idx_space_deleted_created_by` (`space_id`,`created_by`,`deleted_at`),
  KEY `idx_space_deleted_status` (`space_id`,`status`,`deleted_at`),
  KEY `idx_deleted_dataset` (`space_id`,`eval_set_version_id`,`deleted_at`),
  KEY `idx_deleted_target_type` (`space_id`,`target_type`,`deleted_at`),
  KEY `idx_target_id_delete_at` (`space_id`,`target_id`,`deleted_at`),
  KEY `idx_eval_set_id_delete_at` (`space_id`,`eval_set_id`,`deleted_at`),
  KEY `idx_space_start_at` (`space_id`,`start_at`),
  KEY `idx_space_end_at` (`space_id`,`end_at`),
  KEY `idx_source_type_source_id` (`source_type`,`source_id`),
  KEY `idx_space_expt_template_id_delete_at` (`space_id`,`expt_template_id`,`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='experiment';

CREATE TABLE IF NOT EXISTS `expt_aggr_result` (
  `id` bigint unsigned NOT NULL COMMENT 'idgen id',
  `space_id` bigint unsigned NOT NULL COMMENT '空间id',
  `experiment_id` bigint unsigned NOT NULL COMMENT '实验id',
  `field_type` int DEFAULT NULL COMMENT '聚合字段类型 1：评估器得分',
  `field_key` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT '聚合字段唯一标识',
  `score` decimal(10,4) DEFAULT NULL COMMENT '聚合后的平均得分',
  `aggr_result` blob COMMENT '详细聚合结果',
  `version` bigint unsigned NOT NULL DEFAULT '0' COMMENT '版本号(用于乐观锁)',
  `status` int NOT NULL COMMENT '计算状态 1:idle 2: caculating',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_experiment_id_field_type_field_key` (`experiment_id`,`field_type`,`field_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='实验聚合结果表';

CREATE TABLE IF NOT EXISTS `expt_evaluator_ref` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 id',
  `expt_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '实验 id',
  `evaluator_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评估器 id',
  `evaluator_version_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评估器版本 id',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_space_expt` (`space_id`,`expt_id`),
  KEY `idx_space_evaluator` (`space_id`,`evaluator_id`),
  KEY `idx_space_evaluator_version` (`space_id`,`evaluator_version_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_evaluator_ref';

CREATE TABLE IF NOT EXISTS `expt_insight_analysis_feedback_comment` (
  `id` bigint unsigned NOT NULL COMMENT '唯一标识 idgen生成',
  `space_id` bigint unsigned NOT NULL COMMENT 'SpaceID',
  `expt_id` bigint unsigned NOT NULL COMMENT 'exptID',
  `analysis_record_id` bigint unsigned DEFAULT NULL COMMENT '洞察分析记录ID',
  `comment` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '评论内容',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建者 id',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_space_id_expt_id_analysis_record_id_created_by` (`space_id`,`expt_id`,`analysis_record_id`,`created_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='实验洞察分析反馈评论表';

CREATE TABLE IF NOT EXISTS `expt_insight_analysis_feedback_vote` (
  `id` bigint unsigned NOT NULL COMMENT '唯一标识 idgen生成',
  `space_id` bigint unsigned NOT NULL COMMENT 'SpaceID',
  `expt_id` bigint unsigned NOT NULL COMMENT 'exptID',
  `vote_type` int NOT NULL COMMENT '反馈类型',
  `analysis_record_id` bigint unsigned DEFAULT NULL COMMENT '洞察分析记录ID',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建者 id',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_space_id_expt_id_analysis_record_id_created_by` (`space_id`,`expt_id`,`analysis_record_id`,`created_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='实验洞察分析反馈点赞表';

CREATE TABLE IF NOT EXISTS `expt_insight_analysis_record` (
  `id` bigint unsigned NOT NULL COMMENT '唯一标识 idgen生成',
  `space_id` bigint unsigned NOT NULL COMMENT 'SpaceID',
  `expt_id` bigint unsigned NOT NULL COMMENT 'exptID',
  `status` int NOT NULL COMMENT '状态',
  `expt_result_file_path` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '原始报告文件路径',
  `analysis_report_id` bigint unsigned DEFAULT NULL COMMENT '洞察分析报告ID',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建者 id',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_space_id_expt_id` (`space_id`,`expt_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='实验洞察分析记录表';

CREATE TABLE IF NOT EXISTS `expt_item_result` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 id',
  `expt_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '实验 id',
  `expt_run_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '实验运行 id',
  `item_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'item_id',
  `item_idx` int unsigned DEFAULT NULL COMMENT 'item 序号',
  `status` int unsigned NOT NULL DEFAULT '0' COMMENT '状态',
  `err_msg` blob COMMENT '错误信息',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  `log_id` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '日志 id',
  `ext` blob COMMENT '补充信息',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_expt_item_idx` (`space_id`,`expt_id`,`item_id`),
  KEY `idx_expt_status` (`space_id`,`expt_id`,`status`),
  KEY `idx_expt_item_turn_idx` (`space_id`,`expt_id`,`item_idx`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_item_result';

CREATE TABLE IF NOT EXISTS `expt_item_result_run_log` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 id',
  `expt_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '实验 id',
  `expt_run_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '实验运行 id',
  `item_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'item_id',
  `status` int unsigned NOT NULL DEFAULT '0' COMMENT '状态',
  `err_msg` blob COMMENT '错误信息',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  `log_id` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '日志 id',
  `result_state` int DEFAULT NULL COMMENT '回写结果表状态',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_expt_run_item_turn` (`space_id`,`expt_id`,`expt_run_id`,`item_id`),
  KEY `idx_expt_item_turn` (`space_id`,`expt_id`,`item_id`),
  KEY `idx_expt_run_result_state` (`space_id`,`expt_id`,`expt_run_id`,`result_state`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_item_result_run_log';

CREATE TABLE IF NOT EXISTS `expt_result_export_record` (
  `id` bigint unsigned NOT NULL COMMENT 'export_id 导出的唯一标识 idgen生成',
  `space_id` bigint unsigned NOT NULL COMMENT 'SpaceID',
  `expt_id` bigint unsigned NOT NULL COMMENT 'exptID',
  `csv_export_status` int NOT NULL COMMENT 'CSV导出状态：1-导出中, 2-导出成功 3-导出失败',
  `file_path` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'tos文件路径',
  `start_at` timestamp NULL DEFAULT NULL COMMENT '开始执行时间',
  `end_at` timestamp NULL DEFAULT NULL COMMENT '结束执行时间',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建者 id',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  `err_msg` blob COMMENT '错误信息',
  PRIMARY KEY (`id`),
  KEY `idx_space_id_expt_id` (`space_id`,`expt_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='实验导出信息表';

CREATE TABLE IF NOT EXISTS `expt_run_log` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 id',
  `created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建者 id',
  `expt_id` bigint NOT NULL COMMENT '实验 id',
  `expt_run_id` bigint NOT NULL COMMENT '运行 id',
  `item_ids` blob COMMENT '组 ids',
  `mode` int DEFAULT NULL COMMENT '模式',
  `status` bigint DEFAULT NULL COMMENT '状态',
  `pending_cnt` int unsigned NOT NULL DEFAULT '0' COMMENT 'item 未执行数量',
  `success_cnt` int unsigned NOT NULL DEFAULT '0' COMMENT 'item 成功数量',
  `fail_cnt` int unsigned NOT NULL DEFAULT '0' COMMENT 'item 失败数量',
  `credit_cost` decimal(15,2) NOT NULL DEFAULT '0.00' COMMENT 'credit 消耗',
  `token_cost` bigint DEFAULT NULL COMMENT 'token 消耗',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  `status_message` blob COMMENT '提示信息',
  `processing_cnt` int NOT NULL DEFAULT '0' COMMENT 'processing_cnt',
  `terminated_cnt` int NOT NULL DEFAULT '0' COMMENT 'terminated_cnt',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_expt_run` (`space_id`,`expt_id`,`expt_run_id`),
  KEY `idx_expt_run_item_turn` (`space_id`,`expt_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_run_log';

CREATE TABLE IF NOT EXISTS `expt_stats` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 id',
  `expt_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '实验 id',
  `pending_cnt` int NOT NULL DEFAULT '0' COMMENT 'pending_cnt',
  `success_cnt` int NOT NULL DEFAULT '0' COMMENT 'success_cnt',
  `fail_cnt` int NOT NULL DEFAULT '0' COMMENT 'fail_cnt',
  `credit_cost` decimal(15,2) NOT NULL DEFAULT '0.00' COMMENT 'credit 消耗',
  `input_token_cost` bigint DEFAULT NULL COMMENT 'input token 消耗',
  `output_token_cost` bigint DEFAULT NULL COMMENT 'output token 消耗',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  `processing_cnt` int NOT NULL DEFAULT '0' COMMENT 'processing_cnt',
  `terminated_cnt` int NOT NULL DEFAULT '0' COMMENT 'terminated_cnt',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_space_expt` (`space_id`,`expt_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_stats';

CREATE TABLE IF NOT EXISTS `expt_template` (
  `id` bigint unsigned NOT NULL COMMENT 'id',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 id',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '实验模板名称',
  `description` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '实验模板描述',
  `eval_set_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评测集 id（模板创建后不可修改）',
  `eval_set_version_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评测集默认版本 id',
  `target_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评估对象 id（模板创建后不可修改）',
  `target_type` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评估对象类型',
  `target_version_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评估对象默认版本 id',
  `expt_type` int unsigned NOT NULL DEFAULT '1' COMMENT '实验类型，offline:1,online:2...',
  `template_conf` blob COMMENT '实验模板配置，包含评估器列表、字段映射、加权配置、默认并发及调度等，json',
  `expt_info` blob COMMENT '实验运行状态，包含创建实验数量，最后一次实验执行状态，json',
  `created_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '创建人',
  `updated_by` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '0' COMMENT '更新人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_space_id_name_deleted_at` (`space_id`,`name`,`deleted_at`),
  KEY `idx_space_id_created_by_deleted_at` (`space_id`,`created_by`,`deleted_at`),
  KEY `idx_space_id_eval_set_id_deleted_at` (`space_id`,`eval_set_id`,`deleted_at`),
  KEY `idx_space_id_target_id_deleted_at` (`space_id`,`target_id`,`deleted_at`),
  KEY `idx_space_id_expt_type_deleted_at` (`space_id`,`expt_type`,`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_template';

CREATE TABLE IF NOT EXISTS `expt_template_evaluator_ref` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 id',
  `expt_template_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '实验模板 id',
  `evaluator_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评估器 id',
  `evaluator_version_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '评估器版本 id',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_space_id_expt_template_id` (`space_id`,`expt_template_id`),
  KEY `idx_space_id_evaluator_id` (`space_id`,`evaluator_id`),
  KEY `idx_space_id_evaluator_version_id` (`space_id`,`evaluator_version_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_template_evaluator_ref';

CREATE TABLE IF NOT EXISTS `expt_turn_annotate_record_ref` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `space_id` bigint unsigned NOT NULL COMMENT '空间 id',
  `expt_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '实验 id',
  `expt_turn_result_id` bigint unsigned NOT NULL COMMENT '实验 turn result id',
  `tag_key_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '标签 id',
  `annotate_record_id` bigint unsigned NOT NULL COMMENT '人工标注结果 id',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_space_expt_turn_result_tag_key_id` (`space_id`,`expt_id`,`expt_turn_result_id`,`tag_key_id`),
  KEY `idx_turn_annotate_record_id` (`space_id`,`expt_turn_result_id`,`annotate_record_id`),
  KEY `idx_turn_tag_key_id` (`space_id`,`expt_turn_result_id`,`tag_key_id`),
  KEY `idx_space_expt_tag_key_id` (`space_id`,`expt_id`,`tag_key_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_turn_annotate_record_ref';

CREATE TABLE IF NOT EXISTS `expt_turn_evaluator_result_ref` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `space_id` bigint unsigned NOT NULL COMMENT '空间 id',
  `expt_turn_result_id` bigint unsigned NOT NULL COMMENT '实验 turn result id',
  `evaluator_version_id` bigint unsigned NOT NULL COMMENT '评估器版本 id',
  `evaluator_result_id` bigint unsigned NOT NULL COMMENT '评估器结果 id',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  `expt_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '实验 id',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_space_expt_turn_result_evaluator` (`space_id`,`expt_id`,`expt_turn_result_id`,`evaluator_version_id`),
  KEY `idx_turn_evaluator_result` (`space_id`,`expt_turn_result_id`,`evaluator_result_id`),
  KEY `idx_turn_evaluator_version` (`space_id`,`expt_turn_result_id`,`evaluator_version_id`),
  KEY `idx_expt_evaluator_result` (`space_id`,`expt_id`,`evaluator_result_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_turn_evaluator_result_ref';

CREATE TABLE IF NOT EXISTS `expt_turn_result` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `space_id` bigint unsigned NOT NULL COMMENT '空间 id',
  `expt_id` bigint unsigned NOT NULL COMMENT '实验 id',
  `expt_run_id` bigint unsigned NOT NULL COMMENT '实验运行 id',
  `item_id` bigint unsigned NOT NULL COMMENT 'item_id',
  `turn_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'turn_id',
  `turn_idx` int unsigned DEFAULT NULL COMMENT 'turn 序号',
  `status` int unsigned NOT NULL DEFAULT '0' COMMENT '状态',
  `trace_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'trace_id',
  `log_id` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '日志 id',
  `target_result_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'target_result_id',
  `err_msg` blob COMMENT '错误信息',
  `weighted_score` decimal(10,4) DEFAULT NULL COMMENT '加权汇总得分',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_expt_item_turn` (`space_id`,`expt_id`,`item_id`,`turn_id`),
  KEY `idx_expt_status` (`space_id`,`expt_id`,`status`),
  KEY `idx_expt_item_turn_idx` (`space_id`,`expt_id`,`item_id`,`turn_idx`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_turn_result';

CREATE TABLE IF NOT EXISTS `expt_turn_result_filter_key_mapping` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `space_id` bigint NOT NULL COMMENT '空间id',
  `expt_id` bigint NOT NULL COMMENT '实验id',
  `from_field` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '筛选项唯一键，评估器: evaluator_version_id，人工标准：tag_key_id',
  `to_key` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'ck侧的map key，评估器：key1 ~ key10，人工标准：key1 ~ key100',
  `field_type` int NOT NULL COMMENT '映射类型，Evaluator —— 1，人工标注—— 2',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  `created_by` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建人',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_idx_space_expt_from_type` (`space_id`,`expt_id`,`field_type`,`from_field`)
) ENGINE=InnoDB AUTO_INCREMENT=6692 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_turn_result_filter二级key映射表';

CREATE TABLE IF NOT EXISTS `expt_turn_result_run_log` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `space_id` bigint unsigned NOT NULL COMMENT '空间 id',
  `expt_id` bigint unsigned NOT NULL COMMENT '实验 id',
  `expt_run_id` bigint unsigned NOT NULL COMMENT '实验运行 id',
  `item_id` bigint unsigned NOT NULL COMMENT 'item_id',
  `turn_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'turn_id',
  `status` int unsigned NOT NULL DEFAULT '0' COMMENT '状态',
  `trace_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'trace_id',
  `log_id` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '日志 id',
  `target_result_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'target_result_id',
  `evaluator_result_ids` blob COMMENT 'evaluator_result_ids，json list 格式',
  `err_msg` blob COMMENT '错误信息',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_expt_run_item_turn` (`space_id`,`expt_id`,`expt_run_id`,`item_id`,`turn_id`),
  KEY `idx_expt_item_turn` (`space_id`,`expt_id`,`item_id`,`turn_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_item_result_run_log';

CREATE TABLE IF NOT EXISTS `expt_turn_result_tag_ref` (
  `id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'id',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 id',
  `expt_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '实验 id',
  `tag_key_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '标签 id',
  `total_cnt` int NOT NULL DEFAULT '0' COMMENT 'total_cnt',
  `complete_cnt` int NOT NULL DEFAULT '0' COMMENT 'complete_cnt',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` timestamp NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_space_tag_key_id` (`space_id`,`expt_id`,`tag_key_id`),
  KEY `idx_space_expt` (`space_id`,`expt_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='expt_turn_result_tag_ref';

CREATE TABLE IF NOT EXISTS `model_request_record` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间id',
  `user_id` varchar(256) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'user id',
  `usage_scene` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '场景',
  `usage_scene_entity_id` varchar(256) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '场景实体id',
  `frame` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '使用的框架，如eino',
  `protocol` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '使用的协议，如ark/deepseek等',
  `model_identification` varchar(1024) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '模型唯一标识',
  `model_ak` varchar(1024) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '模型的AK',
  `model_id` varchar(256) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'model id',
  `model_name` varchar(1024) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '模型展示名称',
  `input_token` bigint unsigned NOT NULL DEFAULT '0' COMMENT '输入token数量',
  `output_token` bigint unsigned NOT NULL DEFAULT '0' COMMENT '输出token数量',
  `logid` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'logid',
  `error_code` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'error_code',
  `error_msg` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'error_msg',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_space_id_create_time` (`space_id`,`created_at`) USING BTREE COMMENT 'space_id_create_time'
) ENGINE=InnoDB AUTO_INCREMENT=21 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='模型流量记录开源表';

CREATE TABLE IF NOT EXISTS `observability_trajectory_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `workspace_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 ID',
  `filter` json DEFAULT NULL COMMENT 'trace展示的过滤配置',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建人',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `updated_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '修改人',
  `is_deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除, 0 表示未删除, 1 表示已删除',
  `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
  `deleted_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '删除人',
  PRIMARY KEY (`id`),
  KEY `idx_space_id` (`workspace_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='观测轨迹配置';

CREATE TABLE IF NOT EXISTS `observability_view` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `enterprise_id` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '企业id',
  `workspace_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间 ID',
  `view_name` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '视图名称',
  `platform_type` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '数据来源',
  `span_list_type` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '列表信息',
  `filters` varchar(2048) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '过滤条件信息',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建人',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
  `updated_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '修改人',
  `is_deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除, 0 表示未删除, 1 表示已删除',
  `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
  `deleted_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '删除人',
  PRIMARY KEY (`id`),
  KEY `idx_space_id_created_by` (`workspace_id`,`created_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='观测视图信息';

CREATE TABLE IF NOT EXISTS `prompt_basic` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `space_id` bigint unsigned NOT NULL COMMENT '空间ID',
  `prompt_key` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT 'Prompt key',
  `name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT 'Prompt名称',
  `description` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT '描述',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT '创建人',
  `updated_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT '更新人',
  `commit_status` tinyint NOT NULL DEFAULT '0' COMMENT '提交状态',
  `latest_version` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT '最新版本',
  `latest_commit_time` datetime DEFAULT NULL COMMENT '最新提交时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT '删除时间',
  `prompt_type` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'normal' COMMENT 'Prompt类型',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_space_id_prompt_key_deleted_at` (`space_id`,`prompt_key`,`deleted_at`),
  KEY `idx_created_at` (`created_at`) USING BTREE,
  KEY `idx_pid_ptype_delat` (`space_id`,`prompt_type`,`deleted_at`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=7603228121373868034 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Prompt基础表';

CREATE TABLE IF NOT EXISTS `prompt_commit` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `space_id` bigint unsigned NOT NULL COMMENT '空间ID',
  `prompt_id` bigint unsigned NOT NULL COMMENT 'Prompt ID',
  `prompt_key` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'Prompt key',
  `template_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT 'normal' COMMENT '模版类型',
  `messages` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '托管消息列表',
  `model_config` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '模型配置',
  `variable_defs` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '变量定义',
  `tools` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'tools',
  `tool_call_config` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'tool调用配置',
  `version` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '版本',
  `base_version` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '来源版本',
  `committed_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '提交人',
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '提交版本描述',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `ext_info` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'Extended information field',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_prompt_id_version` (`prompt_id`,`version`),
  KEY `idx_prompt_key_version` (`prompt_key`,`version`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=7540967127478435842 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Commit表';

CREATE TABLE IF NOT EXISTS `prompt_commit_label_mapping` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `space_id` bigint unsigned NOT NULL COMMENT '空间ID',
  `prompt_id` bigint unsigned NOT NULL COMMENT 'Prompt ID',
  `label_key` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT 'Label唯一标识',
  `prompt_version` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT 'Prompt版本',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT '创建人',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT '更新人',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_prompt_id_label_key_deleted_at` (`prompt_id`,`label_key`,`deleted_at`),
  KEY `idx_prompt_id_version` (`prompt_id`,`prompt_version`) USING BTREE,
  KEY `idx_created_at` (`created_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Prompt提交版本和Label关联表';

CREATE TABLE IF NOT EXISTS `prompt_debug_context` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `prompt_id` bigint NOT NULL DEFAULT '0' COMMENT 'prompt id',
  `user_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'user id',
  `mock_contexts` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '上下文信息，json格式',
  `mock_variables` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'mock变量值，json格式',
  `mock_tools` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'mock tool结果，json格式',
  `debug_config` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '调试配置',
  `compare_config` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '训练场配置',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_prompt_id_user_id` (`prompt_id`,`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7553200131282042882 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='用户调试prompt上下文信息表';

CREATE TABLE IF NOT EXISTS `prompt_debug_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `prompt_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Prompt ID',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间ID',
  `prompt_key` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'prompt key',
  `version` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'version',
  `input_tokens` bigint NOT NULL DEFAULT '0' COMMENT 'input_tokens',
  `output_tokens` bigint NOT NULL DEFAULT '0' COMMENT 'output_tokens',
  `started_at` bigint unsigned DEFAULT '0' COMMENT '请求开始毫秒时间戳',
  `ended_at` bigint unsigned DEFAULT '0' COMMENT '响应结束毫秒时间戳',
  `cost_ms` bigint unsigned DEFAULT '0' COMMENT '响应耗时毫秒',
  `status_code` int DEFAULT NULL COMMENT '状态码',
  `debugged_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '0' COMMENT '执行人UserID',
  `debug_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'debug_id',
  `debug_step` int NOT NULL DEFAULT '1' COMMENT 'debug_step',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_prompt_id_debugged_by_started_at` (`prompt_id`,`debugged_by`,`started_at`) USING BTREE,
  KEY `idx_debug_id_step` (`debug_id`,`debug_step`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=7553200127742050306 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='debug表';

CREATE TABLE IF NOT EXISTS `prompt_label` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `space_id` bigint unsigned NOT NULL COMMENT '空间ID',
  `label_key` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT 'Label唯一标识',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT '创建人',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT '更新人',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_space_id_label_key_deleted_at` (`space_id`,`label_key`,`deleted_at`),
  KEY `idx_created_at` (`created_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Prompt Label表';

CREATE TABLE IF NOT EXISTS `prompt_relation` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `space_id` bigint unsigned NOT NULL COMMENT '空间ID',
  `main_prompt_id` bigint unsigned NOT NULL COMMENT '主Prompt ID',
  `main_prompt_version` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '主Prompt版本',
  `main_draft_user_id` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '主Prompt草稿Owner',
  `sub_prompt_id` bigint unsigned NOT NULL COMMENT '子Prompt ID',
  `sub_prompt_version` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '子Prompt版本',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_main_prompt_id_version` (`main_prompt_id`,`main_prompt_version`) COMMENT '主prompt_id_版本',
  KEY `idx_main_prompt_id_user` (`main_prompt_id`,`main_draft_user_id`) COMMENT '主prompt_id_user',
  KEY `idx_sub_prompt_id_version_create_time` (`sub_prompt_id`,`sub_prompt_version`,`create_time`) COMMENT '子prompt_id_版本'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Prompt关联表';

CREATE TABLE IF NOT EXISTS `prompt_user_draft` (
  `id` bigint unsigned NOT NULL COMMENT '主键ID',
  `space_id` bigint unsigned NOT NULL COMMENT '空间ID',
  `prompt_id` bigint unsigned NOT NULL COMMENT 'Prompt ID',
  `user_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '用户ID',
  `template_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT 'Normal' COMMENT '模版类型',
  `messages` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '托管消息列表',
  `model_config` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '模型配置',
  `variable_defs` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT '变量定义',
  `tools` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'tools',
  `tool_call_config` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'tool调用配置',
  `base_version` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '草稿关联版本',
  `is_draft_edited` tinyint NOT NULL DEFAULT '0' COMMENT '草稿内容是否基于BaseVersion有变更',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT '删除时间',
  `ext_info` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'Extended information field',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_prompt_id_user_id_deleted_at` (`prompt_id`,`user_id`,`deleted_at`),
  KEY `idx_prompt_id_user_id` (`prompt_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Draft表';

CREATE TABLE IF NOT EXISTS `space` (
  `id` bigint unsigned NOT NULL COMMENT 'Primary Key ID, Space ID',
  `owner_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Owner ID',
  `name` varchar(200) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Space Name',
  `description` varchar(2000) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Space Description',
  `space_type` tinyint NOT NULL DEFAULT '0' COMMENT 'Space Type, 1: Personal, 2: Team',
  `icon_uri` varchar(200) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Icon URI',
  `created_by` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Creator ID',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT '删除时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_owner_id` (`owner_id`),
  KEY `idx_creator_id` (`created_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Space Table';

CREATE TABLE IF NOT EXISTS `space_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary Key ID, Auto Increment',
  `space_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Space ID',
  `user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'User ID',
  `role_type` int NOT NULL DEFAULT '3' COMMENT 'Role Type: 1.owner 2.admin 3.member',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT '删除时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_space_user` (`space_id`,`user_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=125 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Space Member Table';

CREATE TABLE IF NOT EXISTS `tag_key` (
  `id` bigint unsigned NOT NULL COMMENT '主键id',
  `app_id` int NOT NULL DEFAULT '0' COMMENT 'application id',
  `space_id` bigint unsigned NOT NULL COMMENT '归属space id,做分片键',
  `version_num` int NOT NULL DEFAULT '0' COMMENT 'tag自增版本',
  `version` varchar(64) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'tag版本文案',
  `tag_key_id` bigint unsigned NOT NULL COMMENT 'tag id，唯一标识一个标签',
  `tag_key_name` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'tag名称',
  `description` varchar(2000) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'tag描述',
  `parent_key_id` bigint unsigned DEFAULT NULL COMMENT '级联标签场景,上层tag key id',
  `created_at` timestamp NOT NULL COMMENT '创建时间',
  `updated_at` timestamp NOT NULL COMMENT '更新时间',
  `change_log` json NOT NULL COMMENT '变更日志',
  `status` varchar(32) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'tag状态,active,inactive,deprecated',
  `tag_type` varchar(32) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'tag类型,tag,option',
  `created_by` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建者',
  `updated_by` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '更新者',
  `tag_target_type` varchar(256) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'tag目标对象列表,resource|dataset_item,多个值由,分隔',
  `content_type` varchar(64) COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'categorical' COMMENT '内容类型: 自由文本,连续分值,分类,布尔值',
  `spec` json DEFAULT NULL COMMENT '标签规格',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_space_id_tag_key_id_version_num_tag_type` (`space_id`,`tag_key_id`,`version_num`,`tag_type`),
  KEY `idx_space_id_tag_type_status_tag_key_name` (`space_id`,`tag_type`,`status`,`tag_key_name`),
  KEY `idx_space_id_tag_type_status_tag_created_by_tag_key_name` (`space_id`,`tag_type`,`status`,`created_by`,`tag_key_name`),
  KEY `idx_space_id_tag_type_status_content_type` (`space_id`,`tag_type`,`status`,`content_type`),
  KEY `idx_space_id_tag_type_status_updated_at` (`space_id`,`tag_type`,`status`,`updated_at`),
  KEY `idx_space_id_tag_type_status_created_at` (`space_id`,`tag_type`,`status`,`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='tag元数据表';

CREATE TABLE IF NOT EXISTS `tag_value` (
  `id` bigint unsigned NOT NULL COMMENT '主键id',
  `app_id` int NOT NULL DEFAULT '0' COMMENT 'application id',
  `space_id` bigint unsigned NOT NULL COMMENT '归属space id,做分片键',
  `tag_key_id` bigint unsigned NOT NULL COMMENT 'tag id，唯一标识一个标签',
  `tag_value_id` bigint unsigned NOT NULL COMMENT 'tag value id，唯一标识一个标签',
  `tag_value_name` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'tag value名称',
  `description` varchar(2000) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'tag value描述',
  `parent_value_id` bigint unsigned NOT NULL COMMENT '级联标签场景,上层tag value id',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间',
  `version_num` int NOT NULL DEFAULT '0' COMMENT 'tag自增版本',
  `status` varchar(32) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '状态,active,inactive,deprecated',
  `created_by` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '创建者',
  `updated_by` varchar(64) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '更新者',
  PRIMARY KEY (`id`),
  KEY `idx_space_id_tag_key_id_version_num` (`space_id`,`tag_key_id`,`version_num`),
  KEY `idx_space_id_tag_value_id_version_num` (`space_id`,`tag_value_id`,`version_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='tag value元数据表';

CREATE TABLE IF NOT EXISTS `task` (
  `id` bigint unsigned NOT NULL COMMENT 'Task ID',
  `workspace_id` bigint unsigned NOT NULL COMMENT '空间ID',
  `name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '任务名称',
  `description` varchar(2048) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '' COMMENT '任务描述',
  `task_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '任务类型',
  `task_status` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '任务状态',
  `task_detail` json DEFAULT NULL COMMENT '任务运行状态详情',
  `span_filter` json DEFAULT NULL COMMENT 'span 过滤条件',
  `effective_time` json DEFAULT NULL COMMENT '生效时间',
  `backfill_effective_time` json DEFAULT NULL COMMENT '历史回溯生效时间',
  `sampler` json DEFAULT NULL COMMENT '采样器',
  `task_config` json DEFAULT NULL COMMENT '相关任务的配置信息',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间',
  `created_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '创建人',
  `updated_by` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '更新人',
  `task_source` varchar(50) COLLATE utf8mb4_general_ci DEFAULT 'user' COMMENT '任务来源',
  PRIMARY KEY (`id`),
  KEY `idx_space_id_status` (`workspace_id`,`task_status`),
  KEY `idx_space_id_type` (`workspace_id`,`task_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='任务信息';

CREATE TABLE IF NOT EXISTS `user` (
  `id` bigint NOT NULL COMMENT 'Primary Key ID',
  `name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'User Nickname',
  `unique_name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'User Unique Name',
  `email` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'Email',
  `password` varchar(128) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'Password (Encrypted)',
  `description` varchar(512) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'User Description',
  `icon_uri` varchar(512) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'Avatar URI',
  `user_verified` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'User Verification Status',
  `country_code` bigint NOT NULL DEFAULT '0' COMMENT 'Country Code',
  `session_key` varchar(512) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'Session Key',
  `deleted_at` bigint NOT NULL DEFAULT '0' COMMENT '删除时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_name` (`unique_name`),
  UNIQUE KEY `idx_email` (`email`),
  KEY `idx_session_key` (`session_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='User Table';
