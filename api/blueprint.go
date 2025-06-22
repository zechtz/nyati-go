package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"time"

	"github.com/zechtz/nyatictl/config"
)

// Blueprint represents a reusable deployment template
type Blueprint struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"` // e.g., "nodejs", "php", "django"
	Version     string            `json:"version"`
	Tasks       []config.Task     `json:"tasks"`
	Parameters  map[string]string `json:"parameters"` // Default parameters values
	CreatedBy   int               `json:"created_by"`
	IsPublic    bool              `json:"is_public"` // Available to all users or just the creator
	CreatedAt   string            `json:"created_at"`
}

// GetBlueprintTypes returns the list of available blueprint types
func GetBlueprintTypes() []string {
	return []string{
		"nodejs",
		"php",
		"python",
		"ruby",
		"java",
		"golang",
		"static",
		"custom",
	}
}

// SaveBlueprint saves a blueprint to the database
func SaveBlueprint(db *sql.DB, blueprint Blueprint) error {
	// Serialize tasks and parameters to JSON
	tasksJSON, err := json.Marshal(blueprint.Tasks)
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %v", err)
	}

	paramsJSON, err := json.Marshal(blueprint.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %v", err)
	}

	// Check if blueprint exists
	var exists bool
	err = db.QueryRow("SELECT 1 FROM blueprints WHERE id = ?", blueprint.ID).Scan(&exists)

	switch err {
	case nil:
		// Update existing blueprint
		_, err = db.Exec(
			`UPDATE blueprints SET 
				name = ?, 
				description = ?, 
				type = ?, 
				version = ?, 
				tasks = ?, 
				parameters = ?,
				is_public = ?
			WHERE id = ?`,
			blueprint.Name,
			blueprint.Description,
			blueprint.Type,
			blueprint.Version,
			tasksJSON,
			paramsJSON,
			blueprint.IsPublic,
		)
		if err != nil {
			return fmt.Errorf("failed to update blueprint: %v", err)
		}
	case sql.ErrNoRows:
		// Insert new blueprint
		_, err = db.Exec(
			`INSERT INTO blueprints (
				name, 
				description, 
				type, 
				version, 
				tasks, 
				parameters, 
				created_by, 
				is_public, 
				created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			blueprint.Name,
			blueprint.Description,
			blueprint.Type,
			blueprint.Version,
			tasksJSON,
			paramsJSON,
			blueprint.CreatedBy,
			blueprint.IsPublic,
			time.Now().Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to insert blueprint: %v", err)
		}
	default:
		return fmt.Errorf("failed to check blueprint existence: %v", err)
	}

	return nil
}

// GetBlueprints retrieves all blueprints visible to a user
func GetBlueprints(db *sql.DB, userID int) ([]Blueprint, error) {
	// Get public blueprints and those created by the user
	rows, err := db.Query(
		`SELECT 
			id, name, description, type, version, 
			tasks, parameters, created_by, is_public, created_at 
		FROM blueprints 
		WHERE is_public = 1 OR created_by = ?
		ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query blueprints: %v", err)
	}
	defer rows.Close()

	var blueprints []Blueprint
	for rows.Next() {
		var blueprint Blueprint
		var tasksJSON, paramsJSON []byte

		err := rows.Scan(
			&blueprint.ID,
			&blueprint.Name,
			&blueprint.Description,
			&blueprint.Type,
			&blueprint.Version,
			&tasksJSON,
			&paramsJSON,
			&blueprint.CreatedBy,
			&blueprint.IsPublic,
			&blueprint.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan blueprint: %v", err)
		}

		// Deserialize tasks and parameters from JSON
		if err := json.Unmarshal(tasksJSON, &blueprint.Tasks); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tasks: %v", err)
		}

		if err := json.Unmarshal(paramsJSON, &blueprint.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %v", err)
		}

		blueprints = append(blueprints, blueprint)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during blueprint row iteration: %v", err)
	}

	return blueprints, nil
}

// GetBlueprintByID retrieves a specific blueprint by ID
func GetBlueprintByID(db *sql.DB, id string, userID int) (*Blueprint, error) {
	var blueprint Blueprint
	var tasksJSON, paramsJSON []byte

	err := db.QueryRow(
		`SELECT 
			id, name, description, type, version, 
			tasks, parameters, created_by, is_public, created_at 
		FROM blueprints 
		WHERE id = ? AND (is_public = 1 OR created_by = ?)`,
		id, userID,
	).Scan(
		&blueprint.ID,
		&blueprint.Name,
		&blueprint.Description,
		&blueprint.Type,
		&blueprint.Version,
		&tasksJSON,
		&paramsJSON,
		&blueprint.CreatedBy,
		&blueprint.IsPublic,
		&blueprint.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("blueprint not found or not accessible")
		}
		return nil, fmt.Errorf("failed to get blueprint: %v", err)
	}

	// Deserialize tasks and parameters from JSON
	if err := json.Unmarshal(tasksJSON, &blueprint.Tasks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tasks: %v", err)
	}

	if err := json.Unmarshal(paramsJSON, &blueprint.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %v", err)
	}

	return &blueprint, nil
}

// DeleteBlueprint deletes a blueprint from the database
func DeleteBlueprint(db *sql.DB, id string, userID int) error {
	// Only allow deletion by the creator
	result, err := db.Exec(
		"DELETE FROM blueprints WHERE id = ? AND created_by = ?",
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete blueprint: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("blueprint not found or you don't have permission to delete it")
	}

	return nil
}

// GenerateConfigFromBlueprint creates a config file from a blueprint
func GenerateConfigFromBlueprint(blueprint *Blueprint, name string, params map[string]string) (*config.Config, error) {
	// Start with the parameters from the blueprint
	mergedParams := make(map[string]string)
	maps.Copy(mergedParams, blueprint.Parameters)

	// Override with the provided parameters
	maps.Copy(mergedParams, params)

	// Create a new config
	cfg := &config.Config{
		Version:        "0.1.2", // Use the current version
		AppName:        name,
		Tasks:          blueprint.Tasks,
		Params:         mergedParams,
		Hosts:          make(map[string]config.Host),
		ReleaseVersion: time.Now().UnixMilli(),
	}

	return cfg, nil
}
