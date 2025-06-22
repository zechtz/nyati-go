package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// RegisterRequest represents the registration form data
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// validateEmail checks if the email format is valid
func validateEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// validatePassword checks if password meets security requirements
func validatePassword(password string) []string {
	var errors []string
	
	if len(password) < 8 {
		errors = append(errors, "Password must be at least 8 characters long")
	}
	
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\?]`).MatchString(password)
	
	if !hasUpper {
		errors = append(errors, "Password must contain at least one uppercase letter")
	}
	if !hasLower {
		errors = append(errors, "Password must contain at least one lowercase letter")
	}
	if !hasNumber {
		errors = append(errors, "Password must contain at least one number")
	}
	if !hasSpecial {
		errors = append(errors, "Password must contain at least one special character")
	}
	
	return errors
}

// sanitizeInput removes potentially dangerous characters from input
func sanitizeInput(input string) string {
	// Remove null bytes and control characters
	cleaned := strings.ReplaceAll(input, "\x00", "")
	cleaned = regexp.MustCompile(`[\x00-\x1f\x7f]`).ReplaceAllString(cleaned, "")
	
	// Trim whitespace
	cleaned = strings.TrimSpace(cleaned)
	
	return cleaned
}

// HandleRegister processes user registration requests
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Sanitize inputs
	req.Email = sanitizeInput(req.Email)
	req.Name = sanitizeInput(req.Name)
	// Note: Don't sanitize password as it might remove valid special characters

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// Validate email format
	if !validateEmail(req.Email) {
		http.Error(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	// Validate password strength
	if passwordErrors := validatePassword(req.Password); len(passwordErrors) > 0 {
		errorMsg := "Password validation failed: " + strings.Join(passwordErrors, ", ")
		http.Error(w, errorMsg, http.StatusBadRequest)
		return
	}

	// Check if the email is already in use
	var exists bool
	err := s.db.DB.QueryRow("SELECT 1 FROM users WHERE email = ?", req.Email).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if err == nil { // User exists
		http.Error(w, "Email already in use", http.StatusConflict)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Create user record
	_, err = s.db.DB.Exec(
		"INSERT INTO users (email, password, created_at) VALUES (?, ?, ?)",
		req.Email,
		string(hashedPassword),
		time.Now().Format(time.RFC3339),
	)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}
