package response

import (
	"encoding/json"
	"log"
	"net/http"
)

// Response represents the structure of a standardized JSON response for API endpoints.
// It provides a consistent format for both success and error responses.
//
// Fields:
//   - Success: A boolean indicating whether the request was successful.
//   - Data: Optional payload containing any data to return to the client when the request succeeds.
//   - Error: Optional error message describing what went wrong when the request fails.
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Writer is a utility that wraps an http.ResponseWriter to simplify writing
// standardized JSON responses across the application.
// It encapsulates common HTTP response logic and formats.
type Writer struct {
	w http.ResponseWriter
}

// NewWriter creates a new instance of the Writer wrapper around an existing http.ResponseWriter.
//
// Parameters:
//   - w: The underlying http.ResponseWriter provided by the net/http framework.
//
// Returns:
//   - A pointer to a new Writer instance for handling API responses.
func NewWriter(w http.ResponseWriter) *Writer {
	return &Writer{w: w}
}

// writeJSON is a helper method that encodes and writes a JSON response to the client
// using the given HTTP status code and Response struct.
//
// Parameters:
//   - status: The HTTP status code to set in the response header.
//   - resp: The Response struct to encode and send to the client.
//
// Behavior:
//   - Sets the "Content-Type" header to "application/json".
//   - Writes the status code.
//   - Serializes the Response struct as JSON and writes it to the response body.
//   - Logs an error if JSON encoding fails.
func (rw *Writer) writeJSON(status int, resp Response) {
	rw.w.Header().Set("Content-Type", "application/json")
	rw.w.WriteHeader(status)

	if err := json.NewEncoder(rw.w).Encode(resp); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// Success sends a 200 OK response with the provided data payload,
// indicating that the request was successfully processed.
//
// Parameters:
//   - data: The data to include in the response body.
func (rw *Writer) Success(data any) {
	rw.writeJSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// Created sends a 201 Created response with the provided data payload,
// typically used after a successful creation of a resource.
//
// Parameters:
//   - data: The data to include in the response body.
func (rw *Writer) Created(data any) {
	rw.writeJSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// NoContent sends a 204 No Content response, indicating that the request
// was successful but there is no content to return.
//
// This is commonly used for delete operations or updates where no response body is needed.
func (rw *Writer) NoContent() {
	rw.w.Header().Set("Content-Type", "application/json")
	rw.w.WriteHeader(http.StatusNoContent)
}

// Error sends a JSON error response with the given status code and error message.
// It indicates that the request failed due to a client or server error.
//
// Parameters:
//   - status: The HTTP status code to use (e.g., 400, 404, 500).
//   - message: The error message to include in the response.
func (rw *Writer) Error(status int, message string) {
	rw.writeJSON(status, Response{
		Success: false,
		Error:   message,
	})
}

// BadRequest sends a 400 Bad Request error response,
// indicating that the client sent invalid or malformed input.
//
// Parameters:
//   - message: A descriptive error message to help the client fix their request.
func (rw *Writer) BadRequest(message string) {
	rw.Error(http.StatusBadRequest, message)
}

// Unauthorized sends a 401 Unauthorized error response,
// indicating that the request requires authentication and none was provided or was invalid.
//
// Parameters:
//   - message: An optional message explaining the authentication error.
func (rw *Writer) Unauthorized(message string) {
	rw.Error(http.StatusUnauthorized, message)
}

// Forbidden sends a 403 Forbidden error response,
// indicating that the client does not have permission to access the requested resource.
//
// Parameters:
//   - message: An explanation of why access is forbidden.
func (rw *Writer) Forbidden(message string) {
	rw.Error(http.StatusForbidden, message)
}

// NotFound sends a 404 Not Found error response,
// indicating that the requested resource does not exist.
//
// Parameters:
//   - message: An explanation of what was not found.
func (rw *Writer) NotFound(message string) {
	rw.Error(http.StatusNotFound, message)
}

// InternalServerError sends a 500 Internal Server Error response,
// indicating that an unexpected server-side error occurred while processing the request.
//
// Parameters:
//   - message: A message describing the error, for logging or debugging purposes.
func (rw *Writer) InternalServerError(message string) {
	rw.Error(http.StatusInternalServerError, message)
}
