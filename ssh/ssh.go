package ssh

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zechtz/nyatictl/config"
	"github.com/zechtz/nyatictl/logger"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Manager orchestrates connections to multiple SSH clients.
// It manages which hosts to connect to based on CLI args, initializes clients,
// and provides lifecycle methods like Open() and Close().
type Manager struct {
	Clients []*Client      // List of connected SSH clients
	Config  *config.Config // Global config, loaded from nyati.yaml
	args    []string       // CLI args to determine host targeting
	debug   bool           // Whether debug mode is enabled
}

// Client represents a single SSH session to a remote host.
// It encapsulates SSH connection configuration, runtime connection,
// and environment variables loaded from an optional env file.
type Client struct {
	Name   string            // Identifier name (host alias)
	Server config.Host       // Host configuration from nyati.yaml
	config *ssh.ClientConfig // SSH configuration used to establish connection
	client *ssh.Client       // Active SSH connection
	env    map[string]string // Environment variables loaded from optional env file
}

// getKnownHostsFile returns the path to the known_hosts file
func getKnownHostsFile() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".ssh", "known_hosts")
}

// createHostKeyCallback creates a secure host key callback that validates
// against known_hosts file and prompts user for unknown hosts
func createHostKeyCallback() ssh.HostKeyCallback {
	knownHostsFile := getKnownHostsFile()
	
	// Try to load known hosts file if it exists
	var knownHostsCallback ssh.HostKeyCallback
	if knownHostsFile != "" && fileExists(knownHostsFile) {
		var err error
		knownHostsCallback, err = knownhosts.New(knownHostsFile)
		if err != nil {
			logger.Log(fmt.Sprintf("Warning: Could not load known_hosts file (%s): %v", knownHostsFile, err))
		}
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// If we have a known_hosts callback, try it first
		if knownHostsCallback != nil {
			err := knownHostsCallback(hostname, remote, key)
			if err == nil {
				return nil // Host key is already known and valid
			}
		}

		// For unknown hosts, show the key fingerprint and require explicit approval
		keyHash := sha256.Sum256(key.Marshal())
		fingerprint := hex.EncodeToString(keyHash[:])
		
		logger.Log(fmt.Sprintf("WARNING: Unknown host key for %s", hostname))
		logger.Log(fmt.Sprintf("Host key fingerprint (SHA256): %s", fingerprint))
		logger.Log(fmt.Sprintf("Key type: %s", key.Type()))
		
		// In automated mode, we should reject unknown hosts for security
		// In interactive mode, we could prompt the user
		// For now, we'll log the details and reject for security
		return fmt.Errorf("host key verification failed: unknown host %s with fingerprint %s", hostname, fingerprint)
	}
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// NewManager returns a new Manager instance, bound to config and CLI args.
//
// Parameters:
//   - cfg: Global application configuration
//   - args: CLI arguments (used to determine which host(s) to connect to)
//   - debug: Flag to enable debug output
//
// Returns:
//   - *Manager: initialized SSH manager
//   - error: if configuration is invalid (currently always nil)
func NewManager(cfg *config.Config, args []string, debug bool) (*Manager, error) {
	return &Manager{Config: cfg, args: args, debug: debug}, nil
}

// Open connects to the selected hosts defined in CLI args.
// It supports deploying to all hosts or a specific one.
// Each connection is authenticated using password or private key.
//
// Returns:
//   - error: if connection fails or hosts are not found
func (m *Manager) Open() error {
	var selectedHosts []string

	// Determine target host(s) based on CLI args
	if len(m.args) > 0 {
		if m.args[0] == "deploy" && len(m.args) > 1 {
			if m.args[1] == "all" {
				// Deploy to all configured hosts
				for hostName := range m.Config.Hosts {
					selectedHosts = append(selectedHosts, hostName)
				}
			} else if _, ok := m.Config.Hosts[m.args[1]]; ok {
				selectedHosts = append(selectedHosts, m.args[1])
			} else {
				return fmt.Errorf("host %s not found", m.args[1])
			}
		} else if _, ok := m.Config.Hosts[m.args[0]]; ok {
			selectedHosts = append(selectedHosts, m.args[0])
		}
	}

	if len(selectedHosts) == 0 {
		return fmt.Errorf("no hosts selected; use deploy <host> or <host>")
	}

	// Create SSH clients for selected hosts
	for _, name := range selectedHosts {
		host := m.Config.Hosts[name]
		client, err := NewClient(name, host, m.debug)
		if err != nil {
			return err
		}
		if err := client.Connect(); err != nil {
			return fmt.Errorf("failed to connect to %s: %v", name, err)
		}
		m.Clients = append(m.Clients, client)

		// Log connection status
		msg := fmt.Sprintf("📡 Connected: %s (%s@%s)", name, host.Username, host.Host)
		logger.Log(msg)
		fmt.Println(msg)
	}

	return nil
}

// Close disconnects all open SSH sessions managed by the Manager.
func (m *Manager) Close() {
	for _, client := range m.Clients {
		client.Disconnect()
	}
}

// NewClient creates a new SSH client for a single host using password
// or private key authentication.
//
// Parameters:
//   - name: Identifier of the host (e.g., 'server1')
//   - server: Host definition from the config
//   - debug: Whether debug output is enabled
//
// Returns:
//   - *Client: Initialized client instance
//   - error: If authentication setup or env loading fails
func NewClient(name string, server config.Host, debug bool) (*Client, error) {
	authMethods := []ssh.AuthMethod{}

	// Determine authentication method
	if server.Password != "" {
		authMethods = append(authMethods, ssh.Password(server.Password))
	} else if server.PrivateKey != "" {
		key, err := os.ReadFile(server.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %v", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		return nil, fmt.Errorf("host %s: password or private_key required", name)
	}

	// Load env file if specified
	env, err := config.LoadEnv(server.EnvFile)
	if err != nil {
		return nil, err
	}

	return &Client{
		Name:   name,
		Server: server,
		config: &ssh.ClientConfig{
			User:            server.Username,
			Auth:            authMethods,
			HostKeyCallback: createHostKeyCallback(),
			Timeout:         10 * time.Second,
		},
		env: env,
	}, nil
}

// Connect dials the remote host and establishes an SSH connection.
//
// Parameters:
//   - ctx: context for cancellation and timeout control
//
// Returns:
//   - error: if dialing the host fails or context is cancelled
func (c *Client) ConnectWithContext(ctx context.Context) error {
	// Create a dialer that respects context cancellation
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}
	
	conn, err := dialer.DialContext(ctx, "tcp", c.Server.Host+":22")
	if err != nil {
		return fmt.Errorf("failed to dial SSH host: %v", err)
	}
	
	clientConn, chans, reqs, err := ssh.NewClientConn(conn, c.Server.Host+":22", c.config)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SSH client connection: %v", err)
	}
	
	c.client = ssh.NewClient(clientConn, chans, reqs)
	return nil
}

