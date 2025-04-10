package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zechtz/nyatictl/api/response"
	"github.com/zechtz/nyatictl/env"
)

// Environment API endpoints for web UI
// These endpoints enable the web UI to manage environments and variables

// InitEnvRoutes sets up the environment-related API routes
func (s *Server) InitEnvRoutes(r *mux.Router) {
	// Register environment management endpoints
	// Environment management endpoints

	api := r.PathPrefix("/env").Subrouter()

	api.HandleFunc("/list", s.handleListEnvironments).Methods("GET")
	api.HandleFunc("/current", s.handleGetCurrentEnvironment).Methods("GET")
	api.HandleFunc("/switch/{name}", s.handleSwitchEnvironment).Methods("POST")
	api.HandleFunc("/create", s.handleCreateEnvironment).Methods("POST")
	api.HandleFunc("/delete/{name}", s.handleDeleteEnvironment).Methods("DELETE")

	// Variable management endpoints
	api.HandleFunc("/vars/{env}", s.handleListVariables).Methods("GET")
	api.HandleFunc("/vars/{env}", s.handleSetVariable).Methods("POST")
	api.HandleFunc("/vars/{env}/{key}", s.handleGetVariable).Methods("GET")
	api.HandleFunc("/vars/{env}/{key}", s.handleDeleteVariable).Methods("DELETE")

	// Import/export endpoints
	api.HandleFunc("/export/{env}", s.handleExportEnvironment).Methods("POST")
	api.HandleFunc("/import/{env}", s.handleImportEnvironment).Methods("POST")
}

// EnvironmentRequest represents a request to create or modify an environment
type EnvironmentRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// VariableRequest represents a request to set a variable
type VariableRequest struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
}

// handleListEnvironments returns a list of all environments
func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	envFile, err := env.LoadEnvironmentFile("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load environments: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to a simpler structure for the API
	type EnvInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsCurrent   bool   `json:"is_current"`
		VarCount    int    `json:"var_count"`
		SecretCount int    `json:"secret_count"`
	}

	var envs []EnvInfo
	for _, e := range envFile.Environments {
		envs = append(envs, EnvInfo{
			Name:        e.Name,
			Description: e.Description,
			IsCurrent:   e.Name == envFile.CurrentEnv,
			VarCount:    len(e.Variables),
			SecretCount: len(e.Secrets),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(envs)
}

// handleGetCurrentEnvironment returns the current active environment
func (s *Server) handleGetCurrentEnvironment(w http.ResponseWriter, r *http.Request) {
	envFile, err := env.LoadEnvironmentFile("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load environments: %v", err), http.StatusInternalServerError)
		return
	}

	environment, err := env.GetCurrentEnvironment(envFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get current environment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"name":         environment.Name,
		"description":  environment.Description,
		"var_count":    len(environment.Variables),
		"secret_count": len(environment.Secrets),
	})
}

// handleSwitchEnvironment changes the current active environment
func (s *Server) handleSwitchEnvironment(w http.ResponseWriter, r *http.Request) {
	// Get the environment name from the URL
	vars := mux.Vars(r)
	name := vars["name"]

	envFile, err := env.LoadEnvironmentFile("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load environments: %v", err), http.StatusInternalServerError)
		return
	}

	// Switch to the specified environment
	if err := env.SetCurrentEnvironment(envFile, name); err != nil {
		http.Error(w, fmt.Sprintf("Failed to switch environment: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Switched to environment '%s'", name),
	})
}

// handleCreateEnvironment creates a new environment
func (s *Server) handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var req EnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Environment name is required", http.StatusBadRequest)
		return
	}

	envFile, err := env.LoadEnvironmentFile("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load environments: %v", err), http.StatusInternalServerError)
		return
	}

	// Create new environment
	newEnv := env.NewEnvironment(req.Name, req.Description)

	// Add to file
	if err := env.AddEnvironment(envFile, newEnv); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add environment: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Environment '%s' created successfully", req.Name),
	})
}

// handleDeleteEnvironment deletes an environment
func (s *Server) handleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	// Get the environment name from the URL
	vars := mux.Vars(r)
	name := vars["name"]

	envFile, err := env.LoadEnvironmentFile("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load environments: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete the environment
	if err := env.RemoveEnvironment(envFile, name); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete environment: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Environment '%s' deleted successfully", name),
	})
}

