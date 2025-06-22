-- UP
CREATE TABLE IF NOT EXISTS configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    description TEXT,
    path TEXT UNIQUE,
    status TEXT,
    user_id INTEGER DEFAULT 1,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create index on user_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_configs_user_id ON configs(user_id);

-- Create unique index on path
CREATE UNIQUE INDEX IF NOT EXISTS idx_configs_path ON configs(path);

-- DOWN
DROP INDEX IF EXISTS idx_configs_path;
DROP INDEX IF EXISTS idx_configs_user_id;
DROP TABLE IF EXISTS configs;
