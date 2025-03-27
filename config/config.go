package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the nyati.yaml structure.
type Config struct {
	Version        string            `mapstructure:"version"`
	AppName        string            `mapstructure:"appname"`
	Hosts          map[string]Host   `mapstructure:"hosts"`
	Tasks          []Task            `mapstructure:"tasks"`
	Params         map[string]string `mapstructure:"params"`
	ReleaseVersion int64             // Set at runtime
}

type Host struct {
	Host       string `mapstructure:"host"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password,omitempty"`
	PrivateKey string `mapstructure:"private_key,omitempty"`
	EnvFile    string `mapstructure:"envfile,omitempty"`
}

type Task struct {
	Name      string   `mapstructure:"name"`
	Cmd       string   `mapstructure:"cmd"`
	Dir       string   `mapstructure:"dir,omitempty"`
	Expect    int      `mapstructure:"expect"`
	Message   string   `mapstructure:"message,omitempty"`
	Retry     bool     `mapstructure:"retry,omitempty"`
	AskPass   bool     `mapstructure:"askpass,omitempty"`
	Lib       bool     `mapstructure:"lib,omitempty"`
	Output    bool     `mapstructure:"output,omitempty"`
	DependsOn []string `mapstructure:"depends_on,omitempty"`
}

// Load reads and validates the config file.
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

	// Validation
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

	// Validate tasks
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

	// Validate depends_on
	for i, task := range cfg.Tasks {
		for _, dep := range task.DependsOn {
			if !taskNames[dep] {
				return nil, fmt.Errorf("task '%s' at index %d: depends_on task '%s' does not exist", task.Name, i, dep)
			}
		}
	}

	// Check for circular dependencies
	if err := checkCircularDependencies(cfg.Tasks); err != nil {
		return nil, err
	}

	// Set runtime values
	cfg.ReleaseVersion = time.Now().UnixMilli()

	// Parse literals in tasks
	for i, task := range cfg.Tasks {
		cfg.Tasks[i].Cmd = parseLiteral(&cfg, task.Cmd)
		cfg.Tasks[i].Dir = parseLiteral(&cfg, task.Dir)
		cfg.Tasks[i].Message = parseLiteral(&cfg, task.Message)
	}

	return &cfg, nil
}

// checkCircularDependencies detects circular dependencies using a DFS approach.
func checkCircularDependencies(tasks []Task) error {
	// Build a graph of task dependencies
	graph := make(map[string][]string)
	for _, task := range tasks {
		graph[task.Name] = task.DependsOn
	}

	// Track visited nodes and nodes in the current recursion stack
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var path []string

	var dfs func(taskName string) error
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
				// Circular dependency found
				cycle := append([]string{dep}, path...)
				return fmt.Errorf("circular dependency detected: %s", strings.Join(cycle, " -> "))
			}
		}

		recStack[taskName] = false
		path = path[:len(path)-1]
		return nil
	}

	// Run DFS for each task
	for _, task := range tasks {
		if !visited[task.Name] {
			if err := dfs(task.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

// parseLiteral replaces ${param} placeholders with config values.
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

// LoadEnv loads environment variables from a file if specified.
func LoadEnv(envFile string) (map[string]string, error) {
	if envFile == "" {
		return nil, nil
	}
	absPath, err := filepath.Abs(envFile)
	if err != nil || !fileExists(absPath) {
		return nil, fmt.Errorf("env file %s not found", envFile)
	}
	// Simplified env parsing (could use a library like godotenv)
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
