package web

import (
	"encoding/json"
	"os"
)

// ConfigEntry represents a single configuration entry in the UI.
type ConfigEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

// LoadConfigs loads the list of configurations from configs.json.
func LoadConfigs() ([]ConfigEntry, error) {
	data, err := os.ReadFile("configs.json")
	if err != nil {
		if os.IsNotExist(err) {
			return []ConfigEntry{}, nil // Return empty list if file doesn't exist
		}
		return nil, err
	}

	var configs []ConfigEntry
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, err
	}
	return configs, nil
}

// SaveConfigs saves the list of configurations to configs.json.
func SaveConfigs(configs []ConfigEntry) error {
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("configs.json", data, 0644)
}
