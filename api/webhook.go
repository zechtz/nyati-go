package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zechtz/nyatictl/logger"
)

// parseTimeWithLogging safely parses a time string and returns a zero time if parsing fails
func parseTimeWithLogging(timeStr string, fieldName string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	
	parsedTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		logger.Log(fmt.Sprintf("Warning: failed to parse %s time '%s': %v", fieldName, timeStr, err))
		return time.Time{}
	}
	
	return parsedTime
}

// Webhook represents a webhook configuration
type Webhook struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Secret      string    `json:"secret,omitempty"` // Secret for HMAC signature validation
	Event       string    `json:"event"`            // Event type (e.g., "deployment", "task-execution")
	UserID      int       `json:"user_id"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WebhookPayload represents the data sent in a webhook request
type WebhookPayload struct {
	Event      string         `json:"event"`
	Action     string         `json:"action"`
	Status     string         `json:"status"`
	Timestamp  time.Time      `json:"timestamp"`
	ConfigPath string         `json:"config_path,omitempty"`
	TaskName   string         `json:"task_name,omitempty"`
	Host       string         `json:"host,omitempty"`
	UserID     int            `json:"user_id,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
}

// CreateWebhook creates a new webhook in the database
func CreateWebhook(db *sql.DB, webhook Webhook) (int, error) {
	query := `
		INSERT INTO webhooks (
			name, description, url, secret, event, user_id, active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	result, err := db.Exec(
		query,
		webhook.Name,
		webhook.Description,
		webhook.URL,
		webhook.Secret,
		webhook.Event,
		webhook.UserID,
		webhook.Active,
		now,
		now,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create webhook: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get webhook ID: %v", err)
	}

	return int(id), nil
}

// GetWebhooks retrieves all webhooks for a user
func GetWebhooks(db *sql.DB, userID int) ([]Webhook, error) {
	query := `
		SELECT id, name, description, url, event, user_id, active, created_at, updated_at
		FROM webhooks
		WHERE user_id = ?
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhooks: %v", err)
	}
	defer rows.Close()

	var webhooks []Webhook
	for rows.Next() {
		var webhook Webhook
		var createdAt, updatedAt string
		err := rows.Scan(
			&webhook.ID,
			&webhook.Name,
			&webhook.Description,
			&webhook.URL,
			&webhook.Event,
			&webhook.UserID,
			&webhook.Active,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %v", err)
		}

		webhook.CreatedAt = parseTimeWithLogging(createdAt, "created_at")
		webhook.UpdatedAt = parseTimeWithLogging(updatedAt, "updated_at")
		webhooks = append(webhooks, webhook)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during webhook row iteration: %v", err)
	}

	return webhooks, nil
}

// GetWebhooksByEvent retrieves all active webhooks for a specific event
func GetWebhooksByEvent(db *sql.DB, event string) ([]Webhook, error) {
	query := `
		SELECT id, name, description, url, secret, event, user_id, active, created_at, updated_at
		FROM webhooks
		WHERE event = ? AND active = 1
	`
	rows, err := db.Query(query, event)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhooks: %v", err)
	}
	defer rows.Close()

	var webhooks []Webhook
	for rows.Next() {
		var webhook Webhook
		var createdAt, updatedAt string
		err := rows.Scan(
			&webhook.ID,
			&webhook.Name,
			&webhook.Description,
			&webhook.URL,
			&webhook.Secret,
			&webhook.Event,
			&webhook.UserID,
			&webhook.Active,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %v", err)
		}

		webhook.CreatedAt = parseTimeWithLogging(createdAt, "created_at")
		webhook.UpdatedAt = parseTimeWithLogging(updatedAt, "updated_at")
		webhooks = append(webhooks, webhook)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during webhook row iteration: %v", err)
	}

	return webhooks, nil
}

// GetWebhook retrieves a webhook by ID
func GetWebhook(db *sql.DB, id int, userID int) (Webhook, error) {
	query := `
		SELECT id, name, description, url, secret, event, user_id, active, created_at, updated_at
		FROM webhooks
		WHERE id = ? AND user_id = ?
	`
	var webhook Webhook
	var createdAt, updatedAt string
	err := db.QueryRow(query, id, userID).Scan(
		&webhook.ID,
		&webhook.Name,
		&webhook.Description,
		&webhook.URL,
		&webhook.Secret,
		&webhook.Event,
		&webhook.UserID,
		&webhook.Active,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return Webhook{}, fmt.Errorf("failed to get webhook: %v", err)
	}

	webhook.CreatedAt = parseTimeWithLogging(createdAt, "created_at")
	webhook.UpdatedAt = parseTimeWithLogging(updatedAt, "updated_at")
	return webhook, nil
}

