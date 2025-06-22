package ssh

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zechtz/nyatictl/config"
)

func TestGetKnownHostsFile(t *testing.T) {
	knownHostsPath := getKnownHostsFile()
	
	// Should return a path ending with .ssh/known_hosts
	if knownHostsPath == "" {
		t.Error("getKnownHostsFile() returned empty path")
	}
	
	expectedSuffix := filepath.Join(".ssh", "known_hosts")
	if !strings.HasSuffix(knownHostsPath, expectedSuffix) {
		t.Errorf("getKnownHostsFile() = %v, should end with %v", knownHostsPath, expectedSuffix)
	}
}

func TestFileExists(t *testing.T) {
	// Test with existing file
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	if !fileExists(existingFile) {
		t.Error("fileExists() should return true for existing file")
	}
	
	// Test with non-existing file
	nonExistingFile := filepath.Join(tmpDir, "nonexistent.txt")
	if fileExists(nonExistingFile) {
		t.Error("fileExists() should return false for non-existing file")
	}
}

func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]config.Host{
			"testhost": {
				Host:     "example.com",
				Username: "user",
				Password: "pass",
			},
		},
	}
	
	args := []string{"deploy", "testhost"}
	debug := false
	
	manager, err := NewManager(cfg, args, debug)
	if err != nil {
		t.Errorf("NewManager() error = %v", err)
	}
	
	if manager.Config != cfg {
		t.Error("NewManager() config not set correctly")
	}
	
	if len(manager.args) != len(args) {
		t.Error("NewManager() args not set correctly")
	}
	
	if manager.debug != debug {
		t.Error("NewManager() debug not set correctly")
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		server    config.Host
		wantErr   bool
		errString string
	}{
		{
			name: "valid password auth",
			server: config.Host{
				Host:     "example.com",
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name: "missing auth method",
			server: config.Host{
				Host:     "example.com",
				Username: "user",
				// No password or private key
			},
			wantErr:   true,
			errString: "password or private_key required",
		},
		{
			name: "invalid private key path",
			server: config.Host{
				Host:       "example.com",
				Username:   "user",
				PrivateKey: "/nonexistent/path/key",
			},
			wantErr:   true,
			errString: "failed to read private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient("testclient", tt.server, false)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr {
				if tt.errString != "" && err != nil {
					if !strings.Contains(err.Error(), tt.errString) {
						t.Errorf("NewClient() error = %v, should contain %v", err, tt.errString)
					}
				}
			} else {
				if client == nil {
					t.Error("NewClient() should return non-nil client")
				}
				if client.Name != "testclient" {
					t.Errorf("NewClient() client name = %v, want testclient", client.Name)
				}
			}
		})
	}
}

func TestNewClientWithValidPrivateKey(t *testing.T) {
	// Create a temporary private key file (this is a dummy key, not a real one)
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")
	
	// This is a dummy private key content for testing
	keyContent := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAFwAAAAdzc2gtcn
NhAAAAAwEAAQAAAQEA1234567890abcdefghijklmnop
-----END OPENSSH PRIVATE KEY-----`
	
	if err := os.WriteFile(keyPath, []byte(keyContent), 0600); err != nil {
		t.Fatalf("Failed to write test key file: %v", err)
	}
	
	server := config.Host{
		Host:       "example.com",
		Username:   "user",
		PrivateKey: keyPath,
	}
	
	// This will fail because the key is invalid, but we're testing the file reading part
	_, err := NewClient("testclient", server, false)
	
	// We expect an error about invalid private key, not about file reading
	if err == nil {
		t.Error("NewClient() should fail with invalid private key")
	} else if !contains(err.Error(), "invalid private key") {
		// If it's not about invalid private key, check if it's about file reading
		if contains(err.Error(), "failed to read private key") {
			t.Error("NewClient() should read the file successfully but fail on parsing")
		}
	}
}

func TestCreateHostKeyCallback(t *testing.T) {
	callback := createHostKeyCallback()
	if callback == nil {
		t.Error("createHostKeyCallback() should return non-nil callback")
	}
	
	// We can't easily test the actual callback functionality without setting up
	// a real SSH connection, but we can at least verify it returns a function
}

func TestExecWithContextTimeout(t *testing.T) {
	// Test that ExecWithContext handles nil client gracefully
	client := &Client{
		Name: "testclient",
		// client is nil, which should cause an error
	}
	
	// Test context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	
	task := config.Task{
		Name: "test_task",
		Cmd:  "echo hello",
	}
	
	// This should fail quickly due to nil client
	code, output, err := client.ExecWithContext(ctx, task, false)
	
	// We expect an error due to nil client
	if err == nil {
		t.Error("ExecWithContext() should fail with nil client")
	}
	
	if code != -1 {
		t.Errorf("ExecWithContext() code = %v, want -1 for error", code)
	}
	
	_ = output // output might be empty, which is fine for this test
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		   (len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestManagerOpen(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]config.Host{
			"testhost": {
				Host:     "example.com",
				Username: "user",
				Password: "pass",
			},
		},
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no hosts selected",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "nonexistent host",
			args:    []string{"deploy", "nonexistent"},
			wantErr: true,
		},
		{
			name:    "deploy all",
			args:    []string{"deploy", "all"},
			wantErr: true, // Will fail on connection, but that's expected
		},
		{
			name:    "deploy specific host",
			args:    []string{"deploy", "testhost"},
			wantErr: true, // Will fail on connection, but that's expected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(cfg, tt.args, false)
			if err != nil {
				t.Fatalf("NewManager() failed: %v", err)
			}

			err = manager.Open()
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Manager.Open() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			// Clean up any connections that might have been made
			manager.Close()
		})
	}
}