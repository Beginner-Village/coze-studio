ALTER TABLE `single_agent_draft` ADD COLUMN `agent_type` VARCHAR(64) DEFAULT NULL COMMENT 'Agent Type for Runtime Routing' AFTER `force_tool_return`;
ALTER TABLE `single_agent_version` ADD COLUMN `agent_type` VARCHAR(64) DEFAULT NULL COMMENT 'Agent Type for Runtime Routing' AFTER `force_tool_return`;
