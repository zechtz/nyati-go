package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/zechtz/nyatictl/cli"
	"github.com/zechtz/nyatictl/config"
	"github.com/zechtz/nyatictl/logger"
)

//go:embed build/*
var embeddedUI embed.FS

// Server represents the backend web server for the UI.
//
// It handles:
//   - WebSocket log streaming (per session)
//   - REST API endpoints for config management and task execution
//   - Serving the embedded React frontend
type Server struct {
	configs     []ConfigEntry          // In-memory list of available config entries
	configsLock sync.Mutex             // Mutex to protect access to configs
	logChannels map[string]chan string // Session ID -> log channel mapping for WebSocket streaming
	logLock     sync.Mutex             // Mutex to protect logChannels map
	upgrader    websocket.Upgrader     // WebSocket upgrader with origin check disabled
}

// NewServer creates and initializes a new Server instance.
//
// It loads any saved configs (from JSON) and sets up the WebSocket upgrader.
//
// Returns:
//   - *Server: a fully initialized web server instance
//   - error: if config loading fails
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
				return true // Allow all origins for dev; restrict in production!
			},
		},
	}, nil
}

// Start launches the HTTP server on the specified port and attaches all routes.
//
// This includes:
//   - WebSocket for real-time log streaming
//   - REST endpoints for config/task management
//   - Serving the embedded frontend (React UI build)
//
// Parameters:
//   - port: HTTP port (e.g., "8080")
//
// Returns:
//   - error: from ListenAndServe if the server fails to start
func (s *Server) Start(port string) error {
	// Background goroutine to dispatch log messages to each session's WebSocket
	go func() {
		for msg := range logger.LogChan {
			s.logLock.Lock()
			for _, ch := range s.logChannels {
				select {
				case ch <- msg:
				default:
					// Drop log message if client's channel is full
				}
			}
			s.logLock.Unlock()
		}
	}()

	r := mux.NewRouter()

	// --- API ROUTES ---

	r.HandleFunc("/api/configs", s.handleGetConfigs).Methods("GET")
	r.HandleFunc("/api/configs", s.handleSaveConfigs).Methods("POST")
	r.HandleFunc("/api/config-details", s.handleConfigDetails).Methods("GET")
	r.HandleFunc("/api/deploy", s.handleDeploy).Methods("POST")
	r.HandleFunc("/api/task", s.handleExecuteTask).Methods("POST")

	// WebSocket endpoint for real-time logs
	r.HandleFunc("/ws/logs/{sessionID}", s.handleLogsWebSocket)

	// --- EMBEDDED STATIC UI ---

	// Serve embedded frontend files from /build
	uiFS, err := fs.Sub(embeddedUI, "build")
	if err != nil {
		log.Fatalf("Failed to mount embedded UI filesystem: %v", err)
	}
	r.PathPrefix("/").Handler(http.FileServer(http.FS(uiFS)))

	log.Printf("Starting web server on :%s", port)
	return http.ListenAndServe(":"+port, r)
}

// handleGetConfigs returns all saved configuration entries as JSON.
func (s *Server) handleGetConfigs(w http.ResponseWriter, r *http.Request) {
	s.configsLock.Lock()
	defer s.configsLock.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.configs)
}

// handleSaveConfigs accepts a new or updated config entry and persists it to disk.
func (s *Server) handleSaveConfigs(w http.ResponseWriter, r *http.Request) {
	var entry ConfigEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.configsLock.Lock()
	defer s.configsLock.Unlock()

	// Update existing config or append new one
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

	if err := SaveConfigs(s.configs); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save configs: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleConfigDetails loads a specified config file and returns its task and host names.
func (s *Server) handleConfigDetails(w http.ResponseWriter, r *http.Request) {
	configPath := r.URL.Query().Get("path")
	if configPath == "" {
		http.Error(w, "Missing 'path' query parameter", http.StatusBadRequest)
		return
	}

	cfg, err := config.Load(configPath, "0.1.2")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	// Extract task and host names
	tasks := make([]string, 0, len(cfg.Tasks))
	for _, task := range cfg.Tasks {
		tasks = append(tasks, task.Name)
	}

	hosts := make([]string, 0, len(cfg.Hosts)+1)
	hosts = append(hosts, "all") // Add "all" as a deploy option
	for hostName := range cfg.Hosts {
		hosts = append(hosts, hostName)
	}

	response := struct {
		Tasks []string `json:"tasks"`
		Hosts []string `json:"hosts"`
	}{Tasks: tasks, Hosts: hosts}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDeploy triggers a deployment using the provided config and host.
func (s *Server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConfigPath string `json:"configPath"`
		Host       string `json:"host"`
		SessionID  string `json:"sessionID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a log channel scoped to this session
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

		cfg, err := config.Load(req.ConfigPath, "0.1.2")
		if err != nil {
			logger.Log(fmt.Sprintf("Error: %v", err))
			return
		}

		args := []string{"deploy", req.Host}
		if err := cli.Run(cfg, args, "", false, true); err != nil {
			logger.Log(fmt.Sprintf("Error: %v", err))
		}
	}()

	w.WriteHeader(http.StatusOK)
}

// handleExecuteTask runs a single task for a host using CLI execution.
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

		cfg, err := config.Load(req.ConfigPath, "0.1.2")
		if err != nil {
			logger.Log(fmt.Sprintf("Error: %v", err))
			return
		}

		args := []string{"deploy", req.Host}
		if err := cli.Run(cfg, args, req.TaskName, false, true); err != nil {
			logger.Log(fmt.Sprintf("Error: %v", err))
		}
	}()

	w.WriteHeader(http.StatusOK)
}

// handleLogsWebSocket upgrades the HTTP connection to a WebSocket and streams logs
// for the provided session ID in real-time.
func (s *Server) handleLogsWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	var logChan chan string
	// Wait until the log channel becomes available
	for {
		s.logLock.Lock()
		if ch, exists := s.logChannels[sessionID]; exists {
			logChan = ch
			s.logLock.Unlock()
			break
		}
		s.logLock.Unlock()
	}

	// Stream logs to WebSocket client
	for logMsg := range logChan {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(logMsg)); err != nil {
			log.Printf("WebSocket write failed: %v", err)
			return
		}
	}
}
