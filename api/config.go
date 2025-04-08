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
	Name        string `json:"name"`        // Display name of the configuration
	Description string `json:"description"` // Description of the configuration's purpose
	Path        string `json:"path"`        // File path or resource reference
	Status      string `json:"status"`      // Status of the configuration - Note the corrected JSON tag
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
// Returns:
//   - []ConfigEntry: list of loaded configs
//   - error: if the database query fails
func LoadConfigs(db *sql.DB) ([]ConfigEntry, error) {
	rows, err := db.Query("SELECT name, description, path, status  FROM configs")
	if err != nil {
		return nil, fmt.Errorf("failed to query configs: %v", err)
	}
	defer rows.Close()

	var configs []ConfigEntry
	for rows.Next() {
		var config ConfigEntry
		err := rows.Scan(&config.Name, &config.Description, &config.Path, &config.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config: %v", err)
		}

		configs = append(configs, config)
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
func SaveConfigs(db *sql.DB, configs []ConfigEntry) error {
	// Begin a transaction to ensure atomic updates
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Prepare statements for insert and update
	insertStmt, err := tx.Prepare("INSERT INTO configs (name, description, path, status) VALUES (?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare insert statement: %v", err)
	}
	defer insertStmt.Close()

	updateStmt, err := tx.Prepare("UPDATE configs SET name = ?, description = ?, status = ? WHERE path = ?")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare update statement: %v", err)
	}
	defer updateStmt.Close()

	// For each config, check if it exists and update or insert accordingly
	for _, config := range configs {
		// Check if the config already exists
		var exists bool
		err := tx.QueryRow("SELECT 1 FROM configs WHERE path = ?", config.Path).Scan(&exists)

		switch err {
		case sql.ErrNoRows:
			// Config doesn't exist, insert it
			_, err = insertStmt.Exec(config.Name, config.Description, config.Path, config.Status)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to insert config %s: %v", config.Path, err)
			}
		case nil:
			// Config exists, update it
			_, err = updateStmt.Exec(config.Name, config.Description, config.Status, config.Path)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to update config %s: %v", config.Path, err)
			}
		default:
			// Some other database error
			tx.Rollback()
			return fmt.Errorf("database error when checking config %s: %v", config.Path, err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}
