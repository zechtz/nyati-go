-- UP
-- Create webhooks table
CREATE TABLE IF NOT EXISTS webhooks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    url TEXT NOT NULL,
    secret TEXT,
    event TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    active BOOLEAN NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Create an index for quick lookup by event type
CREATE INDEX idx_webhooks_event ON webhooks(event);

-- Create an index for user_id to speed up user-specific webhook queries
CREATE INDEX idx_webhooks_user_id ON webhooks(user_id);

-- DOWN
-- Remove the webhooks table and its indexes
DROP INDEX IF EXISTS idx_webhooks_user_id;
DROP INDEX IF EXISTS idx_webhooks_event;
DROP TABLE IF EXISTS webhooks;
