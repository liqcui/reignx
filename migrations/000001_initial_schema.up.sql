-- ReignX Initial Database Schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Nodes table - all managed servers
CREATE TABLE nodes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hostname VARCHAR(255) NOT NULL,
    ip_address INET NOT NULL UNIQUE,
    mac_address MACADDR,
    mode VARCHAR(20) NOT NULL DEFAULT 'ssh', -- ssh, agent, hybrid
    status VARCHAR(50) NOT NULL DEFAULT 'offline', -- online, offline, maintenance, installing, upgrading, failed
    os_type VARCHAR(50) NOT NULL,
    os_version VARCHAR(100),
    architecture VARCHAR(50),
    agent_version VARCHAR(50),
    last_seen TIMESTAMPTZ,
    tags JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_nodes_hostname ON nodes(hostname);
CREATE INDEX idx_nodes_ip_address ON nodes(ip_address);
CREATE INDEX idx_nodes_mode ON nodes(mode);
CREATE INDEX idx_nodes_status ON nodes(status);
CREATE INDEX idx_nodes_last_seen ON nodes(last_seen);
CREATE INDEX idx_nodes_tags ON nodes USING GIN(tags);

-- SSH configurations table
CREATE TABLE ssh_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL DEFAULT 22,
    "user" VARCHAR(100) NOT NULL,
    key_path VARCHAR(500),
    password_encrypted TEXT,
    jump_host VARCHAR(255),
    timeout INTEGER DEFAULT 30,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(node_id)
);

CREATE INDEX idx_ssh_configs_node_id ON ssh_configs(node_id);

-- BMC/IPMI configurations table
CREATE TABLE bmc_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    address VARCHAR(255) NOT NULL,
    "user" VARCHAR(100) NOT NULL,
    password_encrypted TEXT NOT NULL,
    type VARCHAR(20) NOT NULL, -- ipmi, redfish
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(node_id)
);

CREATE INDEX idx_bmc_configs_node_id ON bmc_configs(node_id);

-- Jobs table - batch operations
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- command, script, file, firmware, os_install, os_upgrade, package, patch, reboot
    mode VARCHAR(20) NOT NULL, -- ssh, agent, auto
    filter JSONB NOT NULL,
    template JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed, cancelled
    batch_size INTEGER DEFAULT 10,
    concurrency INTEGER DEFAULT 10,
    total_tasks INTEGER DEFAULT 0,
    completed INTEGER DEFAULT 0,
    failed INTEGER DEFAULT 0,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    parameters JSONB DEFAULT '{}'
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_type ON jobs(type);
CREATE INDEX idx_jobs_created_by ON jobs(created_by);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);

-- Tasks table - individual operations
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    command TEXT,
    script TEXT,
    parameters JSONB DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed, cancelled, timeout
    priority INTEGER DEFAULT 1,
    timeout INTEGER DEFAULT 300,
    retries INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    result JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_tasks_job_id ON tasks(job_id);
CREATE INDEX idx_tasks_node_id ON tasks(node_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_created_at ON tasks(created_at DESC);
CREATE INDEX idx_tasks_priority ON tasks(priority DESC);

-- Firmware table - firmware versions
CREATE TABLE firmware (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vendor VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    component VARCHAR(50) NOT NULL, -- BIOS, BMC, NIC, RAID, etc.
    version VARCHAR(50) NOT NULL,
    release_date DATE,
    file_path VARCHAR(500),
    checksum VARCHAR(128),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(vendor, model, component, version)
);

CREATE INDEX idx_firmware_vendor ON firmware(vendor);
CREATE INDEX idx_firmware_component ON firmware(component);

-- Node firmware table - firmware installed on nodes
CREATE TABLE node_firmware (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    component VARCHAR(50) NOT NULL,
    version VARCHAR(50) NOT NULL,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(node_id, component)
);

CREATE INDEX idx_node_firmware_node_id ON node_firmware(node_id);
CREATE INDEX idx_node_firmware_component ON node_firmware(component);

-- OS images table
CREATE TABLE os_images (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    os_type VARCHAR(50) NOT NULL,
    os_version VARCHAR(100) NOT NULL,
    architecture VARCHAR(50) NOT NULL,
    image_type VARCHAR(20) NOT NULL, -- iso, qcow2, vmdk
    file_path VARCHAR(500) NOT NULL,
    size BIGINT NOT NULL,
    checksum VARCHAR(128) NOT NULL,
    kickstart_template TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_os_images_os_type ON os_images(os_type);
CREATE INDEX idx_os_images_architecture ON os_images(architecture);

-- Audit logs table (partitioned by month)
CREATE TABLE audit_logs (
    id UUID DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255),
    changes JSONB,
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- Create initial partition for current month
CREATE TABLE audit_logs_2026_04 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL, -- admin, operator, viewer
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login TIMESTAMPTZ
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);

-- Agent certificates table
CREATE TABLE agent_certificates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    serial_number VARCHAR(255) NOT NULL UNIQUE,
    certificate_pem TEXT NOT NULL,
    private_key_pem TEXT,
    not_before TIMESTAMPTZ NOT NULL,
    not_after TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, revoked, expired
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    UNIQUE(node_id)
);

CREATE INDEX idx_agent_certificates_node_id ON agent_certificates(node_id);
CREATE INDEX idx_agent_certificates_serial ON agent_certificates(serial_number);
CREATE INDEX idx_agent_certificates_not_after ON agent_certificates(not_after);

-- Functions
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers
CREATE TRIGGER update_nodes_updated_at BEFORE UPDATE ON nodes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ssh_configs_updated_at BEFORE UPDATE ON ssh_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bmc_configs_updated_at BEFORE UPDATE ON bmc_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create default admin user (password: changeme - bcrypt hash)
INSERT INTO users (username, email, password_hash, role)
VALUES ('admin', 'admin@reignx.local', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'admin');

-- Create view for node details with configurations
CREATE VIEW node_details AS
SELECT
    n.*,
    sc.host as ssh_host,
    sc.port as ssh_port,
    sc.user as ssh_user,
    bc.address as bmc_address,
    bc.type as bmc_type
FROM nodes n
LEFT JOIN ssh_configs sc ON n.id = sc.node_id
LEFT JOIN bmc_configs bc ON n.id = bc.node_id;
