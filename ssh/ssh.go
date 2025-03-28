package ssh

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/zechtz/nyatictl/config"
	"github.com/zechtz/nyatictl/logger"
	"golang.org/x/crypto/ssh"
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
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Replace with proper host key verification
			Timeout:         10 * time.Second,
		},
		env: env,
	}, nil
}

// Connect dials the remote host and establishes an SSH connection.
//
// Returns:
//   - error: if dialing the host fails
func (c *Client) Connect() error {
	client, err := ssh.Dial("tcp", c.Server.Host+":22", c.config)
	if err != nil {
		return err
	}
	c.client = client
	return nil
}

// Disconnect cleanly closes the SSH session.
func (c *Client) Disconnect() {
	if c.client != nil {
		c.client.Close()
	}
}

// Exec executes a command (task.Cmd) on the remote server over SSH.
//
// It optionally changes the working directory, handles password prompt (if AskPass is set),
// captures both stdout and stderr, and returns output + status.
//
// Parameters:
//   - task: Task to be executed on the remote host
//   - debug: Flag to enable printing/logging of the command
//
// Returns:
//   - int: Exit status code
//   - string: Combined stdout and stderr output
//   - error: If the session setup or command execution fails
func (c *Client) Exec(task config.Task, debug bool) (int, string, error) {
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

	// Run command
	err = session.Run(cmd)
	output := stdout.String() + stderr.String()

	if err != nil {
		// Gracefully handle remote command exit codes
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return exitErr.ExitStatus(), output, nil
		}
		return -1, output, err
	}

	return 0, output, nil
}
