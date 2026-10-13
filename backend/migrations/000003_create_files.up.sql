CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    object_key VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL CHECK (size >= 0),
    mime_type VARCHAR(255) NOT NULL DEFAULT 'application/octet-stream',
    sha256_hash VARCHAR(64) NOT NULL,
    parent_folder_id UUID REFERENCES folders(id) ON DELETE SET NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_files_object_key ON files (object_key);
CREATE INDEX IF NOT EXISTS idx_files_user_id ON files (user_id);
CREATE INDEX IF NOT EXISTS idx_files_parent_folder_id ON files (parent_folder_id);
CREATE INDEX IF NOT EXISTS idx_files_is_deleted ON files (is_deleted) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_files_deleted_at ON files (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_files_user_parent_name ON files (user_id, parent_folder_id, name) WHERE is_deleted = FALSE;
