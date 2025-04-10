package cli

import (
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/zechtz/nyatictl/env"
	"golang.org/x/term"
)

// setupEnvCommands adds environment variable management commands to the root command
func setupEnvCommands(rootCmd *cobra.Command) {
	// Create the env command
	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Environment variable management",
		Long:  "Commands for managing environment variables across different deployment environments",
	}

	// Initialize command
	initCmd := &cobra.Command{
		Use:   "init [environment-name]",
		Short: "Initialize a new environment",
		Long:  "Create a new environment configuration file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envName := "development"
			if len(args) > 0 {
				envName = args[0]
			}

			description, _ := cmd.Flags().GetString("description")
			filePath, _ := cmd.Flags().GetString("file")

			// Create default environment file
			envFile := &env.EnvironmentFile{
				Environments: []*env.Environment{
					env.NewEnvironment(envName, description),
				},
				CurrentEnv: envName,
			}

			// Set file path
			envFile.Environments[0].FilePath = filePath

			// Save to disk
			if err := env.SaveEnvironmentFile(envFile, filePath); err != nil {
				return fmt.Errorf("failed to initialize environment: %v", err)
			}

			fmt.Printf("Environment '%s' initialized in %s\n", envName, filePath)
			return nil
		},
	}

	initCmd.Flags().StringP("description", "d", "Default environment", "Description of the environment")
	initCmd.Flags().StringP("file", "f", env.DefaultEnvFile, "Path to environment file")

	// List environments command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all environments",
		Long:  "Display all available environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")

			envFile, err := env.LoadEnvironmentFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to load environments: %v", err)
			}

			fmt.Println("Available environments:")
			fmt.Println("========================")
			for _, e := range envFile.Environments {
				current := " "
				if e.Name == envFile.CurrentEnv {
					current = "*"
				}
				fmt.Printf("%s %-15s - %s\n", current, e.Name, e.Description)
			}

			return nil
		},
	}

	listCmd.Flags().StringP("file", "f", env.DefaultEnvFile, "Path to environment file")

	// Add environment command
	addEnvCmd := &cobra.Command{
		Use:   "add [environment-name]",
		Short: "Add a new environment",
		Long:  "Create a new named environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			newEnvName := args[0]
			description, _ := cmd.Flags().GetString("description")
			filePath, _ := cmd.Flags().GetString("file")

			envFile, err := env.LoadEnvironmentFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to load environments: %v", err)
			}

			// Create new environment
			newEnv := env.NewEnvironment(newEnvName, description)

			// Add to file
			if err := env.AddEnvironment(envFile, newEnv); err != nil {
				return fmt.Errorf("failed to add environment: %v", err)
			}

			fmt.Printf("Environment '%s' added successfully\n", newEnvName)
			return nil
		},
	}

	addEnvCmd.Flags().StringP("description", "d", "", "Description of the environment")
	addEnvCmd.Flags().StringP("file", "f", env.DefaultEnvFile, "Path to environment file")

	// Use environment command
	useCmd := &cobra.Command{
		Use:   "use [environment-name]",
		Short: "Switch to an environment",
		Long:  "Set the current active environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envName := args[0]
			filePath, _ := cmd.Flags().GetString("file")

			envFile, err := env.LoadEnvironmentFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to load environments: %v", err)
			}

			if err := env.SetCurrentEnvironment(envFile, envName); err != nil {
				return fmt.Errorf("failed to switch environment: %v", err)
			}

			fmt.Printf("Switched to environment '%s'\n", envName)
			return nil
		},
	}

	useCmd.Flags().StringP("file", "f", env.DefaultEnvFile, "Path to environment file")

	// Remove environment command
	removeCmd := &cobra.Command{
		Use:   "remove [environment-name]",
		Short: "Remove an environment",
		Long:  "Delete an environment and its variables",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envName := args[0]
			filePath, _ := cmd.Flags().GetString("file")

			envFile, err := env.LoadEnvironmentFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to load environments: %v", err)
			}

			if err := env.RemoveEnvironment(envFile, envName); err != nil {
				return fmt.Errorf("failed to remove environment: %v", err)
			}

			fmt.Printf("Environment '%s' removed successfully\n", envName)
			return nil
		},
	}

	removeCmd.Flags().StringP("file", "f", env.DefaultEnvFile, "Path to environment file")

	// Set variable command
	setCmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set an environment variable",
		Long:  "Add or update an environment variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]
			isSecret, _ := cmd.Flags().GetBool("secret")
			filePath, _ := cmd.Flags().GetString("file")
			envName, _ := cmd.Flags().GetString("env")

			envFile, err := env.LoadEnvironmentFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to load environments: %v", err)
			}

			// Determine which environment to use
			var environment *env.Environment
			if envName != "" {
				environment, err = env.GetEnvironment(envFile, envName)
			} else {
				environment, err = env.GetCurrentEnvironment(envFile)
			}

			if err != nil {
				return fmt.Errorf("failed to get environment: %v", err)
			}

			// If this is a secret, we need an encryption key
			if isSecret {
				encKey, _ := cmd.Flags().GetString("key")

				// If no key provided, prompt for it
				if encKey == "" {
					fmt.Print("Enter encryption key: ")
					byteKey, err := term.ReadPassword(int(syscall.Stdin))
					if err != nil {
						return fmt.Errorf("failed to read encryption key: %v", err)
					}
					fmt.Println() // Add newline after password input
					encKey = string(byteKey)
				}

				environment.SetEncryptionKey(encKey)
			}

			// Set the variable
			if err := environment.Set(key, value, isSecret); err != nil {
				return fmt.Errorf("failed to set variable: %v", err)
			}

			// Save changes
			if err := env.SaveEnvironmentFile(envFile, filePath); err != nil {
				return fmt.Errorf("failed to save environment: %v", err)
			}

			varType := "variable"
			if isSecret {
				varType = "secret"
			}

			fmt.Printf("Set %s '%s' in environment '%s'\n", varType, key, environment.Name)
			return nil
		},
	}

	setCmd.Flags().BoolP("secret", "s", false, "Store as an encrypted secret")
	setCmd.Flags().StringP("key", "k", "", "Encryption key for secrets")
	setCmd.Flags().StringP("file", "f", env.DefaultEnvFile, "Path to environment file")
	setCmd.Flags().StringP("env", "e", "", "Target environment (defaults to current)")

	// Get variable command
	getCmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Get an environment variable",
		Long:  "Retrieve the value of an environment variable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			filePath, _ := cmd.Flags().GetString("file")
			envName, _ := cmd.Flags().GetString("env")

			envFile, err := env.LoadEnvironmentFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to load environments: %v", err)
			}

			// Determine which environment to use
			var environment *env.Environment
			if envName != "" {
				environment, err = env.GetEnvironment(envFile, envName)
			} else {
				environment, err = env.GetCurrentEnvironment(envFile)
			}

			if err != nil {
				return fmt.Errorf("failed to get environment: %v", err)
			}

			// Try to get the variable
			value, isSecret, err := environment.Get(key)

			// If it's a secret and we need a key
			if isSecret && err == env.ErrNoEncryptionKey {
				encKey, _ := cmd.Flags().GetString("key")

				// If no key provided, prompt for it
				if encKey == "" {
					fmt.Print("Enter encryption key: ")
					byteKey, err := term.ReadPassword(int(syscall.Stdin))
					if err != nil {
						return fmt.Errorf("failed to read encryption key: %v", err)
					}
					fmt.Println() // Add newline after password input
					encKey = string(byteKey)
				}

				environment.SetEncryptionKey(encKey)

				// Try again with the key
				value, _, err = environment.Get(key)
			}

			if err != nil {
				return fmt.Errorf("failed to get variable: %v", err)
			}

			if value == "" && !isSecret {
				return fmt.Errorf("variable '%s' not found in environment '%s'", key, environment.Name)
			}

			fmt.Println(value)
			return nil
		},
	}

	getCmd.Flags().StringP("key", "k", "", "Encryption key for secrets")
	getCmd.Flags().StringP("file", "f", env.DefaultEnvFile, "Path to environment file")
	getCmd.Flags().StringP("env", "e", "", "Target environment (defaults to current)")

	// Delete variable command
	delCmd := &cobra.Command{
		Use:   "del [key]",
		Short: "Delete an environment variable",
		Long:  "Remove an environment variable or secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			filePath, _ := cmd.Flags().GetString("file")
			envName, _ := cmd.Flags().GetString("env")

			envFile, err := env.LoadEnvironmentFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to load environments: %v", err)
			}

			// Determine which environment to use
			var environment *env.Environment
			if envName != "" {
				environment, err = env.GetEnvironment(envFile, envName)
			} else {
				environment, err = env.GetCurrentEnvironment(envFile)
			}

			if err != nil {
				return fmt.Errorf("failed to get environment: %v", err)
			}

			// Delete the variable
			environment.Delete(key)

			// Save changes
			if err := env.SaveEnvironmentFile(envFile, filePath); err != nil {
				return fmt.Errorf("failed to save environment: %v", err)
			}

			fmt.Printf("Deleted variable '%s' from environment '%s'\n", key, environment.Name)
			return nil
		},
	}

	delCmd.Flags().StringP("file", "f", env.DefaultEnvFile, "Path to environment file")
	delCmd.Flags().StringP("env", "e", "", "Target environment (defaults to current)")

	// List variables command
	listVarsCmd := &cobra.Command{
		Use:   "vars",
		Short: "List all variables",
		Long:  "Display all variables in the current environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			envName, _ := cmd.Flags().GetString("env")
			showSecrets, _ := cmd.Flags().GetBool("secrets")

			envFile, err := env.LoadEnvironmentFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to load environments: %v", err)
			}

			// Determine which environment to use
			var environment *env.Environment
			if envName != "" {
				environment, err = env.GetEnvironment(envFile, envName)
			} else {
				environment, err = env.GetCurrentEnvironment(envFile)
			}

			if err != nil {
				return fmt.Errorf("failed to get environment: %v", err)
			}

			// If showing secrets, we need an encryption key
			if showSecrets && len(environment.Secrets) > 0 {
				encKey, _ := cmd.Flags().GetString("key")

				// If no key provided, prompt for it
				if encKey == "" {
					fmt.Print("Enter encryption key: ")
					byteKey, err := term.ReadPassword(int(syscall.Stdin))
					if err != nil {
						return fmt.Errorf("failed to read encryption key: %v", err)
					}
					fmt.Println() // Add newline after password input
					encKey = string(byteKey)
				}

				environment.SetEncryptionKey(encKey)
			}

			fmt.Printf("Variables in environment '%s':\n", environment.Name)
			fmt.Println("============================")

			// Print regular variables
			if len(environment.Variables) > 0 {
				fmt.Println("Regular variables:")
				for k, v := range environment.Variables {
					fmt.Printf("  %s=%s\n", k, v)
				}
			} else {
				fmt.Println("No regular variables defined")
			}

			// Print secrets
			if len(environment.Secrets) > 0 {
				fmt.Println("\nSecrets:")
				if showSecrets {
					for k := range environment.Secrets {
						value, _, err := environment.Get(k)
						if err != nil {
							fmt.Printf("  %s=<error: %v>\n", k, err)
						} else {
							fmt.Printf("  %s=%s\n", k, value)
						}
					}
				} else {
					for k := range environment.Secrets {
						fmt.Printf("  %s=<encrypted>\n", k)
					}
					fmt.Println("\nUse --secrets flag to view decrypted values")
				}
			} else {
				fmt.Println("\nNo secrets defined")
			}

			return nil
		},
	}

	listVarsCmd.Flags().BoolP("secrets", "s", false, "Show decrypted secret values")
	listVarsCmd.Flags().StringP("key", "k", "", "Encryption key for secrets")
	listVarsCmd.Flags().StringP("file", "f", env.DefaultEnvFile, "Path to environment file")
	listVarsCmd.Flags().StringP("env", "e", "", "Target environment (defaults to current)")

	// Export to .env file command
	exportCmd := &cobra.Command{
		Use:   "export [output-file]",
		Short: "Export to .env file",
		Long:  "Export the current environment variables to a .env file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			envName, _ := cmd.Flags().GetString("env")

			// Determine output path
			outputPath := ".env"
			if len(args) > 0 {
				outputPath = args[0]
			}

			envFile, err := env.LoadEnvironmentFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to load environments: %v", err)
			}

			// Determine which environment to use
			var environment *env.Environment
			if envName != "" {
				environment, err = env.GetEnvironment(envFile, envName)
			} else {
				environment, err = env.GetCurrentEnvironment(envFile)
			}

			if err != nil {
				return fmt.Errorf("failed to get environment: %v", err)
			}

			// We need an encryption key if there are secrets
			if len(environment.Secrets) > 0 {
				encKey, _ := cmd.Flags().GetString("key")

				// If no key provided, prompt for it
				if encKey == "" {
					fmt.Print("Enter encryption key: ")
					byteKey, err := term.ReadPassword(int(syscall.Stdin))
					if err != nil {
						return fmt.Errorf("failed to read encryption key: %v", err)
					}
					fmt.Println() // Add newline after password input
					encKey = string(byteKey)
				}

				environment.SetEncryptionKey(encKey)
			}

			// Export the environment
			if err := env.ExportDotenv(environment, outputPath); err != nil {
				return fmt.Errorf("failed to export environment: %v", err)
			}

			fmt.Printf("Environment '%s' exported to %s\n", environment.Name, outputPath)
			return nil
		},
	}

	exportCmd.Flags().StringP("key", "k", "", "Encryption key for secrets")
	exportCmd.Flags().StringP("file", "f", env.DefaultEnvFile, "Path to environment file")
	exportCmd.Flags().StringP("env", "e", "", "Target environment (defaults to current)")

	// Import from .env file command
	importCmd := &cobra.Command{
		Use:   "import [input-file]",
		Short: "Import from .env file",
		Long:  "Import variables from a .env file into the current environment",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			envName, _ := cmd.Flags().GetString("env")
			asSecrets, _ := cmd.Flags().GetBool("as-secrets")

			// Determine input path
			inputPath := ".env"
			if len(args) > 0 {
				inputPath = args[0]
			}

			envFile, err := env.LoadEnvironmentFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to load environments: %v", err)
			}

			// Determine which environment to use
			var environment *env.Environment
			if envName != "" {
				environment, err = env.GetEnvironment(envFile, envName)
			} else {
				environment, err = env.GetCurrentEnvironment(envFile)
			}

			if err != nil {
				return fmt.Errorf("failed to get environment: %v", err)
			}

			// If importing as secrets, we need an encryption key
			if asSecrets {
				encKey, _ := cmd.Flags().GetString("key")

				// If no key provided, prompt for it
				if encKey == "" {
					fmt.Print("Enter encryption key: ")
					byteKey, err := term.ReadPassword(int(syscall.Stdin))
					if err != nil {
						return fmt.Errorf("failed to read encryption key: %v", err)
					}
					fmt.Println() // Add newline after password input
					encKey = string(byteKey)
				}

				environment.SetEncryptionKey(encKey)
			}

			// Import the environment
			if err := env.ImportDotenv(environment, inputPath, asSecrets); err != nil {
				return fmt.Errorf("failed to import environment: %v", err)
			}

			fmt.Printf("Variables from %s imported into environment '%s'\n", inputPath, environment.Name)
			return nil
		},
	}

	importCmd.Flags().BoolP("as-secrets", "s", false, "Import as encrypted secrets")
	importCmd.Flags().StringP("key", "k", "", "Encryption key for secrets")
	importCmd.Flags().StringP("file", "f", env.DefaultEnvFile, "Path to environment file")
	importCmd.Flags().StringP("env", "e", "", "Target environment (defaults to current)")

	// Add all commands to the env command
	envCmd.AddCommand(initCmd)
	envCmd.AddCommand(listCmd)
	envCmd.AddCommand(addEnvCmd)
	envCmd.AddCommand(useCmd)
	envCmd.AddCommand(removeCmd)
	envCmd.AddCommand(setCmd)
	envCmd.AddCommand(getCmd)
	envCmd.AddCommand(delCmd)
	envCmd.AddCommand(listVarsCmd)
	envCmd.AddCommand(exportCmd)
	envCmd.AddCommand(importCmd)

	// Add the env command to the root command
	rootCmd.AddCommand(envCmd)
}
