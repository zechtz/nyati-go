-- UP
-- Add user_id column (if it doesn't exist)
ALTER TABLE configs ADD COLUMN user_id INTEGER DEFAULT 1;

-- Update existing records
UPDATE configs SET user_id = 1 WHERE user_id IS NULL;

-- Create an index for performance (not the same as a constraint, but helps)
CREATE INDEX idx_configs_user_id ON configs(user_id);

-- DOWN
-- For the down migration, we'd need to recreate the table without the column
-- since SQLite doesn't support DROP COLUMN directly
CREATE TABLE configs_temp AS SELECT id, name, description, path, status FROM configs;
DROP TABLE configs;
CREATE TABLE configs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT,
  description TEXT,
  path TEXT UNIQUE,
  status TEXT
);
INSERT INTO configs SELECT * FROM configs_temp;
DROP TABLE configs_temp;
