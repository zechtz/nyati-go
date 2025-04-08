package web

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"github.com/zechtz/nyatictl/cli"
	"github.com/zechtz/nyatictl/config"
	"github.com/zechtz/nyatictl/logger"
	"golang.org/x/crypto/bcrypt"
)

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
	db          *sql.DB                // SQLite database connection
}

// NewServer creates and initializes a new Server instance.
//
// It sets up the SQLite database, creates the necessary tables, loads any saved configs,
// and sets up the WebSocket upgrader.
//
// Returns:
//   - *Server: a fully initialized web server instance
//   - error: if database setup or config loading fails
func NewServer() (*Server, error) {
	// Initialize SQLite database connection
	db, err := sql.Open("sqlite3", "./nyatictl.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Create configs table
	createConfigsTable := `
  CREATE TABLE IF NOT EXISTS configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    description TEXT,
    path TEXT UNIQUE,
    status TEXT
    );`
	_, err = db.Exec(createConfigsTable)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create configs table: %v", err)
	}
	// Create users table
	createUsersTable := `CREATE TABLE IF NOT EXISTS users(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE,
    password hash TEXT,
    created_at TEXT
  );`

	_, err = db.Exec(createUsersTable)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create users table: %v", err)
	}

	// insert a default user for testing (in a real app, we'd hash the password)

	password := "secret"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	_, err = db.Exec(`INSERT OR IGNORE INTO users (email, password, created_at) VALUES (?, ?, ?)`, "mtabe@example.com", string(hashedPassword), time.Now().Format(time.RFC3339))
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to insert default user: %v", err)
	}
	// load configs from  the database
	configs, err := LoadConfigs(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to load configs: %v", err)
	}

	return &Server{
		configs:     configs,
		logChannels: make(map[string]chan string),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for WebSocket connections
			},
		},
		db: db,
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
	// Ensure the database connection is closed when the server shuts down
	defer s.db.Close()

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

	// Add CORS middleware
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:3000", "http://localhost:5173"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
		handlers.ExposedHeaders([]string{"Content-Type"}),
		handlers.AllowCredentials(),
	)(r)

	// --- AUTH ROUTES (not protected) ---
	r.HandleFunc("/api/login", s.HandleLogin).Methods("POST")
	r.HandleFunc("/api/logout", s.HandleLogout).Methods("POST")
	r.HandleFunc("/api/register", s.HandleRegister).Methods("POST")

	// --- Protected API Routes ---
	// Create a subrouter for protected routes
	api := r.PathPrefix("/api").Subrouter()

	// Apply the auth middleware to all routes in this subrouter
	api.Use(AuthMiddleware)

	// Add your protected routes to the api subrouter
	api.HandleFunc("/configs", s.handleGetConfigs).Methods("GET")
	api.HandleFunc("/configs", s.handleSaveConfigs).Methods("POST")
	api.HandleFunc("/config-details", s.handleConfigDetails).Methods("GET")
	api.HandleFunc("/deploy", s.handleDeploy).Methods("POST")
	api.HandleFunc("/task", s.handleExecuteTask).Methods("POST")
	api.HandleFunc("/refresh-token", s.HandleRefreshToken).Methods("POST")

	// WebSocket endpoint for real-time logs
	r.HandleFunc("/ws/logs/{sessionID}", s.handleLogsWebSocket)

	// --- EMBEDDED STATIC UI ---

	// Serve embedded frontend files from /build
	uiFS, err := fs.Sub(embeddedUI, "build")
	if err != nil {
		log.Fatalf("Failed to mount embedded UI filesystem: %v", err)
	}

	// Create a file server handler
	fileServer := http.FileServer(http.FS(uiFS))

	// Handle all other requests with the file server
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the path exists in our file system
		_, err := uiFS.Open(strings.TrimPrefix(r.URL.Path, "/"))
		if err != nil && os.IsNotExist(err) {
			// If the file doesn't exist, serve the index.html file
			// This enables client-side routing with React Router
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	log.Printf("Starting web server on :%s", port)
	return http.ListenAndServe(":"+port, corsHandler)
}

// handleGetConfigs returns all saved configuration entries as JSON.
func (s *Server) handleGetConfigs(w http.ResponseWriter, r *http.Request) {
	s.configsLock.Lock()
	defer s.configsLock.Unlock()

	// Reload configs from the database to ensure freshness
	configs, err := LoadConfigs(s.db)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load configs: %v", err), http.StatusInternalServerError)
		return
	}

	// Log the config entries
	for _, cfg := range configs {
		log.Printf("Config Entry: %s, Path: %s, Status: %s", cfg.Name, cfg.Path, cfg.Status)
	}

	s.configs = configs

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

	if err := SaveConfigs(s.db, s.configs); err != nil {
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
			return
		}

		// Update the config status to "DEPLOYED" after successful deployment
		s.configsLock.Lock()
		for i, cfg := range s.configs {
			if cfg.Path == req.ConfigPath {
				s.configs[i].Status = "DEPLOYED"
				break
			}
		}
		if err := SaveConfigs(s.db, s.configs); err != nil {
			logger.Log(fmt.Sprintf("Failed to update config status: %v", err))
		}
		s.configsLock.Unlock()
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
