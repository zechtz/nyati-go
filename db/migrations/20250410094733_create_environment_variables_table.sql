-- UP
-- Write your SQL statements to apply the migration here
-- For example:
-- ALTER TABLE users ADD COLUMN email TEXT;
-- CREATE INDEX idx_users_email ON users(email);

CREATE TABLE environment_variables (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    environment_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value TEXT,
    is_secret BOOLEAN DEFAULT 0,
    encrypted_value TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (environment_id) REFERENCES environments(id) ON DELETE CASCADE,
    UNIQUE (environment_id, key)
);

-- DOWN
-- Write your SQL statements to revert the migration here
-- These statements will be executed when rolling back the migration
-- For example:
-- DROP INDEX IF EXISTS idx_users_email;
-- ALTER TABLE users DROP COLUMN email;

DROP TABLE IF EXISTS environment_variables;
