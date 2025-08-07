-- Add missing bot_mode and layout_info columns to single_agent_draft table
ALTER TABLE `opencoze`.`single_agent_draft` 
ADD COLUMN `bot_mode` tinyint NOT NULL DEFAULT 0 COMMENT 'mod,0:single mode 2:chatflow mode',
ADD COLUMN `layout_info` json NULL COMMENT 'chatflow layout info';

-- Add missing bot_mode and layout_info columns to single_agent_version table  
ALTER TABLE `opencoze`.`single_agent_version`
ADD COLUMN `bot_mode` tinyint NOT NULL DEFAULT 0 COMMENT 'mod,0:single mode 2:chatflow mode',
ADD COLUMN `layout_info` json NULL COMMENT 'chatflow layout info';