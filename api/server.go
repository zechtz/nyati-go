package api

import (
	"database/sql"
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
	"github.com/zechtz/nyatictl/web"
)

// Embed the web/build directory
// Note: This assumes the web/build directory is at the same level as your Go module root
// You may need to adjust the path based on your project structure

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
	// Ensure all migrations are applied before initializing the server
	if err := EnsureDatabaseMigrated(); err != nil {
		return nil, fmt.Errorf("migration check failed: %v", err)
	}

	// Initialize SQLite database connection
	db, err := sql.Open("sqlite3", "./nyatictl.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Database schema is managed through migrations
	// Tables are created via the migration system in EnsureDatabaseMigrated()

	// Check if any users exist, if not, this is the initial setup
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Failed to close database after query error: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to check user count: %v", err)
	}

	// Only create initial setup if no users exist
	if userCount == 0 {
		log.Println("No users found. Initial setup required.")
		log.Println("Please create an admin user by registering through the web interface.")
		log.Println("The first user to register will have admin privileges.")
	}

	// Load all configs from the database initially (for server startup)
	// We don't specify a user_id here because we want all configs
	configs, err := LoadConfigs(db)
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Failed to close database after config load error: %v", closeErr)
		}
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
	// Note: Database connection is intentionally NOT closed here since the server
	// needs it throughout its lifetime. The connection will be closed when the 
	// server instance is garbage collected or explicitly closed by calling Close().

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

	// --- Serve embedded frontend ---
	uiFS, err := fs.Sub(web.EmbeddedUI, "dist")
	if err != nil {
		return fmt.Errorf("failed to access embedded UI: %v", err)
	}

	// Add CORS middleware
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
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

	api.HandleFunc("/deploy", s.handleDeploy).Methods("POST")
	api.HandleFunc("/task", s.handleExecuteTask).Methods("POST")
	api.HandleFunc("/refresh-token", s.HandleRefreshToken).Methods("POST")

	// Register the ConfigRoutes routes to the protected API subrouter
	s.RegisterConfigRoutes(api)

	// Register the RegisterBlueprint routes to the protected API subrouter
	s.RegisterBlueprintRoutes(api)

	// Register the RegisterBlueprint routes to the protected API subrouter
	s.RegisterWebhookRoutes(api)

	// Register the sandbox routes to the protected API subrouter
	s.RegisterSandboxRoutes(api)

	// Register the env routes to the protected API subrouter
	s.InitEnvRoutes(api)

	// WebSocket endpoint for real-time logs
	r.HandleFunc("/ws/logs/{sessionID}", s.handleLogsWebSocket)

	// --- EMBEDDED STATIC UI ---

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

// Close gracefully shuts down the server and closes database connections
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// handleGetConfigs returns all saved configuration entries as JSON.
func (s *Server) handleGetConfigs(w http.ResponseWriter, r *http.Request) {
	// get  user id from context
	claims, ok := GetUserFromContext(r)

	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	s.configsLock.Lock()
	defer s.configsLock.Unlock()

	// Reload configs from the database to ensure freshness
	configs, err := LoadConfigs(s.db, claims.UserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load configs: %v", err), http.StatusInternalServerError)
		return
	}

	// Log the config entries
	// for _, cfg := range configs {
	// 	log.Printf("Config Entry: %s, Path: %s, Status: %s", cfg.Name, cfg.Path, cfg.Status)
	// }

	s.configs = configs

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.configs)
}

// handleSaveConfigs accepts a new or updated config entry and persists it to disk.
func (s *Server) handleSaveConfigs(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the JWT claims in context
	claims, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var entry ConfigEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		log.Printf("JSON decode error: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Set the user ID for the config
	entry.UserID = claims.UserID

	s.configsLock.Lock()
	defer s.configsLock.Unlock()

	// Update existing config or append new one
	updated := false
	for i, cfg := range s.configs {
		if cfg.Path == entry.Path {
			// Only allow updates if the user owns the config
			if cfg.UserID != claims.UserID {
				http.Error(w, "You don't have permission to modify this config", http.StatusForbidden)
				return
			}
			s.configs[i] = entry
			updated = true
			break
		}
	}

	if !updated {
		s.configs = append(s.configs, entry)
	}

	// Save the config to the database
	if err := SaveConfig(s.db, entry); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Config saved successfully"})
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
	// Get user ID from the JWT claims in context
	claims, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ConfigPath string `json:"configPath"`
		Host       string `json:"host"`
		SessionID  string `json:"sessionID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if the user owns this config
	var userID int
	err := s.db.QueryRow("SELECT user_id FROM configs WHERE path = ?", req.ConfigPath).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Config not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Verify ownership
	if userID != claims.UserID {
		http.Error(w, "You don't have permission to deploy this config", http.StatusForbidden)
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

				// Save the updated status to the database
				if err := SaveConfig(s.db, s.configs[i]); err != nil {
					logger.Log(fmt.Sprintf("Failed to update config status: %v", err))
				}
				break
			}
		}
		s.configsLock.Unlock()
	}()

	w.WriteHeader(http.StatusOK)
}

// handleExecuteTask runs a single task for a host using CLI execution.
func (s *Server) handleExecuteTask(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the JWT claims in context
	claims, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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

	// Check if the user owns this config
	var userID int
	err := s.db.QueryRow("SELECT user_id FROM configs WHERE path = ?", req.ConfigPath).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Config not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Verify ownership
	if userID != claims.UserID {
		http.Error(w, "You don't have permission to execute tasks on this config", http.StatusForbidden)
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

			// Trigger webhooks for task failure
			payload := WebhookPayload{
				Event:      "task",
				Action:     "execute",
				Status:     "error",
				Timestamp:  time.Now(),
				ConfigPath: req.ConfigPath,
				TaskName:   req.TaskName,
				Host:       req.Host,
				UserID:     userID,
				Data: map[string]any{
					"error": err.Error(),
				},
			}
			TriggerWebhooks(s.db, "task", payload)
			return
		}
		args := []string{"deploy", req.Host}
		if err := cli.Run(cfg, args, req.TaskName, false, true); err != nil {
			logger.Log(fmt.Sprintf("Error: %v", err))

			// Trigger webhooks for task failure
			payload := WebhookPayload{
				Event:      "task",
				Action:     "execute",
				Status:     "error",
				Timestamp:  time.Now(),
				ConfigPath: req.ConfigPath,
				TaskName:   req.TaskName,
				Host:       req.Host,
				UserID:     userID,
				Data: map[string]any{
					"error": err.Error(),
				},
			}
			TriggerWebhooks(s.db, "task", payload)
		} else {
			// Trigger webhooks for task success
			payload := WebhookPayload{
				Event:      "task",
				Action:     "execute",
				Status:     "success",
				Timestamp:  time.Now(),
				ConfigPath: req.ConfigPath,
				TaskName:   req.TaskName,
				Host:       req.Host,
				UserID:     userID,
				Data: map[string]any{
					"config_name": getConfigName(s.configs, req.ConfigPath),
				},
			}
			TriggerWebhooks(s.db, "task", payload)
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
