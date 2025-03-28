package web

import (
	"encoding/json"
	"os"
)

// ConfigEntry represents a single configuration object used in the UI layer.
//
// Each entry contains:
//   - Name: Human-readable name of the configuration.
//   - Description: Optional description of what this config does.
//   - Path: The local or remote path the config points to.
//
// These are typically stored and read from a JSON file (configs.json).
type ConfigEntry struct {
	Name        string `json:"name"`        // Display name of the configuration
	Description string `json:"description"` // Description of the configuration's purpose
	Path        string `json:"path"`        // File path or resource reference
}

// LoadConfigs reads the configs.json file and unmarshals its contents
// into a slice of ConfigEntry structs.
//
// If the file does not exist, it gracefully returns an empty list without error.
// If the file exists but contains malformed JSON, an error will be returned.
//
// Returns:
//   - []ConfigEntry: List of parsed configurations
//   - error: Any I/O or JSON parsing error encountered
func LoadConfigs() ([]ConfigEntry, error) {
	data, err := os.ReadFile("configs.json")
	if err != nil {
		if os.IsNotExist(err) {
			// No config file? Return an empty slice instead of erroring.
			return []ConfigEntry{}, nil
		}
		// File exists but couldn't be read (e.g., permissions issue)
		return nil, err
	}

	var configs []ConfigEntry
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, err
	}

	return configs, nil
}

// SaveConfigs marshals the provided list of configuration entries into
// pretty-printed JSON and writes it to configs.json.
//
// This overwrites any existing configs.json file.
// If the directory is not writable or marshaling fails, it returns an error.
//
// Parameters:
//   - configs: Slice of ConfigEntry structs to persist
//
// Returns:
//   - error: If the file write or JSON marshaling fails
func SaveConfigs(configs []ConfigEntry) error {
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("configs.json", data, 0644)
}
