-- Enterprise Kiosk Platform Database Schema
-- Version: 1.0.0
-- Date: 2026-03-20
-- Description: PostgreSQL migration from SQLite for enterprise features

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- Core Tables
-- ============================================

-- Tenants table (multi-tenant support)
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    plan VARCHAR(50) NOT NULL DEFAULT 'starter',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_status ON tenants(status);

-- Security policies table (must be before devices)
CREATE TABLE security_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    settings JSONB NOT NULL DEFAULT '{}',
    app_whitelist JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_security_policies_tenant ON security_policies(tenant_id);

-- Devices table
CREATE TABLE devices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_key VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    device_info JSONB NOT NULL DEFAULT '{}',
    security_policy_id UUID REFERENCES security_policies(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ,
    UNIQUE(tenant_id, device_key)
);

CREATE INDEX idx_devices_tenant ON devices(tenant_id);
CREATE INDEX idx_devices_status ON devices(tenant_id, status);
CREATE INDEX idx_devices_last_seen ON devices(last_seen_at DESC);
CREATE INDEX idx_devices_device_key ON devices(device_key);

-- Device groups table
CREATE TABLE device_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES device_groups(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_device_groups_tenant ON device_groups(tenant_id);
CREATE INDEX idx_device_groups_parent ON device_groups(parent_id);

-- Device-group membership table
CREATE TABLE device_group_members (
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_id, group_id)
);

CREATE INDEX idx_group_members_group ON device_group_members(group_id);

-- ============================================
-- Authentication Tables
-- ============================================

-- Device certificates table
CREATE TABLE device_certificates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    certificate_pem TEXT NOT NULL,
    serial_number VARCHAR(100) UNIQUE NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoked_reason VARCHAR(100)
);

CREATE INDEX idx_device_certificates_device ON device_certificates(device_id);
CREATE INDEX idx_device_certificates_expires ON device_certificates(expires_at) WHERE revoked_at IS NULL;
CREATE INDEX idx_device_certificates_serial ON device_certificates(serial_number);

-- Refresh tokens table
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    previous_token_hash VARCHAR(64)
);

CREATE INDEX idx_refresh_tokens_device ON refresh_tokens(device_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at) WHERE revoked_at IS NULL;

-- ============================================
-- ACL Tables
-- ============================================

-- Device ACL table (EMQX access control)
CREATE TABLE device_acls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    permission VARCHAR(10) NOT NULL CHECK (permission IN ('allow', 'deny')),
    action VARCHAR(10) NOT NULL CHECK (action IN ('publish', 'subscribe', 'all')),
    topic VARCHAR(255) NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_device_acls_device ON device_acls(device_id);
CREATE INDEX idx_device_acls_topic ON device_acls(topic);

-- ============================================
-- Integrity Verification Tables
-- ============================================

-- Play Integrity check records
CREATE TABLE integrity_checks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    request_hash VARCHAR(64) NOT NULL,
    device_recognition_verdict VARCHAR(100),
    app_recognition_verdict VARCHAR(100),
    details JSONB NOT NULL DEFAULT '{}',
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_integrity_checks_device ON integrity_checks(device_id, checked_at DESC);

-- ============================================
-- Operational Tables
-- ============================================

-- Firmware versions table
CREATE TABLE firmware_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    version VARCHAR(50) NOT NULL,
    file_url TEXT NOT NULL,
    checksum_sha256 VARCHAR(64) NOT NULL,
    size_bytes BIGINT NOT NULL,
    rollout_strategy JSONB NOT NULL DEFAULT '{}',
    min_version VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::UUID), version)
);

CREATE INDEX idx_firmware_tenant ON firmware_versions(tenant_id);

-- Command history table
CREATE TABLE command_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id UUID REFERENCES devices(id) ON DELETE SET NULL,
    command_type VARCHAR(100) NOT NULL,
    command_id VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    result JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_command_history_device ON command_history(device_id, created_at DESC);
CREATE INDEX idx_command_history_tenant ON command_history(tenant_id, created_at DESC);
CREATE INDEX idx_command_history_status ON command_history(status) WHERE status = 'pending';

-- Audit logs table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_type VARCHAR(50) NOT NULL,
    actor_id VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    details JSONB NOT NULL DEFAULT '{}',
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_tenant ON audit_logs(tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(tenant_id, action);

-- ============================================
-- Functions & Triggers
-- ============================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to relevant tables
CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_security_policies_updated_at
    BEFORE UPDATE ON security_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_devices_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- Default Data
-- ============================================

-- Insert default tenant for migration
INSERT INTO tenants (id, name, slug, plan, status)
VALUES (
    '00000000-0000-0000-0000-000000000001'::UUID,
    'Default Tenant',
    'default',
    'enterprise',
    'active'
) ON CONFLICT (slug) DO NOTHING;

-- Insert default security policy
INSERT INTO security_policies (id, tenant_id, name, description)
VALUES (
    '00000000-0000-0000-0000-000000000001'::UUID,
    '00000000-0000-0000-0000-000000000001'::UUID,
    'Default Policy',
    'Default security policy for new devices'
) ON CONFLICT DO NOTHING;

-- ============================================
-- Grant Permissions (adjust as needed)
-- ============================================

-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO freekiosk;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO freekiosk;