// Connect provides backward compatibility - uses context with default timeout
func (c *Client) Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.ConnectWithContext(ctx)
}

// Disconnect cleanly closes the SSH session.
func (c *Client) Disconnect() {
	if c.client != nil {
		c.client.Close()
	}
}

// ExecWithContext executes a command (task.Cmd) on the remote server over SSH with context support.
//
// It optionally changes the working directory, handles password prompt (if AskPass is set),
// captures both stdout and stderr, and returns output + status.
//
// Parameters:
//   - ctx: context for cancellation and timeout control
//   - task: Task to be executed on the remote host
//   - debug: Flag to enable printing/logging of the command
//
// Returns:
//   - int: Exit status code
//   - string: Combined stdout and stderr output
//   - error: If the session setup or command execution fails
func (c *Client) ExecWithContext(ctx context.Context, task config.Task, debug bool) (int, string, error) {
	if c.client == nil {
		return -1, "", fmt.Errorf("SSH client not connected")
	}
	
	session, err := c.client.NewSession()
	if err != nil {
		return -1, "", err
	}
	defer session.Close()

	var stdout, stderr strings.Builder
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Enable pseudo-terminal if AskPass is set
	if task.AskPass {
		session.RequestPty("xterm", 80, 24, ssh.TerminalModes{})
	}

	// Prepend directory change if specified
	cmd := task.Cmd
	if task.Dir != "" {
		cmd = fmt.Sprintf("cd %s && %s", task.Dir, task.Cmd)
	}

	if debug {
		msg := fmt.Sprintf("🎲 %s@%s: %s", c.Name, c.Server.Host, cmd)
		logger.Log(msg)
		fmt.Println(msg)
	}

	// Create a channel to receive the result
	type result struct {
		err error
	}
	resultChan := make(chan result, 1)

	// Run command in a goroutine
	go func() {
		err := session.Run(cmd)
		resultChan <- result{err: err}
	}()

	// Wait for either command completion or context cancellation
	select {
	case res := <-resultChan:
		output := stdout.String() + stderr.String()
		
		if res.err != nil {
			// Gracefully handle remote command exit codes
			if exitErr, ok := res.err.(*ssh.ExitError); ok {
				return exitErr.ExitStatus(), output, nil
			}
			return -1, output, res.err
		}
		return 0, output, nil

	case <-ctx.Done():
		// Context was cancelled or timed out
		return -1, "", fmt.Errorf("command execution cancelled: %v", ctx.Err())
	}
}

// Exec provides backward compatibility - uses context with default timeout
func (c *Client) Exec(task config.Task, debug bool) (int, string, error) {
	// Use a reasonable default timeout for SSH commands
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return c.ExecWithContext(ctx, task, debug)
}
