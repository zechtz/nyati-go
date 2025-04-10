-- UP
-- Write your SQL statements to apply the migration here
-- For example:
-- ALTER TABLE users ADD COLUMN email TEXT;
-- CREATE INDEX idx_users_email ON users(email);

CREATE TABLE environments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  description TEXT,
  is_current BOOLEAN DEFAULT 0,
  user_id INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);


-- DOWN
-- Write your SQL statements to revert the migration here
-- These statements will be executed when rolling back the migration
-- For example:
-- DROP INDEX IF EXISTS idx_users_email;
-- ALTER TABLE users DROP COLUMN email;

DROP TABLE IF EXISTS environments;
