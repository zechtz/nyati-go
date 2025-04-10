package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zechtz/nyatictl/api/response"
)

// handleGetBlueprints returns all blueprints visible to the user
func (s *Server) handleGetBlueprints(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)

	// Get user ID from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		rw.Unauthorized("Unauthorized")
		return
	}

	// Get blueprints from the database
	blueprints, err := GetBlueprints(s.db, claims.UserID)
	if err != nil {
		rw.InternalServerError(err.Error())
		return
	}

	// Return blueprints as JSON
	rw.Success(blueprints)
}

// handleGetBlueprintByID returns a specific blueprint
func (s *Server) handleGetBlueprintByID(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)

	// Get user ID from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		rw.Unauthorized("Unauthorized")
		return
	}

	// Get blueprint ID from URL
	vars := mux.Vars(r)
	blueprintID := vars["id"]

	// Get blueprint from the database
	blueprint, err := GetBlueprintByID(s.db, blueprintID, claims.UserID)
	if err != nil {
		rw.NotFound(err.Error())
		return
	}

	// Return blueprint as JSON
	rw.Success(blueprint)
}

// handleSaveBlueprint creates or updates a blueprint
func (s *Server) handleSaveBlueprint(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)

	// Get user ID from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		rw.Unauthorized("Unauthorized")
		return
	}

	// Parse blueprint from request body
	var blueprint Blueprint
	if err := json.NewDecoder(r.Body).Decode(&blueprint); err != nil {
		rw.BadRequest("Invalid request body")
		return
	}

	// Set creator ID (only for new blueprints)
	if blueprint.ID == "" {
		blueprint.CreatedBy = claims.UserID
	} else {
		// Check if user is the creator of an existing blueprint
		existingBlueprint, err := GetBlueprintByID(s.db, blueprint.ID, claims.UserID)
		if err != nil {
			rw.NotFound("Blueprint not found or not accessible")
			return
		}

		if existingBlueprint.CreatedBy != claims.UserID {
			rw.Forbidden("You don't have permission to modify this blueprint")
			return
		}
	}

	// log.Printf("Unmarshaled Blueprint: %+v\n", blueprint)

	// Save blueprint to the database
	if err := SaveBlueprint(s.db, blueprint); err != nil {
		rw.InternalServerError(err.Error())
		return
	}

	// Return success response
	response := map[string]string{
		"message": "Blueprint saved successfully",
		"id":      blueprint.ID,
	}
	if blueprint.ID == "" {
		// New resource
		rw.Created(response)
	} else {
		// Updated resource
		rw.Success(response)
	}
}

// handleDeleteBlueprint deletes a blueprint
func (s *Server) handleDeleteBlueprint(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)

	// Get user ID from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		rw.Unauthorized("Unauthorized")
		return
	}

	// Get blueprint ID from URL
	vars := mux.Vars(r)
	blueprintID := vars["id"]

	// Delete blueprint from the database
	if err := DeleteBlueprint(s.db, blueprintID, claims.UserID); err != nil {
		rw.InternalServerError(err.Error())
		return
	}

	rw.NoContent()
}

// handleGenerateConfigFromBlueprint creates a new config from a blueprint
func (s *Server) handleGenerateConfigFromBlueprint(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)

	// Get user ID from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		rw.Unauthorized("Unauthorized")
		return
	}

	// Parse request from body
	var req struct {
		BlueprintID string            `json:"blueprint_id"`
		ConfigName  string            `json:"config_name"`
		Parameters  map[string]string `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rw.BadRequest("Invalid request body")
		return
	}

	// Get blueprint from the database
	blueprint, err := GetBlueprintByID(s.db, req.BlueprintID, claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		rw.NotFound(err.Error())
		return
	}

	// Generate config from blueprint
	cfg, err := GenerateConfigFromBlueprint(blueprint, req.ConfigName, req.Parameters)
	if err != nil {
		rw.InternalServerError(err.Error())
		return
	}

	// Return config as JSON
	rw.Created(cfg)
}

// handleGetBlueprintTypes returns the list of available blueprint types
func (s *Server) handleGetBlueprintTypes(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)

	// Return blueprint types as JSON
	rw.Success(GetBlueprintTypes())
}

// handleGetBlueprintPreset returns a preset blueprint for a specific type
func (s *Server) handleGetBlueprintPreset(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)
	// Get blueprint type from URL
	vars := mux.Vars(r)
	blueprintType := vars["type"]

	// Get the preset blueprint
	preset := GetDefaultBlueprintPreset(blueprintType)

	// If no preset found, return a basic blueprint
	if preset == nil {
		preset = getBasicBlueprint()
	}

	// Return preset as JSON
	rw.Success(preset)
}

// RegisterBlueprintRoutes adds blueprint-related routes to the API router
func (s *Server) RegisterBlueprintRoutes(router *mux.Router) {
	// Blueprint endpoints
	router.HandleFunc("/blueprints", s.handleGetBlueprints).Methods("GET")
	router.HandleFunc("/blueprints", s.handleSaveBlueprint).Methods("POST")
	router.HandleFunc("/blueprints/{id}", s.handleGetBlueprintByID).Methods("GET")
	router.HandleFunc("/blueprints/{id}", s.handleDeleteBlueprint).Methods("DELETE")
	router.HandleFunc("/blueprints/generate", s.handleGenerateConfigFromBlueprint).Methods("POST")
	router.HandleFunc("/blueprint-types", s.handleGetBlueprintTypes).Methods("GET")
	router.HandleFunc("/blueprints/preset/{type}", s.handleGetBlueprintPreset).Methods("GET")
}
