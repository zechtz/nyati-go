package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the top-level structure of the nyati.yaml configuration file.
// It includes metadata (like version and app name), the set of target hosts,
// the list of tasks to run, and key-value parameters used in templates.
type Config struct {
	Version        string            `mapstructure:"version"` // Version of the config file
	AppName        string            `mapstructure:"appname"` // Name of the application being deployed
	Hosts          map[string]Host   `mapstructure:"hosts"`   // Map of host identifiers to Host structs
	Tasks          []Task            `mapstructure:"tasks"`   // List of defined deployment tasks
	Params         map[string]string `mapstructure:"params"`  // Key-value parameters for template substitution
	ReleaseVersion int64             // Populated at runtime to indicate the current release timestamp
}

// Host defines connection details for a target server.
type Host struct {
	Host       string `mapstructure:"host"`                  // IP or hostname of the server
	Username   string `mapstructure:"username"`              // SSH username
	Password   string `mapstructure:"password,omitempty"`    // Optional password (used if no key is provided)
	PrivateKey string `mapstructure:"private_key,omitempty"` // Optional private key path for SSH authentication
	EnvFile    string `mapstructure:"envfile,omitempty"`     // Path to environment file to load before tasks
}

// Task defines a command to run on a host, along with its metadata and dependencies.
type Task struct {
	ID        string   `mapstructure:"id,omitempty" json:"id"`                           // Unique identifier for the task
	Name      string   `mapstructure:"name" json:"name"`                                 // Unique identifier for the task
	Cmd       string   `mapstructure:"cmd" json:"cmd"`                                   // Shell command to run
	Dir       string   `mapstructure:"dir,omitempty" json:"dir,omitempty"`               // Optional working directory for the command
	Expect    int      `mapstructure:"expect" json:"expect"`                             // Expected exit code (0 = success)
	Message   string   `mapstructure:"message,omitempty" json:"message,omitempty"`       // Optional message to display before execution
	Retry     bool     `mapstructure:"retry,omitempty" json:"retry,omitempty"`           // Whether to retry on failure
	AskPass   bool     `mapstructure:"askpass,omitempty" json:"askpass,omitempty"`       // Whether to prompt for password
	Lib       bool     `mapstructure:"lib,omitempty" json:"lib,omitempty"`               // Whether this is a library task (not run by default)
	Output    bool     `mapstructure:"output,omitempty" json:"output,omitempty"`         // Whether to display command output
	DependsOn []string `mapstructure:"depends_on,omitempty" json:"depends_on,omitempty"` // List of task names that must run before this one
}

// Load reads, parses, and validates a YAML configuration file into a Config object.
// It performs multiple checks including required fields, unique task names,
// valid dependencies, version compatibility, and circular dependency detection.
//
// Parameters:
//   - file: path to the YAML config file
//   - appVersion: expected minimum version (usually matches CLI version)
//
// Returns:
//   - *Config: populated config object
//   - error: if validation or parsing fails
func Load(file, appVersion string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(file)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config format: %v", err)
	}

	// Basic field validation
	if cfg.AppName == "" {
		return nil, fmt.Errorf("appname is required")
	}
	if len(cfg.Hosts) == 0 {
		return nil, fmt.Errorf("at least one host is required")
	}
	if len(cfg.Tasks) == 0 {
		return nil, fmt.Errorf("at least one task is required")
	}
	if !strings.HasPrefix(cfg.Version, "0.") || cfg.Version < appVersion {
		return nil, fmt.Errorf("config version %s is outdated; update to %s+", cfg.Version, appVersion)
	}

	// Validate task definitions
	taskNames := make(map[string]bool)
	for i, task := range cfg.Tasks {
		if task.Name == "" {
			return nil, fmt.Errorf("task at index %d: name is required", i)
		}
		if task.Cmd == "" {
			return nil, fmt.Errorf("task '%s': cmd is required", task.Name)
		}
		if taskNames[task.Name] {
			return nil, fmt.Errorf("duplicate task name '%s' at index %d", task.Name, i)
		}
		taskNames[task.Name] = true
	}

	// Check that all dependencies exist
	for i, task := range cfg.Tasks {
		for _, dep := range task.DependsOn {
			if !taskNames[dep] {
				return nil, fmt.Errorf("task '%s' at index %d: depends_on task '%s' does not exist", task.Name, i, dep)
			}
		}
	}

	// Check for circular references
	if err := checkCircularDependencies(cfg.Tasks); err != nil {
		return nil, err
	}

	// Set runtime timestamp for use in task substitution
	cfg.ReleaseVersion = time.Now().UnixMilli()

	// Perform placeholder substitution on command fields
	for i, task := range cfg.Tasks {
		cfg.Tasks[i].Cmd = parseLiteral(&cfg, task.Cmd)
		cfg.Tasks[i].Dir = parseLiteral(&cfg, task.Dir)
		cfg.Tasks[i].Message = parseLiteral(&cfg, task.Message)
	}

	return &cfg, nil
}

