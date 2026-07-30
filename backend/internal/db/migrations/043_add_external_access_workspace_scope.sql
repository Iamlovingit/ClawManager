SET @workspace_access_exists = (
  SELECT COUNT(*)
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'instance_external_access'
    AND COLUMN_NAME = 'workspace_access'
);
SET @workspace_access_sql = IF(
  @workspace_access_exists = 0,
  'ALTER TABLE instance_external_access ADD COLUMN workspace_access VARCHAR(16) NOT NULL DEFAULT ''none'' AFTER auth_mode',
  'SELECT 1'
);
PREPARE workspace_access_stmt FROM @workspace_access_sql;
EXECUTE workspace_access_stmt;
DEALLOCATE PREPARE workspace_access_stmt;
