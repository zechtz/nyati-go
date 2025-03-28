package tasks

import (
	"fmt"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/manifoldco/promptui"
	"github.com/zechtz/nyatictl/config"
	"github.com/zechtz/nyatictl/logger"
	"github.com/zechtz/nyatictl/ssh"
)

// Run concurrently executes a list of deployment tasks across multiple SSH clients.
//
// For each task, it spins up one goroutine per client to execute the command in parallel.
// Results are collected, and optional retry logic is supported for failed executions.
// Debug output and task-specific output can be conditionally displayed based on task config.
//
// Parameters:
//   - m: A reference to the SSH Manager, which contains all connected clients
//   - tasks: List of config.Task objects to execute
//   - debug: Enables debug logging if set to true
//
// Returns:
//   - error: Returns on the first encountered failure (aggregating errors could be future enhancement)
func Run(m *ssh.Manager, tasks []config.Task, debug bool) error {
	var wg sync.WaitGroup

	// Buffered channel to capture possible errors
	errChan := make(chan error, len(m.Clients)*len(tasks))

	// Iterate over each task in the execution plan
	for _, task := range tasks {
		wg.Add(len(m.Clients)) // Add to waitgroup: one for each client

		// Create a spinner (animated loading indicator) for visual feedback
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Prefix = fmt.Sprintf("🎲 %s: ", task.Name)

		// Launch concurrent execution for each SSH client
		for _, client := range m.Clients {
			go func(c *ssh.Client, t config.Task) {
				defer wg.Done()

				s.Start()
				logger.Log(s.Prefix)

				// Execute the command over SSH
				code, output, err := c.Exec(t, debug)
				if err != nil {
					errMsg := fmt.Sprintf("❌ %s@%s: Failed", t.Name, c.Name)
					s.FinalMSG = errMsg + "\n"
					logger.Log(errMsg)
					s.Stop()

					errChan <- fmt.Errorf("%s@%s: %v", c.Name, c.Server.Host, err)
					return
				}

				// If exit code does not match expected, handle retry or log failure
				if code != t.Expect {
					errMsg := fmt.Sprintf("❌ %s@%s: Failed (code %d)", t.Name, c.Name, code)
					s.FinalMSG = errMsg + "\n"
					logger.Log(errMsg)
					s.Stop()

					// Display output if necessary
					if debug || t.Output || t.Retry {
						logger.Log(output)
						fmt.Println(output)
					}

					// Prompt user for retry if the task allows it
					if t.Retry {
						prompt := promptui.Prompt{
							Label:     fmt.Sprintf("Retry '%s' on %s", t.Name, c.Name),
							IsConfirm: true,
						}
						if _, err := prompt.Run(); err == nil {
							// Retry the task once more
							_, _, err = c.Exec(t, debug)
							if err == nil && code == t.Expect {
								successMsg := fmt.Sprintf("🎉 %s@%s: Succeeded after retry", t.Name, c.Name)
								s.FinalMSG = successMsg + "\n"
								logger.Log(successMsg)
							}
						}
					}

					errChan <- fmt.Errorf("task %s failed on %s", t.Name, c.Name)
					return
				}

				// Task completed successfully
				successMsg := fmt.Sprintf("🎉 %s@%s: Succeeded", t.Name, c.Name)
				s.FinalMSG = successMsg + "\n"
				logger.Log(successMsg)
				s.Stop()

				// Output command logs based on flags
				if debug || t.Output || t.Message != "" {
					logger.Log(output)
					fmt.Println(output)
				}

				// Display task message, if present
				if t.Message != "" {
					msg := fmt.Sprintf("📗 %s", t.Message)
					logger.Log(msg)
					fmt.Printf("%s\n", msg)
				}
			}(client, task)
		}

		// Wait for all clients to finish this task
		wg.Wait()
	}

	// After all tasks are processed, check for errors
	close(errChan)
	for err := range errChan {
		return err // Return first error found
	}

	return nil
}