// UpdateWebhook updates a webhook
func UpdateWebhook(db *sql.DB, webhook Webhook) error {
	query := `
		UPDATE webhooks
		SET name = ?, description = ?, url = ?, secret = ?, event = ?, active = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`
	_, err := db.Exec(
		query,
		webhook.Name,
		webhook.Description,
		webhook.URL,
		webhook.Secret,
		webhook.Event,
		webhook.Active,
		time.Now(),
		webhook.ID,
		webhook.UserID,
	)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %v", err)
	}
	return nil
}

// DeleteWebhook deletes a webhook
func DeleteWebhook(db *sql.DB, id int, userID int) error {
	query := `DELETE FROM webhooks WHERE id = ? AND user_id = ?`
	_, err := db.Exec(query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %v", err)
	}
	return nil
}

// TriggerWebhooks sends the payload to all webhooks for a specific event
func TriggerWebhooks(db *sql.DB, event string, payload WebhookPayload) {
	webhooks, err := GetWebhooksByEvent(db, event)
	if err != nil {
		logger.Log(fmt.Sprintf("Failed to get webhooks for event %s: %v", event, err))
		return
	}

	for _, webhook := range webhooks {
		go sendWebhook(webhook, payload)
	}
}

// sendWebhook sends a webhook payload to the configured URL
func sendWebhook(webhook Webhook, payload WebhookPayload) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Log(fmt.Sprintf("Failed to marshal webhook payload: %v", err))
		return
	}

	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.Log(fmt.Sprintf("Failed to create webhook request: %v", err))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NyatiCtl-Webhook")

	// Add signature if webhook has a secret
	if webhook.Secret != "" {
		signature := calculateSignature(payloadBytes, webhook.Secret)
		req.Header.Set("X-NyatiCtl-Signature", signature)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log(fmt.Sprintf("Failed to send webhook: %v", err))
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			if err := resp.Body.Close(); err != nil {
				logger.Log(fmt.Sprintf("Failed to close webhook response body: %v", err))
			}
		}
	}()

	// Record webhook response code
	logger.Log(fmt.Sprintf("Webhook %s (%d) delivered: Status %d", webhook.Name, webhook.ID, resp.StatusCode))
}

// calculateSignature generates an HMAC signature for webhook payloads
func calculateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// verifySignature verifies the webhook signature
func verifySignature(payload []byte, secret string, signature string) bool {
	expectedSignature := calculateSignature(payload, secret)
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// ProcessIncomingWebhook handles incoming webhook requests
func ProcessIncomingWebhook(db *sql.DB, w http.ResponseWriter, r *http.Request, webhookID string) {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body)) // Reset body for future reads

	// Parse the webhook ID
	var id int
	_, err = fmt.Sscanf(webhookID, "%d", &id)
	if err != nil {
		http.Error(w, "Invalid webhook ID", http.StatusBadRequest)
		return
	}

	// Get the webhook configuration
	// Note: For incoming webhooks, we don't check user_id as these are publicly accessible
	query := `SELECT secret FROM webhooks WHERE id = ? AND active = 1`
	var secret string
	err = db.QueryRow(query, id).Scan(&secret)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Webhook not found or inactive", http.StatusNotFound)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Verify signature if secret is provided
	if secret != "" {
		signature := r.Header.Get("X-Hub-Signature-256")
		if signature == "" {
			signature = r.Header.Get("X-GitHub-Signature-256") // GitHub specific
		}
		if signature == "" {
			signature = r.Header.Get("X-GitLab-Token") // GitLab specific
		}

		// If no signature found but secret required
		if signature == "" {
			http.Error(w, "Missing signature header", http.StatusUnauthorized)
			return
		}

		// Verify the signature
		if !verifySignature(body, secret, signature) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// At this point, the webhook is authenticated and validated
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Log the incoming webhook
	logger.Log(fmt.Sprintf("Received webhook %d: %+v", id, payload))

	// TODO: Process the webhook payload (e.g., trigger a deployment or task)
	// This will depend on the specific implementation requirements

	// Return success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "webhook processed"})
}
