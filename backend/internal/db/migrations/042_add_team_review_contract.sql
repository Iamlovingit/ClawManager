ALTER TABLE team_work_items
  ADD COLUMN review_target_assignment_id VARCHAR(255) NULL AFTER validated_revision,
  ADD COLUMN review_target_revision INT NULL AFTER review_target_assignment_id,
  ADD INDEX idx_team_work_items_review_target (root_task_id, review_target_assignment_id, review_target_revision);
