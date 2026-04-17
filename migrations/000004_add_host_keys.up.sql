-- Add host key verification columns to ssh_configs table
ALTER TABLE ssh_configs
    ADD COLUMN IF NOT EXISTS host_key_fingerprint VARCHAR(255),
    ADD COLUMN IF NOT EXISTS host_key_algorithm VARCHAR(50),
    ADD COLUMN IF NOT EXISTS known_hosts_entry TEXT;

-- Create index for efficient host key lookups
CREATE INDEX IF NOT EXISTS idx_ssh_configs_host_key ON ssh_configs(host_key_fingerprint);

-- Comment on new columns
COMMENT ON COLUMN ssh_configs.host_key_fingerprint IS 'SSH host key fingerprint (SHA256) for verification';
COMMENT ON COLUMN ssh_configs.host_key_algorithm IS 'SSH host key algorithm (e.g., ssh-rsa, ecdsa-sha2-nistp256)';
COMMENT ON COLUMN ssh_configs.known_hosts_entry IS 'Full known_hosts entry for the host key';
