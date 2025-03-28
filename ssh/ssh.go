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

// Manager handles multiple SSH clients.
type Manager struct {
	Clients []*Client
	Config  *config.Config
	args    []string
	debug   bool
}

// Client wraps an SSH connection.
type Client struct {
	Name   string
	Server config.Host
	config *ssh.ClientConfig
	client *ssh.Client
	env    map[string]string
}

// NewManager initializes the SSH manager.
func NewManager(cfg *config.Config, args []string, debug bool) (*Manager, error) {
	return &Manager{Config: cfg, args: args, debug: debug}, nil
}

// Open connects to the selected hosts.
func (m *Manager) Open() error {
	var selectedHosts []string
	if len(m.args) > 0 {
		if m.args[0] == "deploy" && len(m.args) > 1 {
			if m.args[1] == "all" {
				// Add all host names (keys of m.Config.Hosts) to selectedHosts
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
		msg := fmt.Sprintf("📡 Connected: %s (%s@%s)", name, host.Username, host.Host)
		logger.Log(msg)
		fmt.Println(msg)
	}
	return nil
}

// Close disconnects all clients.
func (m *Manager) Close() {
	for _, client := range m.Clients {
		client.Disconnect()
	}
}

// NewClient creates a new SSH client.
func NewClient(name string, server config.Host, debug bool) (*Client, error) {
	authMethods := []ssh.AuthMethod{}
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
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Make configurable
			Timeout:         10 * time.Second,
		},
		env: env,
	}, nil
}

// Connect establishes the SSH connection.
func (c *Client) Connect() error {
	client, err := ssh.Dial("tcp", c.Server.Host+":22", c.config)
	if err != nil {
		return err
	}
	c.client = client
	return nil
}

// Disconnect closes the SSH connection.
func (c *Client) Disconnect() {
	if c.client != nil {
		c.client.Close()
	}
}

// Exec runs a command on the remote host.
func (c *Client) Exec(task config.Task, debug bool) (int, string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return -1, "", err
	}
	defer session.Close()

	var stdout, stderr strings.Builder
	session.Stdout = &stdout
	session.Stderr = &stderr

	if task.AskPass {
		session.RequestPty("xterm", 80, 24, ssh.TerminalModes{})
	}

	// Prepend cd command if dir is specified
	cmd := task.Cmd
	if task.Dir != "" {
		cmd = fmt.Sprintf("cd %s && %s", task.Dir, task.Cmd)
	}

	if debug {
		msg := fmt.Sprintf("🎲 %s@%s: %s", c.Name, c.Server.Host, cmd)
		logger.Log(msg)
		fmt.Println(msg)
	}

	err = session.Run(cmd)
	output := stdout.String() + stderr.String()
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return exitErr.ExitStatus(), output, nil
		}
		return -1, output, err
	}
	return 0, output, nil
}
