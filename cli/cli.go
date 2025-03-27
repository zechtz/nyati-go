package cli

import (
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"github.com/zechtz/nyatictl/config"
	"github.com/zechtz/nyatictl/ssh"
	"github.com/zechtz/nyatictl/tasks"
)

// Execute runs the CLI with the given version.
func Execute(version string) error {
	var cfgFile string
	var deployHost string
	var taskName string
	var includeLib bool
	var debug bool

	rootCmd := &cobra.Command{
		Use:   "nyatictl",
		Short: "A remote server automation and deployment tool",
		Long: `Nyatictl is a CLI tool for remote server automation and deployment.
It executes tasks on specified hosts via SSH, inspired by Capistrano.

Usage examples:
  nyatictl [-c nyati.yaml] deploy all    # Run all tasks on all hosts (excludes lib tasks)
  nyatictl [-c nyati.yaml] deploy all --include-lib  # Include lib tasks
  nyatictl [-c nyati.yaml] deploy server1 --task clean  # Run the 'clean' task on server1
  nyatictl [-c nyati.yaml] server1       # Shorthand for deploy server1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If --help is requested, show help and exit
			if cmd.Flag("help").Changed {
				PrintHelp(nil)
			}

			// Automatically detect nyati.yaml or nyati.yml if -c is not provided
			if !cmd.Flag("config").Changed {
				if _, err := os.Stat("nyati.yaml"); err == nil {
					cfgFile = "nyati.yaml"
				} else if _, err := os.Stat("nyati.yml"); err == nil {
					cfgFile = "nyati.yml"
				} else {
					return fmt.Errorf("no config file found; expected nyati.yaml or nyati.yml in current directory")
				}
			}

			cfg, err := config.Load(cfgFile, version)
			if err != nil {
				return err
			}

			// Pass deployHost as an argument if set, otherwise use args
			if deployHost != "" {
				args = []string{"deploy", deployHost}
			}

			return Run(cfg, args, taskName, includeLib, debug)
		},
	}

	// Define flags
	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "Path to config file (default: nyati.yaml or nyati.yml in current directory)")
	rootCmd.Flags().StringVar(&deployHost, "deploy", "", "Host to deploy tasks on (e.g., 'all' or 'server1')")
	rootCmd.Flags().StringVar(&taskName, "task", "", "Specific task to run (e.g., 'clean')")
	rootCmd.Flags().BoolVar(&includeLib, "include-lib", false, "Include tasks marked as lib")
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	rootCmd.Flags().BoolP("help", "h", false, "Show help")

	return rootCmd.Execute()
}

// Run executes the main logic: loads clients, runs tasks, and cleans up.
func Run(cfg *config.Config, args []string, taskName string, includeLib bool, debug bool) error {
	// Show help if no arguments are provided and deploy isn't set
	if len(args) == 0 && !hasDeployFlag(args) {
		PrintHelp(cfg)
		return nil
	}

	clients, err := ssh.NewManager(cfg, args, debug)
	if err != nil {
		return err
	}

	defer clients.Close()

	if err := clients.Open(); err != nil {
		return err
	}

	// Filter tasks
	var tasksToRun []config.Task
	if taskName != "" {
		for _, task := range cfg.Tasks {
			if task.Name == taskName {
				// When running a specific task, include its dependencies
				deps, err := getTaskWithDependencies(cfg.Tasks, taskName)
				if err != nil {
					return err
				}
				tasksToRun = deps
				break
			}
		}
		if len(tasksToRun) == 0 {
			return fmt.Errorf("task '%s' not found", taskName)
		}
	} else {
		// Filter out lib tasks if --include-lib is not set
		var filteredTasks []config.Task
		for _, task := range cfg.Tasks {
			if task.Lib && !includeLib {
				continue
			}
			filteredTasks = append(filteredTasks, task)
		}

		// Sort tasks by dependencies
		sortedTasks, err := topologicalSort(filteredTasks)
		if err != nil {
			return err
		}
		tasksToRun = sortedTasks
	}

	return tasks.Run(clients, tasksToRun, debug)
}

// getTaskWithDependencies returns the task and its dependencies in topological order.
func getTaskWithDependencies(tasks []config.Task, taskName string) ([]config.Task, error) {
	// Build a map of task names to tasks for quick lookup
	taskMap := make(map[string]config.Task)
	for _, task := range tasks {
		taskMap[task.Name] = task
	}

	// Collect the task and its dependencies
	var selectedTasks []config.Task
	visited := make(map[string]bool)

	var collectDeps func(name string) error
	collectDeps = func(name string) error {
		if visited[name] {
			return nil
		}
		task, exists := taskMap[name]
		if !exists {
			return fmt.Errorf("task '%s' not found", name)
		}
		// Recursively collect dependencies
		for _, dep := range task.DependsOn {
			if err := collectDeps(dep); err != nil {
				return err
			}
		}
		visited[name] = true
		selectedTasks = append(selectedTasks, task)
		return nil
	}

	if err := collectDeps(taskName); err != nil {
		return nil, err
	}

	// Sort the selected tasks by dependencies
	sortedTasks, err := topologicalSort(selectedTasks)
	if err != nil {
		return nil, err
	}

	return sortedTasks, nil
}

// topologicalSort sorts tasks based on their dependencies.
func topologicalSort(tasks []config.Task) ([]config.Task, error) {
	// Build a graph and in-degree map
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	taskMap := make(map[string]config.Task)

	for _, task := range tasks {
		taskMap[task.Name] = task
		if _, exists := inDegree[task.Name]; !exists {
			inDegree[task.Name] = 0
		}
		for _, dep := range task.DependsOn {
			graph[dep] = append(graph[dep], task.Name)
			inDegree[task.Name]++
		}
	}

	// Initialize queue with tasks that have no dependencies
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// Perform topological sort
	var sortedTasks []config.Task
	for len(queue) > 0 {
		// Dequeue a task
		taskName := queue[0]
		queue = queue[1:]

		// Add the task to the result
		sortedTasks = append(sortedTasks, taskMap[taskName])

		// Process dependencies
		for _, dep := range graph[taskName] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	// Check for cycles (shouldn't happen due to earlier validation, but just in case)
	if len(sortedTasks) != len(tasks) {
		return nil, fmt.Errorf("unexpected cycle in task dependencies")
	}

	return sortedTasks, nil
}

// hasDeployFlag checks if deploy is present in args.
func hasDeployFlag(args []string) bool {
	return slices.Contains(args, "deploy")
}

// PrintHelp displays usage information.
func PrintHelp(cfg *config.Config) {
	fmt.Println("   -^- Nyatictl -^-    ")
	fmt.Println("Usage:")
	fmt.Println("\tnyatictl [-c config.yaml] [-d] [deploy hostname] [--task taskname] [--include-lib] [hostname]")
	fmt.Println("\tnyatictl [-c config.yaml] [deploy all] [--task taskname] [--include-lib]")
	fmt.Println("\nFlags:")
	fmt.Println("\t-c, --config string   Path to config file (default: nyati.yaml or nyati.yml in current directory)")
	fmt.Println("\tdeploy string         Host to deploy tasks on (e.g., 'all' or 'server1')")
	fmt.Println("\t--task string         Specific task to run (e.g., 'clean')")
	fmt.Println("\t--include-lib         Include tasks marked as lib (default false)")
	fmt.Println("\t-d, --debug           Enable debug output")
	fmt.Println("\t-h, --help            Show help")
	if cfg != nil {
		fmt.Println("\nConfig:")
		fmt.Printf("  Version: %s\n", cfg.Version)
		fmt.Printf("  App: %s\n", cfg.AppName)
		fmt.Printf("  Hosts: %d\n", len(cfg.Hosts))
		for name, host := range cfg.Hosts {
			fmt.Printf("    %s: %s@%s\n", name, host.Username, host.Host)
		}
	}
}