// handleListVariables returns all variables in an environment
func (s *Server) handleListVariables(w http.ResponseWriter, r *http.Request) {
	// Get the environment name from the URL
	vars := mux.Vars(r)
	name := vars["env"]
	showSecrets := r.URL.Query().Get("show_secrets") == "true"

	envFile, err := env.LoadEnvironmentFile("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load environments: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the environment
	environment, err := env.GetEnvironment(envFile, name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Environment not found: %v", err), http.StatusNotFound)
		return
	}

	// Get encryption key if needed
	if showSecrets && len(environment.Secrets) > 0 {
		key := r.Header.Get("X-Encryption-Key")
		if key == "" {
			http.Error(w, "Encryption key required to view secrets", http.StatusBadRequest)
			return
		}

		environment.SetEncryptionKey(key)
	}

	// Prepare response
	type Variable struct {
		Key      string `json:"key"`
		Value    string `json:"value"`
		IsSecret bool   `json:"is_secret"`
	}

	var variables []Variable

	// Add regular variables
	for k, v := range environment.Variables {
		variables = append(variables, Variable{
			Key:      k,
			Value:    v,
			IsSecret: false,
		})
	}

	// Add secrets
	for k := range environment.Secrets {
		v := Variable{
			Key:      k,
			IsSecret: true,
		}

		if showSecrets {
			value, _, err := environment.Get(k)
			if err != nil {
				v.Value = fmt.Sprintf("<error: %v>", err)
			} else {
				v.Value = value
			}
		} else {
			v.Value = "<encrypted>"
		}

		variables = append(variables, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(variables)
}

// handleSetVariable sets a variable in an environment
func (s *Server) handleSetVariable(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)

	// Get the environment name from the URL
	vars := mux.Vars(r)
	name := vars["env"]

	var req VariableRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rw.BadRequest("Invalid request body")
		return
	}

	if req.Key == "" {
		rw.BadRequest("Variable key is required")
		return
	}

	envFile, err := env.LoadEnvironmentFile("")
	if err != nil {
		rw.InternalServerError(err.Error())
		return
	}

	// Get the environment
	environment, err := env.GetEnvironment(envFile, name)
	if err != nil {
		rw.NotFound("Environment not found")
		return
	}

	// If it's a secret, we need an encryption key
	if req.IsSecret {
		key := r.Header.Get("X-Encryption-Key")
		if key == "" {
			rw.BadRequest("Encryption key required for secrets")
			return
		}

		environment.SetEncryptionKey(key)
	}

	// Set the variable
	if err := environment.Set(req.Key, req.Value, req.IsSecret); err != nil {
		rw.InternalServerError(err.Error())
		return
	}

	// Save changes
	if err := env.SaveEnvironmentFile(envFile, ""); err != nil {
		rw.InternalServerError(err.Error())
		return
	}

	rw.Success(req.Key)
}

// handleGetVariable gets a variable from an environment
func (s *Server) handleGetVariable(w http.ResponseWriter, r *http.Request) {
	// Get the environment and key from the URL
	vars := mux.Vars(r)
	name := vars["env"]
	key := vars["key"]

	envFile, err := env.LoadEnvironmentFile("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load environments: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the environment
	environment, err := env.GetEnvironment(envFile, name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Environment not found: %v", err), http.StatusNotFound)
		return
	}

	// Try to get the variable
	value, isSecret, err := environment.Get(key)

	// If it's a secret and we need a key
	if isSecret && err == env.ErrNoEncryptionKey {
		key := r.Header.Get("X-Encryption-Key")
		if key == "" {
			http.Error(w, "Encryption key required for secrets", http.StatusBadRequest)
			return
		}

		environment.SetEncryptionKey(key)

		// Try again with the key
		value, isSecret, err = environment.Get(key)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get variable: %v", err), http.StatusInternalServerError)
		return
	}

	if value == "" && !isSecret {
		http.Error(w, fmt.Sprintf("Variable '%s' not found", key), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"key":       key,
		"value":     value,
		"is_secret": isSecret,
	})
}

// handleDeleteVariable deletes a variable from an environment
func (s *Server) handleDeleteVariable(w http.ResponseWriter, r *http.Request) {
	// Get the environment and key from the URL
	vars := mux.Vars(r)
	name := vars["env"]
	key := vars["key"]

	envFile, err := env.LoadEnvironmentFile("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load environments: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the environment
	environment, err := env.GetEnvironment(envFile, name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Environment not found: %v", err), http.StatusNotFound)
		return
	}

	// Delete the variable
	environment.Delete(key)

	// Save changes
	if err := env.SaveEnvironmentFile(envFile, ""); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save environment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Variable '%s' deleted successfully", key),
	})
}

// handleExportEnvironment exports an environment to a .env file
func (s *Server) handleExportEnvironment(w http.ResponseWriter, r *http.Request) {
	// Get the environment name from the URL
	vars := mux.Vars(r)
	name := vars["env"]

	// Parse request for output path
	type ExportRequest struct {
		OutputPath string `json:"output_path"`
	}

	var req ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.OutputPath == "" {
		req.OutputPath = ".env"
	}

	envFile, err := env.LoadEnvironmentFile("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load environments: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the environment
	environment, err := env.GetEnvironment(envFile, name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Environment not found: %v", err), http.StatusNotFound)
		return
	}

	// If there are secrets, we need an encryption key
	if len(environment.Secrets) > 0 {
		key := r.Header.Get("X-Encryption-Key")
		if key == "" {
			http.Error(w, "Encryption key required to export secrets", http.StatusBadRequest)
			return
		}

		environment.SetEncryptionKey(key)
	}

	// Export the environment
	if err := env.ExportDotenv(environment, req.OutputPath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to export environment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Environment '%s' exported to %s", name, req.OutputPath),
	})
}

// handleImportEnvironment imports variables from a .env file into an environment
func (s *Server) handleImportEnvironment(w http.ResponseWriter, r *http.Request) {
	// Get the environment name from the URL
	vars := mux.Vars(r)
	name := vars["env"]

	// Parse request
	type ImportRequest struct {
		InputPath string `json:"input_path"`
		AsSecrets bool   `json:"as_secrets"`
	}

	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.InputPath == "" {
		req.InputPath = ".env"
	}

	envFile, err := env.LoadEnvironmentFile("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load environments: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the environment
	environment, err := env.GetEnvironment(envFile, name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Environment not found: %v", err), http.StatusNotFound)
		return
	}

	// If importing as secrets, we need an encryption key
	if req.AsSecrets {
		key := r.Header.Get("X-Encryption-Key")
		if key == "" {
			http.Error(w, "Encryption key required to import as secrets", http.StatusBadRequest)
			return
		}

		environment.SetEncryptionKey(key)
	}

	// Import the environment
	if err := env.ImportDotenv(environment, req.InputPath, req.AsSecrets); err != nil {
		http.Error(w, fmt.Sprintf("Failed to import environment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Variables from %s imported into environment '%s'", req.InputPath, name),
	})
}
