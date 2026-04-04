SET @instance_agent_bootstrap_token_column_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'instances'
    AND COLUMN_NAME = 'agent_bootstrap_token'
);
SET @instance_agent_bootstrap_token_column_sql = IF(
  @instance_agent_bootstrap_token_column_exists = 0,
  'ALTER TABLE instances ADD COLUMN agent_bootstrap_token VARCHAR(255) NULL AFTER access_token',
  'SELECT 1'
);
PREPARE instance_agent_bootstrap_token_column_stmt FROM @instance_agent_bootstrap_token_column_sql;
EXECUTE instance_agent_bootstrap_token_column_stmt;
DEALLOCATE PREPARE instance_agent_bootstrap_token_column_stmt;

CREATE TABLE IF NOT EXISTS instance_agents (
  id INT AUTO_INCREMENT PRIMARY KEY,
  instance_id INT NOT NULL,
  agent_id VARCHAR(255) NOT NULL,
  agent_version VARCHAR(50) NOT NULL,
  protocol_version VARCHAR(50) NOT NULL,
  status VARCHAR(30) NOT NULL DEFAULT 'online',
  capabilities_json LONGTEXT NOT NULL,
  host_info_json LONGTEXT NULL,
  session_token VARCHAR(255) NULL,
  session_expires_at TIMESTAMP NULL,
  last_heartbeat_at TIMESTAMP NULL,
  last_reported_at TIMESTAMP NULL,
  last_seen_ip VARCHAR(45) NULL,
  registered_at TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (instance_id) REFERENCES instances(id) ON DELETE CASCADE,
  UNIQUE KEY uk_instance_agents_instance (instance_id),
  UNIQUE KEY uk_instance_agents_session_token (session_token),
  INDEX idx_instance_agents_agent_id (agent_id),
  INDEX idx_instance_agents_status (status),
  INDEX idx_instance_agents_last_heartbeat (last_heartbeat_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS instance_runtime_status (
  id INT AUTO_INCREMENT PRIMARY KEY,
  instance_id INT NOT NULL,
  infra_status VARCHAR(30) NOT NULL DEFAULT 'creating',
  agent_status VARCHAR(30) NOT NULL DEFAULT 'offline',
  openclaw_status VARCHAR(30) NOT NULL DEFAULT 'unknown',
  openclaw_pid INT NULL,
  openclaw_version VARCHAR(100) NULL,
  current_config_revision_id INT NULL,
  desired_config_revision_id INT NULL,
  summary_json LONGTEXT NULL,
  system_info_json LONGTEXT NULL,
  health_json LONGTEXT NULL,
  last_reported_at TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (instance_id) REFERENCES instances(id) ON DELETE CASCADE,
  UNIQUE KEY uk_instance_runtime_status_instance (instance_id),
  INDEX idx_instance_runtime_status_agent_status (agent_status),
  INDEX idx_instance_runtime_status_openclaw_status (openclaw_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS instance_desired_state (
  id INT AUTO_INCREMENT PRIMARY KEY,
  instance_id INT NOT NULL,
  desired_power_state VARCHAR(30) NOT NULL DEFAULT 'running',
  desired_config_revision_id INT NULL,
  desired_runtime_action VARCHAR(50) NULL,
  updated_by INT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (instance_id) REFERENCES instances(id) ON DELETE CASCADE,
  FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL,
  UNIQUE KEY uk_instance_desired_state_instance (instance_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS instance_commands (
  id INT AUTO_INCREMENT PRIMARY KEY,
  instance_id INT NOT NULL,
  agent_id VARCHAR(255) NULL,
  command_type VARCHAR(50) NOT NULL,
  payload_json LONGTEXT NULL,
  status VARCHAR(30) NOT NULL DEFAULT 'pending',
  idempotency_key VARCHAR(255) NOT NULL,
  issued_by INT NULL,
  issued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  dispatched_at TIMESTAMP NULL,
  started_at TIMESTAMP NULL,
  finished_at TIMESTAMP NULL,
  timeout_seconds INT NOT NULL DEFAULT 300,
  result_json LONGTEXT NULL,
  error_message TEXT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (instance_id) REFERENCES instances(id) ON DELETE CASCADE,
  FOREIGN KEY (issued_by) REFERENCES users(id) ON DELETE SET NULL,
  UNIQUE KEY uk_instance_commands_idempotency (instance_id, idempotency_key),
  INDEX idx_instance_commands_instance_status (instance_id, status),
  INDEX idx_instance_commands_agent_status (agent_id, status),
  INDEX idx_instance_commands_issued_at (issued_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS instance_config_revisions (
  id INT AUTO_INCREMENT PRIMARY KEY,
  instance_id INT NOT NULL,
  source_snapshot_id INT NULL,
  source_bundle_id INT NULL,
  revision_no INT NOT NULL,
  content_json LONGTEXT NOT NULL,
  checksum VARCHAR(255) NOT NULL,
  status VARCHAR(30) NOT NULL DEFAULT 'published',
  published_by INT NULL,
  published_at TIMESTAMP NULL,
  activated_at TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (instance_id) REFERENCES instances(id) ON DELETE CASCADE,
  FOREIGN KEY (published_by) REFERENCES users(id) ON DELETE SET NULL,
  UNIQUE KEY uk_instance_config_revision_unique (instance_id, revision_no),
  INDEX idx_instance_config_revision_instance (instance_id, revision_no),
  INDEX idx_instance_config_revision_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
