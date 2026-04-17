-- Rollback initial schema

-- Drop view
DROP VIEW IF EXISTS node_details;

-- Drop triggers
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_bmc_configs_updated_at ON bmc_configs;
DROP TRIGGER IF EXISTS update_ssh_configs_updated_at ON ssh_configs;
DROP TRIGGER IF EXISTS update_nodes_updated_at ON nodes;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (in reverse dependency order)
DROP TABLE IF EXISTS agent_certificates;
DROP TABLE IF EXISTS audit_logs_2026_04;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS os_images;
DROP TABLE IF EXISTS node_firmware;
DROP TABLE IF EXISTS firmware;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS bmc_configs;
DROP TABLE IF EXISTS ssh_configs;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS nodes;

-- Drop extensions
DROP EXTENSION IF EXISTS "uuid-ossp";
