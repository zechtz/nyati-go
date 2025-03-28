package web

import (
	"encoding/json"
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

// LoadConfigs reads the config file from ConfigFilePath and unmarshals its contents
// into a slice of ConfigEntry structs.
//
// Returns an empty slice if the file does not exist, or an error if read/parsing fails.
func LoadConfigs() ([]ConfigEntry, error) {
	data, err := os.ReadFile(ConfigFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ConfigEntry{}, nil // Gracefully return empty if file doesn't exist
		}
		return nil, err
	}

	var configs []ConfigEntry
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, err
	}

	return configs, nil
}

// SaveConfigs marshals the provided list of configuration entries and
// writes them to the file defined in ConfigFilePath.
//
// Overwrites the file if it already exists.
func SaveConfigs(configs []ConfigEntry) error {
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFilePath, data, 0644)
}
