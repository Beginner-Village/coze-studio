ALTER TABLE `single_agent_draft` ADD COLUMN `force_tool_return` TINYINT(1) DEFAULT NULL COMMENT 'Force tool return to model' AFTER `skill_info_list`;
ALTER TABLE `single_agent_version` ADD COLUMN `force_tool_return` TINYINT(1) DEFAULT NULL COMMENT 'Force tool return to model' AFTER `skill_info_list`;
