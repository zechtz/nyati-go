package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		version       string
		wantErr       bool
		wantAppname   string
		wantTasksLen  int
	}{
		{
			name: "valid config",
			configContent: `
version: "0.1.2"
appname: "testapp"
hosts:
  testhost:
    host: "example.com"
    username: "user"
    password: "pass"
tasks:
  - name: "test_task"
    cmd: "echo hello"
    expect: 0
`,
			version:      "0.1.2",
			wantErr:      false,
			wantAppname:  "testapp",
			wantTasksLen: 1,
		},
		{
			name: "version mismatch",
			configContent: `
version: "0.1.0"
appname: "testapp"
`,
			version: "0.1.2",
			wantErr: true,
		},
		{
			name:          "empty file",
			configContent: "",
			version:       "0.1.2",
			wantErr:       true,
		},
		{
			name: "invalid yaml",
			configContent: `
version: "0.1.2"
appname: testapp
invalid: [unclosed
`,
			version: "0.1.2",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test_config.yaml")
			
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			// Test Load function
			config, err := Load(configPath, tt.version)

			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if config.AppName != tt.wantAppname {
					t.Errorf("Load() appname = %v, want %v", config.AppName, tt.wantAppname)
				}
				if len(config.Tasks) != tt.wantTasksLen {
					t.Errorf("Load() tasks length = %v, want %v", len(config.Tasks), tt.wantTasksLen)
				}
			}
		})
	}
}

func TestParseLiteral(t *testing.T) {
	config := &Config{
		AppName: "myapp",
		Params: map[string]string{
			"env":     "production",
			"version": "1.0.0",
		},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "appname substitution",
			input:    "Deploy ${appname} to server",
			expected: "Deploy myapp to server",
		},
		{
			name:     "params substitution",
			input:    "Environment: ${env}",
			expected: "Environment: production",
		},
		{
			name:     "multiple substitutions",
			input:    "${appname} version ${version} in ${env}",
			expected: "myapp version 1.0.0 in production",
		},
		{
			name:     "release_version contains timestamp",
			input:    "Release: ${release_version}",
			expected: "Release: ", // We can't predict the exact timestamp, just check it's not empty
		},
		{
			name:     "no substitutions",
			input:    "No variables here",
			expected: "No variables here",
		},
		{
			name:     "unknown variable",
			input:    "Unknown: ${unknown}",
			expected: "Unknown: ${unknown}", // Should remain unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLiteral(config, tt.input)
			
			if tt.name == "release_version contains timestamp" {
				// Special case: check that release_version was replaced with something
				if result == tt.input || len(result) <= len("Release: ") {
					t.Errorf("parseLiteral() failed to replace release_version")
				}
			} else {
				if result != tt.expected {
					t.Errorf("parseLiteral() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestCheckCircularDependencies(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []Task
		wantErr bool
	}{
		{
			name: "no dependencies",
			tasks: []Task{
				{Name: "task1", Cmd: "echo 1"},
				{Name: "task2", Cmd: "echo 2"},
			},
			wantErr: false,
		},
		{
			name: "valid dependencies",
			tasks: []Task{
				{Name: "task1", Cmd: "echo 1"},
				{Name: "task2", Cmd: "echo 2", DependsOn: []string{"task1"}},
				{Name: "task3", Cmd: "echo 3", DependsOn: []string{"task1", "task2"}},
			},
			wantErr: false,
		},
		{
			name: "circular dependency direct",
			tasks: []Task{
				{Name: "task1", Cmd: "echo 1", DependsOn: []string{"task2"}},
				{Name: "task2", Cmd: "echo 2", DependsOn: []string{"task1"}},
			},
			wantErr: true,
		},
		{
			name: "circular dependency indirect",
			tasks: []Task{
				{Name: "task1", Cmd: "echo 1", DependsOn: []string{"task3"}},
				{Name: "task2", Cmd: "echo 2", DependsOn: []string{"task1"}},
				{Name: "task3", Cmd: "echo 3", DependsOn: []string{"task2"}},
			},
			wantErr: true,
		},
		{
			name: "no missing dependency check - checkCircularDependencies only checks for cycles",
			tasks: []Task{
				{Name: "task1", Cmd: "echo 1", DependsOn: []string{"nonexistent"}},
			},
			wantErr: false, // checkCircularDependencies doesn't validate missing dependencies
		},
		{
			name: "self dependency",
			tasks: []Task{
				{Name: "task1", Cmd: "echo 1", DependsOn: []string{"task1"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkCircularDependencies(tt.tasks)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkCircularDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadEnv(t *testing.T) {
	tests := []struct {
		name        string
		envContent  string
		wantErr     bool
		expectedLen int
	}{
		{
			name: "valid env file",
			envContent: `KEY1=value1
KEY2=value2
# This is a comment
KEY3=value with spaces`,
			wantErr:     false,
			expectedLen: 3,
		},
		{
			name:        "empty env file",
			envContent:  "",
			wantErr:     false,
			expectedLen: 0,
		},
		{
			name:        "nonexistent file",
			envContent:  "", // Will not create file
			wantErr:     true, // LoadEnv returns error for missing files
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var envPath string
			
			if tt.name != "nonexistent file" {
				tmpDir := t.TempDir()
				envPath = filepath.Join(tmpDir, ".env")
				if err := os.WriteFile(envPath, []byte(tt.envContent), 0644); err != nil {
					t.Fatalf("Failed to write test env file: %v", err)
				}
			} else {
				envPath = "/nonexistent/path/.env"
			}

			env, err := LoadEnv(envPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(env) != tt.expectedLen {
				t.Errorf("LoadEnv() env length = %v, want %v", len(env), tt.expectedLen)
			}

			// For valid env file, check specific values
			if tt.name == "valid env file" {
				if env["KEY1"] != "value1" {
					t.Errorf("LoadEnv() KEY1 = %v, want value1", env["KEY1"])
				}
				if env["KEY3"] != "value with spaces" {
					t.Errorf("LoadEnv() KEY3 = %v, want 'value with spaces'", env["KEY3"])
				}
			}
		})
	}
}