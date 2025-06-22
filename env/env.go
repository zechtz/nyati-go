package env

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

var (
	// ErrNoEncryptionKey is returned when trying to encrypt without a key
	ErrNoEncryptionKey = errors.New("encryption key not set")

	// ErrInvalidFormat is returned when the environment file has invalid format
	ErrInvalidFormat = errors.New("invalid environment file format")

	// DefaultEnvFile is the default path to the environment file
	DefaultEnvFile = "nyati.env.json"
)

// Environment represents a collection of environment variables
type Environment struct {
	ID          int               `json:"id,omitempty"` // Database ID
	Name        string            `json:"name"`         // Environment name (e.g., "production", "staging")
	Description string            `json:"description"`  // Description of the environment
	Variables   map[string]string `json:"variables"`    // Plain text variables
	Secrets     map[string]string `json:"secrets"`      // Encrypted sensitive values
	mu          sync.RWMutex      // For concurrent access safety
	encryptKey  []byte            // Encryption key (not serialized)
	FilePath    string            // Path to the environment file
	UserID      int               `json:"user_id"` // User ID associated with the environment
	IsCurrent   bool              `json:"is_current"`
}

// EnvironmentFile represents the structure of the environment file
type EnvironmentFile struct {
	Environments []*Environment `json:"environments"`
	CurrentEnv   string         `json:"current_env"` // Name of the active environment
}

// VariableInfo provides information about a specific environment variable
type VariableInfo struct {
	Name        string `json:"name"`
	Value       string `json:"value,omitempty"`
	IsSecret    bool   `json:"is_secret"`
	Environment string `json:"environment"`
}

// NewEnvironment creates a new Environment
func NewEnvironment(name, description string) *Environment {
	return &Environment{
		Name:        name,
		Description: description,
		Variables:   make(map[string]string),
		Secrets:     make(map[string]string),
	}
}

// SetEncryptionKey sets the key used for encrypting and decrypting secrets
func (e *Environment) SetEncryptionKey(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Use a fixed size key by hashing or padding
	hashedKey := make([]byte, 32) // AES-256 requires 32-byte key
	copy(hashedKey, []byte(key))
	e.encryptKey = hashedKey
}

// Set adds or updates an environment variable
func (e *Environment) Set(name, value string, isSecret bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if isSecret {
		if len(e.encryptKey) == 0 {
			return ErrNoEncryptionKey
		}

		// Encrypt the value
		encrypted, err := encrypt(value, e.encryptKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt value: %v", err)
		}

		e.Secrets[name] = encrypted
	} else {
		e.Variables[name] = value
	}

	return nil
}

// Get retrieves an environment variable
func (e *Environment) Get(name string) (string, bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check regular variables first
	if value, exists := e.Variables[name]; exists {
		return value, false, nil
	}

	// Check secrets
	if encryptedValue, exists := e.Secrets[name]; exists {
		if len(e.encryptKey) == 0 {
			return "", true, ErrNoEncryptionKey
		}

		// Decrypt the value
		decrypted, err := decrypt(encryptedValue, e.encryptKey)
		if err != nil {
			return "", true, fmt.Errorf("failed to decrypt value: %v", err)
		}

		return decrypted, true, nil
	}

	return "", false, nil
}

// Delete removes an environment variable
func (e *Environment) Delete(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.Variables, name)
	delete(e.Secrets, name)
}

// AsMap returns all environment variables (including decrypted secrets) as a map
func (e *Environment) AsMap() (map[string]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string]string, len(e.Variables)+len(e.Secrets))

	// Copy regular variables
	maps.Copy(result, e.Variables)

	// Decrypt and copy secrets
	for k, encryptedValue := range e.Secrets {
		if len(e.encryptKey) == 0 {
			return nil, ErrNoEncryptionKey
		}

		decrypted, err := decrypt(encryptedValue, e.encryptKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt %s: %v", k, err)
		}

		result[k] = decrypted
	}

	return result, nil
}

// LoadEnvironmentFile loads environment file from disk
func LoadEnvironmentFile(FilePath string) (*EnvironmentFile, error) {
	if FilePath == "" {
		FilePath = DefaultEnvFile
	}

	// Create default file if it doesn't exist
	if _, err := os.Stat(FilePath); os.IsNotExist(err) {
		defaultFile := &EnvironmentFile{
			Environments: []*Environment{
				NewEnvironment("development", "Development environment"),
			},
			CurrentEnv: "development",
		}

		if err := SaveEnvironmentFile(defaultFile, FilePath); err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read environment file: %v", err)
	}

	var envFile EnvironmentFile
	if err := json.Unmarshal(data, &envFile); err != nil {
		return nil, fmt.Errorf("failed to parse environment file: %v", err)
	}

	// Set the file path to each environment
	for _, env := range envFile.Environments {
		env.FilePath = FilePath
	}

	return &envFile, nil
}

