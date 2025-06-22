package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/zechtz/nyatictl/api/response"
	"github.com/zechtz/nyatictl/env"
)

// InitEnvRoutes sets up the environment-related API routes
func (s *Server) InitEnvRoutes(r *mux.Router) {
	// Register environment management endpoints
	api := r.PathPrefix("/env").Subrouter()
	api.Use(AuthMiddleware)

	// Environment management endpoints
	api.HandleFunc("/list", s.handleListEnvironments).Methods("GET")
	api.HandleFunc("/current", s.handleGetCurrentEnvironment).Methods("GET")
	api.HandleFunc("/switch/{id}", s.handleSwitchEnvironment).Methods("POST")
	api.HandleFunc("/create", s.handleCreateEnvironment).Methods("POST")
	api.HandleFunc("/delete/{id}", s.handleDeleteEnvironment).Methods("DELETE")

	// Variable management endpoints
	api.HandleFunc("/vars/{env_id}", s.handleListVariables).Methods("GET")
	api.HandleFunc("/vars/{env_id}", s.handleSetVariable).Methods("POST")
	api.HandleFunc("/vars/{env_id}/{key}", s.handleGetVariable).Methods("GET")
	api.HandleFunc("/vars/{env_id}/{key}", s.handleDeleteVariable).Methods("DELETE")
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

// handleListEnvironments returns a list of all environments for the current user
func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)

	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		rw.Unauthorized("Unauthorized")
		return
	}

	environments, err := env.GetEnvironments(s.db.DB, claims.UserID)
	if err != nil {
		rw.InternalServerError(fmt.Sprintf("Failed to load environments: %v", err))
		return
	}

	// Convert to a simpler structure for the API
	type EnvInfo struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		IsCurrent   bool   `json:"is_current"`
		VarCount    int    `json:"var_count"`
		SecretCount int    `json:"secret_count"`
	}

	var envs []EnvInfo
	for _, e := range environments {
		envs = append(envs, EnvInfo{
			ID:          e.ID,
			Name:        e.Name,
			Description: e.Description,
			IsCurrent:   e.IsCurrent,
			VarCount:    len(e.Variables),
			SecretCount: len(e.Secrets),
		})
	}

	rw.Success(envs)
}

// handleGetCurrentEnvironment returns the current active environment
func (s *Server) handleGetCurrentEnvironment(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)
	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		rw.Unauthorized("Unauthorized")
		return
	}

	environment, err := env.GetCurrentEnvironment(s.db.DB, claims.UserID)
	if err != nil {
		rw.InternalServerError(fmt.Sprintf("Failed to get current environment: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	data := map[string]any{
		"id":           environment.ID,
		"name":         environment.Name,
		"description":  environment.Description,
		"is_current":   environment.IsCurrent,
		"var_count":    len(environment.Variables),
		"secret_count": len(environment.Secrets),
	}

	env, err := mapToEnvironment(data)
	if err != nil {
		rw.InternalServerError(err.Error())
	}

	rw.Success(env)
}

// handleSwitchEnvironment changes the current active environment
func (s *Server) handleSwitchEnvironment(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)

	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		rw.Unauthorized("Unauthorized")
		return
	}

	// Get the environment ID from the URL
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		rw.BadRequest("Invalid environment ID: ")
		return
	}

	// Switch to the specified environment
	environment, err := env.SetCurrentEnvironment(s.db.DB, id, claims.UserID)
	if err != nil {
		rw.InternalServerError(fmt.Sprintf("Failed to switch environment: %v", err))
		return
	}

	rw.Success(fmt.Sprintf("Switched to environment '%s'", environment.Name))
}

// handleCreateEnvironment creates a new environment
func (s *Server) handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)

	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		rw.Unauthorized("Unauthorized")
		return
	}

	var req EnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rw.BadRequest("Invalid request body")
		return
	}

	if req.Name == "" {
		rw.BadRequest("Environment name is required")
		return
	}

	// Create new environment
	newEnv := env.NewEnvironment(req.Name, req.Description)
	newEnv.UserID = claims.UserID

	// Save to database
	if err := env.SaveEnvironment(s.db.DB, newEnv); err != nil {
		rw.InternalServerError(fmt.Sprintf("Failed to create environment: %v", err))
		return
	}

	rw.Created(newEnv)
}

