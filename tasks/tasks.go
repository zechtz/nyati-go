package tasks

import (
	"fmt"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/manifoldco/promptui"
	"github.com/zechtz/nyatictl/config"
	"github.com/zechtz/nyatictl/ssh"
)

// Run executes tasks across all clients concurrently.
func Run(m *ssh.Manager, tasks []config.Task, debug bool) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(m.Clients)*len(tasks))

	for _, task := range tasks {
		wg.Add(len(m.Clients))
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Prefix = fmt.Sprintf("🎲 %s: ", task.Name)

		for _, client := range m.Clients {
			go func(c *ssh.Client, t config.Task) {
				defer wg.Done()
				s.Start()
				code, output, err := c.Exec(t, debug)
				if err != nil {
					errChan <- fmt.Errorf("%s@%s: %v", c.Name, c.Server.Host, err)
					s.FinalMSG = fmt.Sprintf("❌ %s@%s: Failed\n", t.Name, c.Name)
					s.Stop()
					return
				}
				if code != t.Expect {
					s.FinalMSG = fmt.Sprintf("❌ %s@%s: Failed (code %d)\n", t.Name, c.Name, code)
					s.Stop()
					if debug || t.Output || t.Retry {
						fmt.Println(output)
					}
					if t.Retry {
						prompt := promptui.Prompt{
							Label:     fmt.Sprintf("Retry '%s' on %s", t.Name, c.Name),
							IsConfirm: true,
						}
						if _, err := prompt.Run(); err == nil {
							// Retry logic: recursive call
							_, _, err = c.Exec(t, debug)
							if err == nil && code == t.Expect {
								s.FinalMSG = fmt.Sprintf("🎉 %s@%s: Succeeded after retry\n", t.Name, c.Name)
							}
						}
					}
					errChan <- fmt.Errorf("task %s failed on %s", t.Name, c.Name)
					return
				}
				s.FinalMSG = fmt.Sprintf("🎉 %s@%s: Succeeded\n", t.Name, c.Name)
				s.Stop()
				if debug || t.Output || t.Message != "" {
					fmt.Println(output)
				}
				if t.Message != "" {
					fmt.Printf("📗 %s\n", t.Message)
				}
			}(client, task)
		}
		wg.Wait()
	}

	close(errChan)
	for err := range errChan {
		return err // Return first error for simplicity; could collect all
	}
	return nil
}
