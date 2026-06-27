-- ============================================================
-- Skill marketplace publish review (review_status)
-- Adds a platform-review layer on top of publishing. Global-scope skills
-- enter `pending` and only appear in the global marketplace once approved.
-- Space/private scope skills are self-governed and default to approved.
-- Existing rows default to 2 (approved) so current marketplace behavior is
-- unchanged.
-- ============================================================

ALTER TABLE `skill` ADD COLUMN `review_status` tinyint(4) NOT NULL DEFAULT 2 COMMENT '1 pending, 2 approved, 3 rejected';
ALTER TABLE `skill` ADD COLUMN `review_note` varchar(512) DEFAULT NULL COMMENT 'reviewer note';
ALTER TABLE `skill` ADD COLUMN `reviewer_id` bigint(20) NOT NULL DEFAULT 0 COMMENT 'reviewer user id';
ALTER TABLE `skill` ADD COLUMN `reviewed_at` bigint(20) NOT NULL DEFAULT 0 COMMENT 'review timestamp in milliseconds';

CREATE INDEX `idx_skill_review_status_scope` ON `skill` (`review_status`, `publish_scope`);
