package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/zechtz/nyatictl/logger"
)

// HandleCreateWebhook creates a new webhook
func (s *Server) HandleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse webhook data from request
	var webhook Webhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set user ID from JWT claims
	webhook.UserID = claims.UserID

	// Validate webhook data
	if webhook.Name == "" || webhook.URL == "" || webhook.Event == "" {
		http.Error(w, "Name, URL, and event are required", http.StatusBadRequest)
		return
	}

	// Create the webhook
	id, err := CreateWebhook(s.db.DB, webhook)
	if err != nil {
		logger.Log(fmt.Sprintf("Failed to create webhook: %v", err))
		http.Error(w, "Failed to create webhook", http.StatusInternalServerError)
		return
	}

	// Return the created webhook
	webhook.ID = id
	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = time.Now()

	// Don't return the secret in the response
	webhook.Secret = ""

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(webhook)
}

// HandleGetWebhooks returns all webhooks for the authenticated user
func (s *Server) HandleGetWebhooks(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get webhooks for the user
	webhooks, err := GetWebhooks(s.db.DB, claims.UserID)
	if err != nil {
		logger.Log(fmt.Sprintf("Failed to get webhooks: %v", err))
		http.Error(w, "Failed to get webhooks", http.StatusInternalServerError)
		return
	}

	// Return the webhooks
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(webhooks)
}

// HandleGetWebhook returns a specific webhook by ID
func (s *Server) HandleGetWebhook(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse webhook ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid webhook ID", http.StatusBadRequest)
		return
	}

	// Get the webhook
	webhook, err := GetWebhook(s.db.DB, id, claims.UserID)
	if err != nil {
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}

	// Don't return the secret in the response
	webhook.Secret = ""

	// Return the webhook
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(webhook)
}

// HandleUpdateWebhook updates an existing webhook
func (s *Server) HandleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse webhook ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid webhook ID", http.StatusBadRequest)
		return
	}

	// Parse webhook data from request
	var webhookUpdate Webhook
	if err := json.NewDecoder(r.Body).Decode(&webhookUpdate); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify the webhook exists and belongs to the user
	existingWebhook, err := GetWebhook(s.db.DB, id, claims.UserID)
	if err != nil {
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}

	// Update webhook fields
	webhookUpdate.ID = existingWebhook.ID
	webhookUpdate.UserID = claims.UserID

	// If no new secret is provided, keep the existing one
	if webhookUpdate.Secret == "" {
		webhookUpdate.Secret = existingWebhook.Secret
	}

	// Validate webhook data
	if webhookUpdate.Name == "" || webhookUpdate.URL == "" || webhookUpdate.Event == "" {
		http.Error(w, "Name, URL, and event are required", http.StatusBadRequest)
		return
	}

	// Update the webhook
	err = UpdateWebhook(s.db.DB, webhookUpdate)
	if err != nil {
		logger.Log(fmt.Sprintf("Failed to update webhook: %v", err))
		http.Error(w, "Failed to update webhook", http.StatusInternalServerError)
		return
	}

	// Don't return the secret in the response
	webhookUpdate.Secret = ""
	webhookUpdate.UpdatedAt = time.Now()

	// Return the updated webhook
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(webhookUpdate)
}

// HandleDeleteWebhook deletes a webhook
func (s *Server) HandleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse webhook ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid webhook ID", http.StatusBadRequest)
		return
	}

	// Delete the webhook
	err = DeleteWebhook(s.db.DB, id, claims.UserID)
	if err != nil {
		logger.Log(fmt.Sprintf("Failed to delete webhook: %v", err))
		http.Error(w, "Failed to delete webhook", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}

// HandleIncomingWebhook processes an incoming webhook from external services
func (s *Server) HandleIncomingWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["webhookID"]

	ProcessIncomingWebhook(s.db.DB, w, r, webhookID)
}

// getConfigName retrieves the name of a config from its path
func getConfigName(configs []ConfigEntry, path string) string {
	for _, cfg := range configs {
		if cfg.Path == path {
			if cfg.Name != "" {
				return cfg.Name
			}
			return path // Return path if name is empty
		}
	}
	return path // Return path if config not found
}

func (s *Server) RegisterWebhookRoutes(r *mux.Router) {
	r.HandleFunc("/webhooks", s.HandleGetWebhooks).Methods("GET")
	r.HandleFunc("/webhooks", s.HandleCreateWebhook).Methods("POST")
	r.HandleFunc("/webhooks/{id:[0-9]+}", s.HandleGetWebhook).Methods("GET")
	r.HandleFunc("/webhooks/{id:[0-9]+}", s.HandleUpdateWebhook).Methods("PUT")
	r.HandleFunc("/webhooks/{id:[0-9]+}", s.HandleDeleteWebhook).Methods("DELETE")

	r.HandleFunc("/webhooks/incoming/{webhookID}", s.HandleIncomingWebhook).Methods("POST")
}