// handleDeleteEnvironment deletes an environment
func (s *Server) handleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)

	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		rw.Unauthorized("Unauthorized")
		return
	}

	// Get the environment ID from the URL
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		rw.BadRequest("Invalid environment ID")
		return
	}

	// First verify that this environment belongs to the user
	environment, err := env.GetEnvironment(s.db.DB, id)
	if err != nil {
		rw.NotFound(fmt.Sprintf("Environment not found: %v", err))
		return
	}

	if environment.UserID != claims.UserID {
		rw.Forbidden("Unauthorized access to this environment")
		return
	}

	// Cannot delete current environment
	if environment.IsCurrent {
		rw.Error(400, "Cannot delete the current active environment")
		return
	}

	// Delete the environment - TODO: Add a DeleteEnvironment function to env package
	_, err = s.db.DB.Exec("DELETE FROM environment_variables WHERE environment_id = ?", id)
	if err != nil {
		rw.InternalServerError(fmt.Sprintf("Failed to delete environment variables: %v", err))
		return
	}

	_, err = s.db.DB.Exec("DELETE FROM environments WHERE id = ?", id)
	if err != nil {
		rw.InternalServerError(fmt.Sprintf("Failed to delete environment: %v", err))
		return
	}

	rw.NoContent()
}

// handleListVariables returns all variables in an environment
func (s *Server) handleListVariables(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get the environment ID from the URL
	vars := mux.Vars(r)
	idStr := vars["env_id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid environment ID", http.StatusBadRequest)
		return
	}

	showSecrets := r.URL.Query().Get("show_secrets") == "true"

	// Get the environment
	environment, err := env.GetEnvironment(s.db.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Environment not found: %v", err), http.StatusNotFound)
		return
	}

	// Verify user has access to this environment
	if environment.UserID != claims.UserID {
		http.Error(w, "Unauthorized access to this environment", http.StatusForbidden)
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
	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get the environment ID from the URL
	vars := mux.Vars(r)
	idStr := vars["env_id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid environment ID", http.StatusBadRequest)
		return
	}

	var req VariableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Key == "" {
		http.Error(w, "Variable key is required", http.StatusBadRequest)
		return
	}

	// Get the environment
	environment, err := env.GetEnvironment(s.db.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Environment not found: %v", err), http.StatusNotFound)
		return
	}

	// Verify user has access to this environment
	if environment.UserID != claims.UserID {
		http.Error(w, "Unauthorized access to this environment", http.StatusForbidden)
		return
	}

	// If it's a secret, we need an encryption key
	if req.IsSecret {
		key := r.Header.Get("X-Encryption-Key")
		if key == "" {
			http.Error(w, "Encryption key required for secrets", http.StatusBadRequest)
			return
		}

		environment.SetEncryptionKey(key)
	}

	// Set the variable
	if err := environment.Set(req.Key, req.Value, req.IsSecret); err != nil {
		http.Error(w, fmt.Sprintf("Failed to set variable: %v", err), http.StatusInternalServerError)
		return
	}

	// Save changes
	if err := env.SaveEnvironment(s.db.DB, environment); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save environment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Variable '%s' set successfully", req.Key),
	})
}

// handleGetVariable gets a variable from an environment
func (s *Server) handleGetVariable(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get the environment ID and key from the URL
	vars := mux.Vars(r)
	idStr := vars["env_id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid environment ID", http.StatusBadRequest)
		return
	}

	key := vars["key"]

	// Get the environment
	environment, err := env.GetEnvironment(s.db.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Environment not found: %v", err), http.StatusNotFound)
		return
	}

	// Verify user has access to this environment
	if environment.UserID != claims.UserID {
		http.Error(w, "Unauthorized access to this environment", http.StatusForbidden)
		return
	}

	// Try to get the variable
	value, isSecret, err := environment.Get(key)

	// If it's a secret and we need a key
	if isSecret && err == env.ErrNoEncryptionKey {
		encKey := r.Header.Get("X-Encryption-Key")
		if encKey == "" {
			http.Error(w, "Encryption key required for secrets", http.StatusBadRequest)
			return
		}

		environment.SetEncryptionKey(encKey)

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
	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get the environment ID and key from the URL
	vars := mux.Vars(r)
	idStr := vars["env_id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid environment ID", http.StatusBadRequest)
		return
	}

	key := vars["key"]

	// Get the environment
	environment, err := env.GetEnvironment(s.db.DB, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Environment not found: %v", err), http.StatusNotFound)
		return
	}

	// Verify user has access to this environment
	if environment.UserID != claims.UserID {
		http.Error(w, "Unauthorized access to this environment", http.StatusForbidden)
		return
	}

	// Delete the variable
	environment.Delete(key)

	// Save changes
	if err := env.SaveEnvironment(s.db.DB, environment); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save environment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Variable '%s' deleted successfully", key),
	})
}

func mapToEnvironment(data map[string]any) (*env.Environment, error) {
	// Step 1: Marshal the map to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Step 2: Unmarshal JSON into the struct
	var env env.Environment
	if err := json.Unmarshal(jsonBytes, &env); err != nil {
		return nil, err
	}

	return &env, nil
}