// SaveEnvironmentFile saves the environment file to disk
func SaveEnvironmentFile(envFile *EnvironmentFile, filePath string) error {
	// Handle empty file path by using the default or existing path
	if filePath == "" {
		// If environments exist, use the FilePath from the first environment
		if len(envFile.Environments) > 0 && envFile.Environments[0].FilePath != "" {
			filePath = envFile.Environments[0].FilePath
		} else {
			// Otherwise use the default path
			filePath = DefaultEnvFile
		}
	}

	data, err := json.MarshalIndent(envFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal environment file: %v", err)
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Use restricted permissions for security
	return os.WriteFile(filePath, data, 0600)
}

// GetEnvironment loads an environment from the database
func GetEnvironment(db *sql.DB, id int) (*Environment, error) {
	env := &Environment{
		Variables: make(map[string]string),
		Secrets:   make(map[string]string),
	}

	// Get environment info
	err := db.QueryRow("SELECT id, name, description, is_current, user_id FROM environments WHERE id = ?", id).
		Scan(&env.ID, &env.Name, &env.Description, &env.IsCurrent, &env.UserID)
	if err != nil {
		return nil, err
	}

	// Load variables
	rows, err := db.Query("SELECT key, value, is_secret, encrypted_value FROM environment_variables WHERE environment_id = ?", id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var key, value, encValue string
		var isSecret bool

		if err := rows.Scan(&key, &value, &isSecret, &encValue); err != nil {
			return nil, err
		}

		if isSecret {
			env.Secrets[key] = encValue
		} else {
			env.Variables[key] = value
		}
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during environment variable row iteration: %v", err)
	}

	return env, nil
}

func GetEnvironments(db *sql.DB, userID int) ([]*Environment, error) {
	// Query for all environments for this user
	rows, err := db.Query("SELECT id, name, description, is_current, user_id FROM environments WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var environments []*Environment

	for rows.Next() {
		env := &Environment{
			Variables: make(map[string]string),
			Secrets:   make(map[string]string),
		}

		if err := rows.Scan(&env.ID, &env.Name, &env.Description, &env.IsCurrent, &env.UserID); err != nil {
			return nil, err
		}

		environments = append(environments, env)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during environment row iteration: %v", err)
	}

	// Load variables for each environment
	for _, env := range environments {
		varRows, err := db.Query("SELECT key, value, is_secret, encrypted_value FROM environment_variables WHERE environment_id = ?", env.ID)
		if err != nil {
			return nil, err
		}

		for varRows.Next() {
			var key, value, encValue string
			var isSecret bool

			if err := varRows.Scan(&key, &value, &isSecret, &encValue); err != nil {
				varRows.Close()
				return nil, err
			}

			if isSecret {
				env.Secrets[key] = encValue
			} else {
				env.Variables[key] = value
			}
		}

		// Check for errors during iteration
		if err := varRows.Err(); err != nil {
			varRows.Close()
			return nil, fmt.Errorf("error during environment variable row iteration: %v", err)
		}

		varRows.Close()
	}

	return environments, nil
}

func GetActiveEnvironment(db *sql.DB, userID int) (*Environment, error) {
	env := &Environment{
		Variables: make(map[string]string),
		Secrets:   make(map[string]string),
	}

	// Get the active environment for this user
	err := db.QueryRow(`
        SELECT id, name, description, is_current, user_id 
        FROM environments 
        WHERE user_id = ? AND is_current = 1 
        LIMIT 1`, userID).
		Scan(&env.ID, &env.Name, &env.Description, &env.IsCurrent, &env.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active environment found for user %d", userID)
		}
		return nil, err
	}

	// Load variables
	rows, err := db.Query("SELECT key, value, is_secret, encrypted_value FROM environment_variables WHERE environment_id = ?", env.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value, encValue string
		var isSecret bool

		if err := rows.Scan(&key, &value, &isSecret, &encValue); err != nil {
			return nil, err
		}

		if isSecret {
			env.Secrets[key] = encValue
		} else {
			env.Variables[key] = value
		}
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during environment variable row iteration: %v", err)
	}

	return env, nil
}

func SetActiveEnvironment(db *sql.DB, id int, userID int) (*Environment, error) {
	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Defer rollback in case of error
	defer tx.Rollback()

	// First check if the environment exists and belongs to this user
	var envExists bool
	err = tx.QueryRow("SELECT 1 FROM environments WHERE id = ? AND user_id = ?", id, userID).Scan(&envExists)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("environment with ID %d not found for user %d", id, userID)
		}
		return nil, err
	}

	// Unset any currently active environment for this user
	_, err = tx.Exec("UPDATE environments SET is_current = 0 WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}

	// Set this environment as active
	_, err = tx.Exec("UPDATE environments SET is_current = 1 WHERE id = ?", id)
	if err != nil {
		return nil, err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Return the environment
	return GetEnvironment(db, id)
}

// GetCurrentEnvironment returns the current active environment for a user
func GetCurrentEnvironment(db *sql.DB, userID int) (*Environment, error) {
	return GetActiveEnvironment(db, userID)
}

// SetCurrentEnvironment sets the current active environment
func SetCurrentEnvironment(db *sql.DB, id int, userID int) (*Environment, error) {
	return SetActiveEnvironment(db, id, userID)
}

// AddEnvironment adds a new environment to the file
func AddEnvironment(envFile *EnvironmentFile, env *Environment) error {
	// Check if environment with this name already exists
	for _, e := range envFile.Environments {
		if e.Name == env.Name {
			return fmt.Errorf("environment %s already exists", env.Name)
		}
	}

	env.FilePath = envFile.Environments[0].FilePath
	envFile.Environments = append(envFile.Environments, env)

	return SaveEnvironmentFile(envFile, env.FilePath)
}

// RemoveEnvironment removes an environment from the file
func RemoveEnvironment(envFile *EnvironmentFile, name string) error {
	idx := -1
	for i, env := range envFile.Environments {
		if env.Name == name {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("environment %s not found", name)
	}

	// Can't remove the current environment
	if envFile.CurrentEnv == name {
		return fmt.Errorf("cannot remove current environment")
	}

	// Remove environment from slice
	envFile.Environments = append(envFile.Environments[:idx], envFile.Environments[idx+1:]...)

	return SaveEnvironmentFile(envFile, envFile.Environments[0].FilePath)
}

// ExportDotenv exports the current environment to a .env file
func ExportDotenv(env *Environment, outputPath string) error {
	if outputPath == "" {
		outputPath = ".env"
	}

	vars, err := env.AsMap()
	if err != nil {
		return err
	}

	// Convert to .env format
	var lines []string
	for k, v := range vars {
		lines = append(lines, fmt.Sprintf("%s=%s", k, v))
	}

	content := strings.Join(lines, "\n")
	return os.WriteFile(outputPath, []byte(content), 0600)
}

// ImportDotenv imports variables from a .env file into the environment
func ImportDotenv(env *Environment, inputPath string, isSecret bool) error {
	if inputPath == "" {
		inputPath = ".env"
	}

	// Load the .env file
	vars, err := godotenv.Read(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read .env file: %v", err)
	}

	// Add all variables to the environment
	for k, v := range vars {
		if err := env.Set(k, v, isSecret); err != nil {
			return err
		}
	}

	// Save the changes
	envFile, err := LoadEnvironmentFile(env.FilePath)
	if err != nil {
		return err
	}

	return SaveEnvironmentFile(envFile, env.FilePath)
}

// encrypt encrypts a string using AES-GCM
func encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a string using AES-GCM
func decrypt(encryptedText string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// SaveEnvironment persists an environment to the database
func SaveEnvironment(db *sql.DB, env *Environment) error {
	// Begin a transaction for atomicity
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Defer rollback in case of error - will be ignored if we commit successfully
	defer tx.Rollback()

	var result sql.Result
	// If env has an ID, update it; otherwise insert a new one
	if env.ID > 0 {
		_, err = tx.Exec(`
            UPDATE environments 
            SET name = ?, description = ?, is_current = ?, user_id = ? 
            WHERE id = ?`,
			env.Name, env.Description, env.IsCurrent, env.UserID, env.ID)
	} else {
		result, err = tx.Exec(`
            INSERT INTO environments (name, description, is_current, user_id) 
            VALUES (?, ?, ?, ?)`,
			env.Name, env.Description, env.IsCurrent, env.UserID)

		if err == nil {
			id, _ := result.LastInsertId()
			env.ID = int(id)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to save environment: %v", err)
	}

	// Save variables and secrets
	if err := saveEnvironmentVariables(tx, env); err != nil {
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// saveEnvironmentVariables is a helper function to save environment variables
func saveEnvironmentVariables(tx *sql.Tx, env *Environment) error {
	// First, delete existing variables for this environment
	if env.ID > 0 {
		_, err := tx.Exec("DELETE FROM environment_variables WHERE environment_id = ?", env.ID)
		if err != nil {
			return fmt.Errorf("failed to clear existing variables: %v", err)
		}
	}

	// Insert regular variables
	for key, value := range env.Variables {
		_, err := tx.Exec(`
            INSERT INTO environment_variables 
            (environment_id, key, value, is_secret, encrypted_value) 
            VALUES (?, ?, ?, ?, ?)`,
			env.ID, key, value, false, "")
		if err != nil {
			return fmt.Errorf("failed to insert variable %s: %v", key, err)
		}
	}

	// Insert secrets
	for key, encValue := range env.Secrets {
		_, err := tx.Exec(`
            INSERT INTO environment_variables 
            (environment_id, key, value, is_secret, encrypted_value) 
            VALUES (?, ?, ?, ?, ?)`,
			env.ID, key, "", true, encValue)
		if err != nil {
			return fmt.Errorf("failed to insert secret %s: %v", key, err)
		}
	}

	return nil
}
