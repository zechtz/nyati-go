package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zechtz/nyatictl/api/response"
	"github.com/zechtz/nyatictl/config"
	"github.com/zechtz/nyatictl/logger"
)

// SimulationRequest represents the request parameters for a sandbox simulation
type SimulationRequest struct {
	ConfigPath string `json:"configPath"` // Path to the configuration file
	Host       string `json:"host"`       // Target host to simulate deployment on
	SessionID  string `json:"sessionID"`  // Session ID for tracking and logging
}

// SimulationTaskResult represents the outcome of a simulated task
type SimulationTaskResult struct {
	Name       string `json:"name"`       // Task name
	Successful bool   `json:"successful"` // Whether the simulation succeeded
	Output     string `json:"output"`     // Simulated command output
	Duration   int    `json:"duration"`   // Simulated execution time in milliseconds
}

// SimulationResponse contains the complete results of a simulation
type SimulationResponse struct {
	SuccessRate float64                `json:"successRate"` // Overall success rate (0-100)
	Tasks       []SimulationTaskResult `json:"tasks"`       // Individual task results
	Host        string                 `json:"host"`        // Host the simulation ran against
	Duration    int                    `json:"duration"`    // Total simulation time in milliseconds
}

// handleSandboxSimulation processes a request to simulate deployment without executing real SSH commands
func (s *Server) handleSandboxSimulation(w http.ResponseWriter, r *http.Request) {
	rw := response.NewWriter(w)

	// Get user ID from the JWT claims in context
	claims, ok := GetUserFromContext(r)
	if !ok {
		rw.Unauthorized("Unauthorized")
		return
	}

	var req SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rw.BadRequest("Invalid request body")
		return
	}

	// Check if the user owns this config
	var userID int
	err := s.db.QueryRow("SELECT user_id FROM configs WHERE path = ?", req.ConfigPath).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			rw.NotFound("Config not found")
		} else {
			rw.InternalServerError(err.Error())
		}
		return
	}

	// Verify ownership
	if userID != claims.UserID {
		rw.Forbidden("You don't have permission to simulate this config")
		return
	}

	// Load the configuration file
	cfg, err := config.Load(req.ConfigPath, "0.1.2")
	if err != nil {
		rw.InternalServerError(err.Error())
		return
	}

	// Create a log channel scoped to this session
	logChan := make(chan string, 100)
	s.logLock.Lock()
	s.logChannels[req.SessionID] = logChan
	s.logLock.Unlock()

	// Simulate the deployment in a goroutine to allow for streaming logs
	go func() {
		defer func() {
			s.logLock.Lock()
			delete(s.logChannels, req.SessionID)
			close(logChan)
			s.logLock.Unlock()
		}()

		// Initialize random number generator with a seed for consistent results
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))

		// Log simulation start
		logger.Log(fmt.Sprintf("[SANDBOX] Starting simulation for config: %s on host: %s", req.ConfigPath, req.Host))

		// Determine which hosts to simulate
		var hostsToSimulate []string
		if req.Host == "all" {
			for hostName := range cfg.Hosts {
				hostsToSimulate = append(hostsToSimulate, hostName)
			}
		} else if _, exists := cfg.Hosts[req.Host]; exists {
			hostsToSimulate = append(hostsToSimulate, req.Host)
		} else {
			logger.Log(fmt.Sprintf("[SANDBOX] Error: Host '%s' not found in config", req.Host))
			return
		}

		// Sort tasks by dependency order (using the same logic as real deployments)
		sortedTasks, err := topologicalSort(cfg.Tasks)
		if err != nil {
			logger.Log(fmt.Sprintf("[SANDBOX] Error sorting tasks: %v", err))
			return
		}

		// Simulate each task on each selected host
		for _, host := range hostsToSimulate {
			for _, task := range sortedTasks {
				// Skip lib tasks unless they are explicitly included
				if task.Lib {
					continue
				}

				// Simulate a delay to make the simulation feel realistic
				time.Sleep(time.Duration(500+rng.Intn(1000)) * time.Millisecond)

				// Simulate a 90% success rate
				success := rng.Float64() <= 0.9

				var logMsg string
				if success {
					logMsg = fmt.Sprintf("[SANDBOX] Task '%s' on host '%s' completed successfully", task.Name, host)
					logger.Log(logMsg)

					// If task has output enabled, simulate some command output
					if task.Output {
						outputMsg := fmt.Sprintf("[SANDBOX] Output for '%s':\n> Command executed in working directory: %s\n> Exit code: 0",
							task.Name, task.Dir)
						logger.Log(outputMsg)
					}

					// If task has a success message, display it
					if task.Message != "" {
						msgOutput := fmt.Sprintf("[SANDBOX] Message: %s", task.Message)
						logger.Log(msgOutput)
					}
				} else {
					// Simulate random failure reasons
					failureReasons := []string{
						"Connection timed out",
						"Permission denied",
						"Command not found",
						"No such file or directory",
						"Unable to allocate memory",
					}

					reason := failureReasons[rng.Intn(len(failureReasons))]
					logMsg = fmt.Sprintf("[SANDBOX] Task '%s' on host '%s' failed: %s", task.Name, host, reason)
					logger.Log(logMsg)
				}
			}
		}

		logger.Log("[SANDBOX] Simulation completed")
	}()

	// Return immediate acknowledgement
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message":   "Simulation started",
		"sessionId": req.SessionID,
	})
}

// Helper function to copy from cli/cli.go (we reuse the topological sort functionality)
func topologicalSort(tasks []config.Task) ([]config.Task, error) {
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	taskMap := make(map[string]config.Task)

	for _, task := range tasks {
		taskMap[task.Name] = task
		if _, ok := inDegree[task.Name]; !ok {
			inDegree[task.Name] = 0
		}
		for _, dep := range task.DependsOn {
			graph[dep] = append(graph[dep], task.Name)
			inDegree[task.Name]++
		}
	}

	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var sortedTasks []config.Task
	for len(queue) > 0 {
		taskName := queue[0]
		queue = queue[1:]
		sortedTasks = append(sortedTasks, taskMap[taskName])

		for _, dep := range graph[taskName] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(sortedTasks) != len(tasks) {
		return nil, fmt.Errorf("unexpected cycle in task dependencies")
	}

	return sortedTasks, nil
}

// RegisterSandboxRoutes adds blueprint-related routes to the API router
func (s *Server) RegisterSandboxRoutes(router *mux.Router) {
	// Blueprint endpoints
	router.HandleFunc("/sandbox", s.handleSandboxSimulation).Methods("GET")
}
