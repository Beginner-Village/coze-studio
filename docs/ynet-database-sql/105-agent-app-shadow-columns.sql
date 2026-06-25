ALTER TABLE `single_agent_draft`
  ADD COLUMN `source_product_id` bigint(20) NOT NULL DEFAULT 0
    COMMENT 'agent_app product id this draft was materialized from; 0 = normal agent';
ALTER TABLE `single_agent_draft`
  ADD COLUMN `source_product_version` varchar(64) NOT NULL DEFAULT ''
    COMMENT 'pinned agent_app product version for this shadow instance';
