package api

import (
	"database/sql"
	"fmt"
	"os"
)

// ConfigFilePath defines the path used to read/write configuration entries.
// This variable can be overridden at runtime to support custom paths or environments.
var ConfigFilePath = "configs.json"

// ConfigEntry represents a single configuration object used in the UI layer.
//
// Each entry contains:
//   - Name: Human-readable name of the configuration.
//   - Description: Optional description of what this config does.
//   - Path: The local or remote path the config points to.
type ConfigEntry struct {
	ID          int    `json:"id,omitempty"`      // Add omitempty to the id field
	Name        string `json:"name"`              // Display name of the configuration
	Description string `json:"description"`       // Description of the configuration's purpose
	Path        string `json:"path"`              // File path or resource reference
	Status      string `json:"status"`            // Status of the configuration - Note the corrected JSON tag
	UserID      int    `json:"user_id,omitempty"` // ID of the user who created this config
}

// EnsureConfigsFile checks if the file defined by ConfigFilePath exists on disk.
// If the file is missing, it creates it with a default empty JSON array ([]).
//
// This function is safe to call on every application start. If the file already exists,
// it is left untouched.
//
// Returns:
//   - error: if the file cannot be created or written
func EnsureConfigsFile() error {
	if _, err := os.Stat(ConfigFilePath); os.IsNotExist(err) {
		emptyData := []byte("[]")
		return os.WriteFile(ConfigFilePath, emptyData, 0644)
	}
	return nil
}

// LoadConfigs reads the configs from the SQLite database and returns them as a slice of ConfigEntry structs.
//
// Parameters:
//   - db: SQLite database connection
//
// If userID is > 0, it filters configs for that specific user.
// If userID is 0, it loads all configs (used during server initialization).
// Returns:
//   - []ConfigEntry: list of loaded configs
//   - error: if the database query fails
func LoadConfigs(db *sql.DB, userID ...int) ([]ConfigEntry, error) {
	var query string
	var args []any

	if len(userID) > 0 && userID[0] > 0 {
		// Load configs for specific user
		query = `SELECT id, name, description, path, status, user_id 
				FROM configs WHERE user_id = ?`
		args = []any{userID[0]}
	} else {
		// Load all configs (for server initialization)
		query = `SELECT id, name, description, path, status, user_id 
				FROM configs`
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query configs: %v", err)
	}
	defer rows.Close()

	var configs []ConfigEntry
	for rows.Next() {
		var cfg ConfigEntry
		if err := rows.Scan(&cfg.ID, &cfg.Name, &cfg.Description, &cfg.Path, &cfg.Status, &cfg.UserID); err != nil {
			return nil, fmt.Errorf("failed to scan config: %v", err)
		}
		configs = append(configs, cfg)
	}

	return configs, nil
}

// SaveConfigs saves the provided list of configuration entries to the SQLite database.
// It updates existing configs and inserts new ones based on the path field.
//
// Parameters:
//   - db: SQLite database connection
//   - configs: list of configs to save
//
// Returns:
//   - error: if the database operation fails
func SaveConfig(db *sql.DB, config ConfigEntry) error {
	// Check if the config exists
	var exists bool
	var existingUserID int
	err := db.QueryRow("SELECT 1, user_id FROM configs WHERE path = ?", config.Path).Scan(&exists, &existingUserID)

	// If config exists, update it, otherwise insert it
	switch err {
	case nil:
		// Update existing config, preserving user_id
		_, err = db.Exec(
			"UPDATE configs SET name = ?, description = ?, status = ? WHERE path = ?",
			config.Name, config.Description, config.Status, config.Path,
		)
		if err != nil {
			return fmt.Errorf("failed to update config: %v", err)
		}
	case sql.ErrNoRows:
		// Insert new config
		_, err = db.Exec(
			"INSERT INTO configs (name, description, path, status, user_id) VALUES (?, ?, ?, ?, ?)",
			config.Name, config.Description, config.Path, config.Status, config.UserID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert config: %v", err)
		}
	default:
		return fmt.Errorf("failed to check config existence: %v", err)
	}

	return nil
}

// SaveConfigs saves multiple configuration entries to the database
func SaveConfigs(db *sql.DB, configs []ConfigEntry) error {
	for _, config := range configs {
		if err := SaveConfig(db, config); err != nil {
			return err
		}
	}
	return nil
}
