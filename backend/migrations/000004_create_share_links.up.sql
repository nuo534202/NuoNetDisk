CREATE TABLE share_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resource_type VARCHAR(20) NOT NULL CHECK (resource_type IN ('file', 'folder')),
    resource_id UUID NOT NULL,
    token VARCHAR(64) NOT NULL,
    permission VARCHAR(20) NOT NULL CHECK (permission IN ('read', 'write')),
    expires_at TIMESTAMPTZ,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_share_links_token ON share_links (token);
CREATE INDEX idx_share_links_user_id ON share_links (user_id);
CREATE INDEX idx_share_links_resource ON share_links (resource_type, resource_id);
CREATE INDEX idx_share_links_active ON share_links (is_revoked, expires_at) WHERE is_revoked = FALSE;
