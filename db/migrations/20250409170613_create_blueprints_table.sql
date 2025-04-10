-- UP
-- Write your SQL statements to apply the migration here
-- For example:
-- ALTER TABLE users ADD COLUMN email TEXT;
-- CREATE INDEX idx_users_email ON users(email);

CREATE TABLE blueprints (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    version TEXT NOT NULL,
    tasks JSON NOT NULL,        -- Stored as JSON string
    parameters JSON NOT NULL,   -- Stored as JSON string
    created_by INTEGER NOT NULL,
    is_public BOOLEAN NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    FOREIGN KEY (created_by) REFERENCES users(id)
);


-- DOWN
-- Write your SQL statements to revert the migration here
-- These statements will be executed when rolling back the migration
-- For example:
-- DROP INDEX IF EXISTS idx_users_email;
-- ALTER TABLE users DROP COLUMN email;
