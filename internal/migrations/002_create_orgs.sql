-- Create organizations table
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Create org_members table
CREATE TABLE IF NOT EXISTS org_members (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    UNIQUE(org_id, user_id, deleted_at)
);

-- Create indexes
CREATE INDEX idx_orgs_owner ON organizations(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_orgs_deleted_at ON organizations(deleted_at);
CREATE INDEX idx_org_members_org ON org_members(org_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_org_members_user ON org_members(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_org_members_deleted_at ON org_members(deleted_at);

