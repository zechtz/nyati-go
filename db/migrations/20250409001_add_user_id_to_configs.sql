-- UP
CREATE TABLE IF NOT EXISTS configs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT,
  description TEXT,
  path TEXT UNIQUE,
  status TEXT
);

ALTER TABLE configs ADD COLUMN user_id INTEGER DEFAULT 1;

UPDATE configs SET user_id = 1 WHERE user_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_configs_user_id ON configs(user_id);

-- DOWN
CREATE TABLE configs_temp AS
SELECT id, name, description, path, status FROM configs;

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

DROP INDEX IF EXISTS idx_configs_user_id;