// checkCircularDependencies uses DFS to identify any circular task dependencies.
// It builds a graph of tasks and traverses it, tracking recursion depth.
//
// Parameters:
//   - tasks: list of tasks from config
//
// Returns:
//   - error: if a cycle is found, returns an error describing the cycle
func checkCircularDependencies(tasks []Task) error {
	graph := make(map[string][]string)
	for _, task := range tasks {
		graph[task.Name] = task.DependsOn
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var path []string

	var dfs func(string) error
	dfs = func(taskName string) error {
		visited[taskName] = true
		recStack[taskName] = true
		path = append(path, taskName)

		for _, dep := range graph[taskName] {
			if !visited[dep] {
				if err := dfs(dep); err != nil {
					return err
				}
			} else if recStack[dep] {
				// Cycle found: format path and return error
				cycle := append([]string{dep}, path...)
				return fmt.Errorf("circular dependency detected: %s", strings.Join(cycle, " -> "))
			}
		}

		recStack[taskName] = false
		path = path[:len(path)-1]
		return nil
	}

	for _, task := range tasks {
		if !visited[task.Name] {
			if err := dfs(task.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

// parseLiteral replaces parameter placeholders (e.g. ${param}) in a string
// with actual values from the config.Params map, as well as built-in values.
//
// Parameters:
//   - cfg: the loaded Config object
//   - input: the raw input string containing placeholders
//
// Returns:
//   - string: the input string with placeholders resolved
func parseLiteral(cfg *Config, input string) string {
	if input == "" {
		return input
	}
	output := input
	for key, value := range cfg.Params {
		output = strings.ReplaceAll(output, fmt.Sprintf("${%s}", key), value)
	}
	output = strings.ReplaceAll(output, "${appname}", cfg.AppName)
	output = strings.ReplaceAll(output, "${release_version}", fmt.Sprintf("%d", cfg.ReleaseVersion))
	return output
}

// LoadEnv reads key=value pairs from a file and loads them into a map,
// skipping empty lines and comments. Used for injecting environment variables.
//
// Parameters:
//   - envFile: the path to the .env file
//
// Returns:
//   - map[string]string: map of parsed environment variables
//   - error: if the file does not exist or cannot be read
func LoadEnv(envFile string) (map[string]string, error) {
	if envFile == "" {
		return nil, nil
	}
	absPath, err := filepath.Abs(envFile)
	if err != nil || !fileExists(absPath) {
		return nil, fmt.Errorf("env file %s not found", envFile)
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	env := make(map[string]string)
	for _, line := range strings.Split(string(content), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				env[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	return env, nil
}

// fileExists returns true if the given file path exists on disk.
//
// Parameters:
//   - path: path to the file
//
// Returns:
//   - bool: true if file exists, false otherwise
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
