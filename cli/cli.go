package cli

import (
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"github.com/zechtz/nyatictl/config"
	"github.com/zechtz/nyatictl/env"
	"github.com/zechtz/nyatictl/ssh"
	"github.com/zechtz/nyatictl/tasks"
)

// Execute initializes and executes the root Cobra command for nyatictl.
//
// It sets up command-line flags, handles configuration loading,
// determines which tasks to run, and delegates the core logic
// to the Run() function.
//
// Parameters:
//   - version: A string specifying the version of the application.
//
// Returns:
//   - error: If any error occurs during execution, it will be returned.
func Execute(version string) error {
	var cfgFile string    // Path to configuration file
	var deployHost string // Host to deploy tasks to (e.g., "all", "server1")
	var taskName string   // Optional task name to execute
	var includeLib bool   // Whether to include "lib" tasks
	var debug bool        // Enable debug output
	var envName string    // Environment to use for deployment
	var envFile string    // Path to environment file

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
			// Display help if explicitly requested
			if cmd.Flag("help").Changed {
				PrintHelp(nil)
			}

			// Automatically infer config file if not provided
			if !cmd.Flag("config").Changed {
				if _, err := os.Stat("nyati.yaml"); err == nil {
					cfgFile = "nyati.yaml"
				} else if _, err := os.Stat("nyati.yml"); err == nil {
					cfgFile = "nyati.yml"
				} else {
					return fmt.Errorf("no config file found; expected nyati.yaml or nyati.yml in current directory")
				}
			}

			// Load the configuration file
			cfg, err := config.Load(cfgFile, version)
			if err != nil {
				return err
			}

			// Override args if deploy flag is set
			if deployHost != "" {
				args = []string{"deploy", deployHost}
			}

			// Execute main logic
			return Run(cfg, args, taskName, includeLib, debug)
		},
	}

	// Add database migration commands
	setupMigrationCommands(rootCmd)

	// Define supported flags
	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "Path to config file (default: nyati.yaml or nyati.yml in current directory)")
	rootCmd.Flags().StringVar(&deployHost, "deploy", "", "Host to deploy tasks on (e.g., 'all' or 'server1')")
	rootCmd.Flags().StringVar(&taskName, "task", "", "Specific task to run (e.g., 'clean')")
	rootCmd.Flags().BoolVar(&includeLib, "include-lib", false, "Include tasks marked as lib")
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	rootCmd.Flags().StringVarP(&envName, "env", "e", "", "Environment to use for deployment")
	rootCmd.Flags().StringVar(&envFile, "env-file", env.DefaultEnvFile, "Path to environment file")
	rootCmd.Flags().BoolP("help", "h", false, "Show help")

	// Start CLI
	return rootCmd.Execute()
}

// Run handles the core task execution workflow.
//
// It creates SSH clients, filters and sorts tasks (with or without dependencies),
// and executes them on the target hosts.
//
// Parameters:
//   - cfg: The loaded configuration object
//   - args: CLI arguments determining what to run
//   - taskName: Optional specific task to run
//   - includeLib: Whether to include tasks marked as lib
//   - debug: Enable debug output
//
// Returns:
//   - error: Any encountered error
func Run(cfg *config.Config, args []string, taskName string, includeLib bool, debug bool) error {
	// Display help if nothing to do
	if len(args) == 0 && !hasDeployFlag(args) {
		PrintHelp(cfg)
		return nil
	}

	// Initialize SSH clients
	clients, err := ssh.NewManager(cfg, args, debug)
	if err != nil {
		return err
	}
	defer clients.Close()

	// Establish SSH connections
	if err := clients.Open(); err != nil {
		return err
	}

	// Determine which tasks to run
	var tasksToRun []config.Task
	if taskName != "" {
		// Run only the specified task and its dependencies
		for _, task := range cfg.Tasks {
			if task.Name == taskName {
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
		// Run all tasks, optionally excluding lib tasks
		var filteredTasks []config.Task
		for _, task := range cfg.Tasks {
			if task.Lib && !includeLib {
				continue
			}
			filteredTasks = append(filteredTasks, task)
		}

		// Sort tasks by dependency order
		sortedTasks, err := topologicalSort(filteredTasks)
		if err != nil {
			return err
		}
		tasksToRun = sortedTasks
	}

	// Run the tasks over SSH
	return tasks.Run(clients, tasksToRun, debug)
}

// getTaskWithDependencies builds a dependency-aware list of tasks,
// starting from the named task and including all its prerequisites.
//
// Parameters:
//   - tasks: List of all tasks from config
//   - taskName: Name of the entry task
//
// Returns:
//   - []config.Task: Ordered list of tasks
//   - error: If the task or its dependencies are missing
func getTaskWithDependencies(tasks []config.Task, taskName string) ([]config.Task, error) {
	taskMap := make(map[string]config.Task)
	for _, task := range tasks {
		taskMap[task.Name] = task
	}

	var selectedTasks []config.Task
	visited := make(map[string]bool)

	var collectDeps func(string) error
	collectDeps = func(name string) error {
		if visited[name] {
			return nil
		}
		task, ok := taskMap[name]
		if !ok {
			return fmt.Errorf("task '%s' not found", name)
		}
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

	return topologicalSort(selectedTasks)
}

// topologicalSort returns tasks sorted in dependency-respecting order.
//
// It uses Kahn's algorithm to detect cycles and establish execution order.
//
// Parameters:
//   - tasks: List of tasks to sort
//
// Returns:
//   - []config.Task: Ordered list of tasks
//   - error: If a cyclic dependency is found
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

// hasDeployFlag checks if "deploy" keyword is present in CLI args.
//
// Parameters:
//   - args: List of CLI arguments
//
// Returns:
//   - bool: True if "deploy" is in args
func hasDeployFlag(args []string) bool {
	return slices.Contains(args, "deploy")
}

// PrintHelp prints help message and optionally configuration details.
//
// Parameters:
//   - cfg: Optional loaded config to display host/version info
func PrintHelp(cfg *config.Config) {
	fmt.Println("   -^- Nyatictl -^-    ")
	fmt.Println("Usage:")
	fmt.Println("\tnyatictl [-c config.yaml] [-d] [deploy hostname] [--task taskname] [--include-lib] [hostname]")
	fmt.Println("\tnyatictl [-c config.yaml] [deploy all] [--task taskname] [--include-lib]")
	fmt.Println("\tnyatictl env - Environment management commands")
	fmt.Println("\nFlags:")
	fmt.Println("\t-c, --config string   Path to config file (default: nyati.yaml or nyati.yml in current directory)")
	fmt.Println("\tdeploy string         Host to deploy tasks on (e.g., 'all' or 'server1')")
	fmt.Println("\t--task string         Specific task to run (e.g., 'clean')")
	fmt.Println("\t--include-lib         Include tasks marked as lib (default false)")
	fmt.Println("\t-e, --env string      Environment to use for deployment")
	fmt.Println("\t--env-file string     Path to environment file (default: nyati.env.json)")
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
