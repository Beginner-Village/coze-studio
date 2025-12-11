-- 为 model_entity 表添加 is_public 字段
ALTER TABLE `opencoze`.`model_entity`
ADD COLUMN `is_public` TINYINT(1) NOT NULL DEFAULT 0
COMMENT '是否为公共模型: 0=私有, 1=公共' AFTER `status`;

ALTER TABLE `opencoze`.`model_entity`
ADD INDEX `idx_is_public` (`is_public`);

-- 为 model_meta 表添加 is_public 字段
ALTER TABLE `opencoze`.`model_meta`
ADD COLUMN `is_public` TINYINT(1) NOT NULL DEFAULT 0
COMMENT '是否为公共模型: 0=私有, 1=公共' AFTER `status`;

-- 创建管理员用户表
CREATE TABLE IF NOT EXISTS `opencoze`.`admin_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` bigint unsigned NOT NULL COMMENT '关联的用户ID',
  `role` varchar(32) NOT NULL DEFAULT 'admin' COMMENT '角色: super_admin/admin',
  `created_at` bigint unsigned NOT NULL DEFAULT 0 COMMENT '创建时间（毫秒）',
  `created_by` bigint unsigned NOT NULL DEFAULT 0 COMMENT '创建者ID',
  `updated_at` bigint unsigned NOT NULL DEFAULT 0 COMMENT '更新时间（毫秒）',
  `deleted_at` bigint unsigned NULL COMMENT '删除时间（毫秒）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_id` (`user_id`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='管理员用户表';
