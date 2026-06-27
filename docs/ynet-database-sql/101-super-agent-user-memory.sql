-- ============================================================
-- Super-agent per-user long-term memory
-- Replaces the old flat string-array sandbox file
-- (/workspace/.agent/memory.json) with structured, per-user,
-- DB-backed memory so the super-agent "grows with you" across
-- conversations and across agents.
--
-- Scope: per user_id (space_id / agent_id are provenance + optional
-- filters, NOT isolation). Storage: OB/MySQL. Relevance recall in app
-- layer (content/tags match now; vector recall via OB added later).
-- ============================================================

CREATE TABLE IF NOT EXISTS `super_agent_user_memory` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) NOT NULL COMMENT 'owner user id (per-user scope)',
  `space_id` bigint(20) NOT NULL DEFAULT '0' COMMENT 'provenance / optional filter',
  `agent_id` bigint(20) NOT NULL DEFAULT '0' COMMENT 'agent that observed it (provenance)',
  `kind` tinyint(4) NOT NULL DEFAULT '3' COMMENT '1 profile, 2 preference, 3 fact, 4 project, 5 feedback',
  `mem_key` varchar(128) NOT NULL DEFAULT '' COMMENT 'stable key for upsert/supersede; app-level unique per (user_id, mem_key) when non-empty',
  `content` text NOT NULL COMMENT 'memory content',
  `tags` varchar(512) NOT NULL DEFAULT '' COMMENT 'comma/JSON tags to aid recall',
  `source_conversation_id` bigint(20) NOT NULL DEFAULT '0' COMMENT 'conversation that produced it',
  `source_run_id` bigint(20) NOT NULL DEFAULT '0' COMMENT 'run that produced it',
  `status` tinyint(4) NOT NULL DEFAULT '1' COMMENT '1 active, 2 archived/superseded',
  `created_at` bigint(20) NOT NULL DEFAULT '0' COMMENT 'create time in ms',
  `updated_at` bigint(20) NOT NULL DEFAULT '0' COMMENT 'update time in ms',
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_status` (`user_id`, `status`),
  KEY `idx_user_kind_status` (`user_id`, `kind`, `status`),
  KEY `idx_user_memkey` (`user_id`, `mem_key`)
) DEFAULT CHARSET=utf8mb4;
