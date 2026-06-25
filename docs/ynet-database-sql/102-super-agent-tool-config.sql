-- ============================================================
-- Super-agent capability switches (per-agent)
-- Stores the super-agent tool/capability config as JSON: sandbox master
-- switch (off = pure-MCP mode), per-tool permissions (web_search/web_fetch/
-- run_bash/deep_task/skill_manage) and the dynamic MCP server list.
-- NULL = all enabled (backward compatible: existing super-agents keep full power).
-- ============================================================

ALTER TABLE `single_agent_draft` ADD COLUMN `super_agent_tool_config` json DEFAULT NULL COMMENT 'Super-agent capability switches (sandbox/web/tools + MCP servers)';
