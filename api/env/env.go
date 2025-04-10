package env

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"slices"
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
	Name        string            `json:"name"`        // Environment name (e.g., "production", "staging")
	Description string            `json:"description"` // Description of the environment
	Variables   map[string]string `json:"variables"`   // Plain text variables
	Secrets     map[string]string `json:"secrets"`     // Encrypted sensitive values
	mu          sync.RWMutex      // For concurrent access safety
	encryptKey  []byte            // Encryption key (not serialized)
	filePath    string            // Path to the environment file
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
	for k, v := range e.Variables {
		result[k] = v
	}

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
func LoadEnvironmentFile(filePath string) (*EnvironmentFile, error) {
	if filePath == "" {
		filePath = DefaultEnvFile
	}

	// Create default file if it doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		defaultFile := &EnvironmentFile{
			Environments: []*Environment{
				NewEnvironment("development", "Development environment"),
			},
			CurrentEnv: "development",
		}

		if err := SaveEnvironmentFile(defaultFile, filePath); err != nil {
			return nil, err
		}
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read environment file: %v", err)
	}

	var envFile EnvironmentFile
	if err := json.Unmarshal(data, &envFile); err != nil {
		return nil, fmt.Errorf("failed to parse environment file: %v", err)
	}

	// Set the file path to each environment
	for _, env := range envFile.Environments {
		env.filePath = filePath
	}

	return &envFile, nil
}

// SaveEnvironmentFile saves the environment file to disk
func SaveEnvironmentFile(envFile *EnvironmentFile, filePath string) error {
	if filePath == "" {
		filePath = DefaultEnvFile
	}

	data, err := json.MarshalIndent(envFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal environment file: %v", err)
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	return ioutil.WriteFile(filePath, data, 0600) // Use restricted permissions for security
}

// GetEnvironment returns the environment with the given name
func GetEnvironment(envFile *EnvironmentFile, name string) (*Environment, error) {
	for _, env := range envFile.Environments {
		if env.Name == name {
			return env, nil
		}
	}

	return nil, fmt.Errorf("environment %s not found", name)
}

// GetCurrentEnvironment returns the current active environment
func GetCurrentEnvironment(envFile *EnvironmentFile) (*Environment, error) {
	return GetEnvironment(envFile, envFile.CurrentEnv)
}

// SetCurrentEnvironment sets the current active environment
func SetCurrentEnvironment(envFile *EnvironmentFile, name string) error {
	_, err := GetEnvironment(envFile, name)
	if err != nil {
		return err
	}

	envFile.CurrentEnv = name
	return SaveEnvironmentFile(envFile, envFile.Environments[0].filePath)
}

// AddEnvironment adds a new environment to the file
func AddEnvironment(envFile *EnvironmentFile, env *Environment) error {
	// Check if environment with this name already exists
	for _, e := range envFile.Environments {
		if e.Name == env.Name {
			return fmt.Errorf("environment %s already exists", env.Name)
		}
	}

	env.filePath = envFile.Environments[0].filePath
	envFile.Environments = append(envFile.Environments, env)

	return SaveEnvironmentFile(envFile, env.filePath)
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
	envFile.Environments = slices.Delete(envFile.Environments, idx, idx+1)

	return SaveEnvironmentFile(envFile, envFile.Environments[0].filePath)
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
	return ioutil.WriteFile(outputPath, []byte(content), 0600)
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
	envFile, err := LoadEnvironmentFile(env.filePath)
	if err != nil {
		return err
	}

	return SaveEnvironmentFile(envFile, env.filePath)
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
