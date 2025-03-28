package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/zechtz/nyatictl/cli"
	"github.com/zechtz/nyatictl/config"
	"github.com/zechtz/nyatictl/logger"
)

// Server manages the web UI backend.
type Server struct {
	configs     []ConfigEntry
	configsLock sync.Mutex
	logChannels map[string]chan string // Map of session ID to log channel
	logLock     sync.Mutex
	upgrader    websocket.Upgrader
}

// NewServer initializes a new web server.
func NewServer() (*Server, error) {
	configs, err := LoadConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to load configs: %v", err)
	}

	return &Server{
		configs:     configs,
		logChannels: make(map[string]chan string),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins (adjust for production)
			},
		},
	}, nil
}

// Start runs the web server.
func (s *Server) Start(port string) error {
	// Start a goroutine to redirect global logs to session-specific channels
	go func() {
		for msg := range logger.LogChan {
			s.logLock.Lock()
			for _, ch := range s.logChannels {
				select {
				case ch <- msg:
				default:
					// Drop message if channel is full
				}
			}
			s.logLock.Unlock()
		}
	}()

	r := mux.NewRouter()

	// API endpoints
	r.HandleFunc("/api/configs", s.handleGetConfigs).Methods("GET")
	r.HandleFunc("/api/configs", s.handleSaveConfigs).Methods("POST")
	r.HandleFunc("/api/config-details", s.handleConfigDetails).Methods("GET") // New endpoint
	r.HandleFunc("/api/deploy", s.handleDeploy).Methods("POST")
	r.HandleFunc("/api/task", s.handleExecuteTask).Methods("POST")

	// WebSocket endpoint for logs
	r.HandleFunc("/ws/logs/{sessionID}", s.handleLogsWebSocket)

	// Serve the React frontend (assumes build files are in ./web/build)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/build")))

	log.Printf("Starting web server on :%s", port)
	return http.ListenAndServe(":"+port, r)
}

// handleGetConfigs returns the list of configurations.
func (s *Server) handleGetConfigs(w http.ResponseWriter, r *http.Request) {
	s.configsLock.Lock()
	defer s.configsLock.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.configs)
}

// handleSaveConfigs saves a new or updated configuration.
func (s *Server) handleSaveConfigs(w http.ResponseWriter, r *http.Request) {
	var entry ConfigEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.configsLock.Lock()
	defer s.configsLock.Unlock()

	// Check if the config path already exists
	updated := false
	for i, cfg := range s.configs {
		if cfg.Path == entry.Path {
			s.configs[i] = entry
			updated = true
			break
		}
	}
	if !updated {
		s.configs = append(s.configs, entry)
	}

	// Save to file
	if err := SaveConfigs(s.configs); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save configs: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleConfigDetails returns the tasks and hosts for a given config file.
func (s *Server) handleConfigDetails(w http.ResponseWriter, r *http.Request) {
	configPath := r.URL.Query().Get("path")
	if configPath == "" {
		http.Error(w, "Missing 'path' query parameter", http.StatusBadRequest)
		return
	}

	// Load the config file
	cfg, err := config.Load(configPath, "0.1.2")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	// Extract tasks
	tasks := make([]string, 0, len(cfg.Tasks))
	for _, task := range cfg.Tasks {
		tasks = append(tasks, task.Name)
	}

	// Extract hosts
	hosts := make([]string, 0, len(cfg.Hosts)+1)
	hosts = append(hosts, "all") // Add "all" as an option
	for hostName := range cfg.Hosts {
		hosts = append(hosts, hostName)
	}

	// Return the response
	response := struct {
		Tasks []string `json:"tasks"`
		Hosts []string `json:"hosts"`
	}{
		Tasks: tasks,
		Hosts: hosts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDeploy executes a deployment for the specified config.
func (s *Server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConfigPath string `json:"configPath"`
		Host       string `json:"host"` // e.g., "all" or "server1"
		SessionID  string `json:"sessionID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a log channel for this session
	logChan := make(chan string, 100)
	s.logLock.Lock()
	s.logChannels[req.SessionID] = logChan
	s.logLock.Unlock()

	go func() {
		defer func() {
			s.logLock.Lock()
			delete(s.logChannels, req.SessionID)
			close(logChan)
			s.logLock.Unlock()
		}()

		// Load the config
		cfg, err := config.Load(req.ConfigPath, "0.1.2")
		if err != nil {
			logger.Log(fmt.Sprintf("Error: %v", err))
			return
		}

		// Run the deployment
		args := []string{"deploy", req.Host}
		err = cli.Run(cfg, args, "", false, true) // taskName="", includeLib=false, debug=true
		if err != nil {
			logger.Log(fmt.Sprintf("Error: %v", err))
		}
	}()

	w.WriteHeader(http.StatusOK)
}

// handleExecuteTask executes a specific task for the specified config.
func (s *Server) handleExecuteTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConfigPath string `json:"configPath"`
		Host       string `json:"host"`
		TaskName   string `json:"taskName"`
		SessionID  string `json:"sessionID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a log channel for this session
	logChan := make(chan string, 100)
	s.logLock.Lock()
	s.logChannels[req.SessionID] = logChan
	s.logLock.Unlock()

	go func() {
		defer func() {
			s.logLock.Lock()
			delete(s.logChannels, req.SessionID)
			close(logChan)
			s.logLock.Unlock()
		}()

		// Load the config
		cfg, err := config.Load(req.ConfigPath, "0.1.2")
		if err != nil {
			logger.Log(fmt.Sprintf("Error: %v", err))
			return
		}

		// Run the specific task
		args := []string{"deploy", req.Host}
		err = cli.Run(cfg, args, req.TaskName, false, true) // includeLib=false, debug=true
		if err != nil {
			logger.Log(fmt.Sprintf("Error: %v", err))
		}
	}()

	w.WriteHeader(http.StatusOK)
}

// handleLogsWebSocket streams logs to the client via WebSocket.
func (s *Server) handleLogsWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Wait for the log channel to be created
	var logChan chan string
	for {
		s.logLock.Lock()
		if ch, exists := s.logChannels[sessionID]; exists {
			logChan = ch
			s.logLock.Unlock()
			break
		}
		s.logLock.Unlock()
	}

	// Stream logs to the client
	for logMsg := range logChan {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(logMsg)); err != nil {
			log.Printf("WebSocket write failed: %v", err)
			return
		}
	}
}
