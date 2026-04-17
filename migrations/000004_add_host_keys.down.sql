-- Remove host key verification columns from ssh_configs table
DROP INDEX IF EXISTS idx_ssh_configs_host_key;

ALTER TABLE ssh_configs
    DROP COLUMN IF EXISTS host_key_fingerprint,
    DROP COLUMN IF EXISTS host_key_algorithm,
    DROP COLUMN IF EXISTS known_hosts_entry;
