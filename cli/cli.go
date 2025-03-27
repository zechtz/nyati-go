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
				tasksToRun = append(tasksToRun, task)
				break
			}
		}
		if len(tasksToRun) == 0 {
			return fmt.Errorf("task '%s' not found", taskName)
		}
	} else {
		for _, task := range cfg.Tasks {
			if task.Lib && !includeLib {
				continue // Skip lib tasks unless --include-lib is set
			}
			tasksToRun = append(tasksToRun, task)
		}
	}

	return tasks.Run(clients, tasksToRun, debug)
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
