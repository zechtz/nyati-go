package cli

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

const (
	dbPath        = "./nyatictl.db"
	migrationsDir = "./db/migrations"
)

// Migration represents a database migration file.
type Migration struct {
	Name string
	SQL  string
}

// setupMigrationCommands adds database migration commands to the provided root command.
// This is called from the Execute function in cli.go
func setupMigrationCommands(rootCmd *cobra.Command) {
	// Create the db command
	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
		Long:  "Commands for managing the NyatiCtl database schema",
	}

	// Add the migrate command
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		Long:  "Apply all pending database migrations in sequential order",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrations()
		},
	}

	// Add the generate command
	generateCmd := &cobra.Command{
		Use:   "generate [name]",
		Short: "Generate a new migration file",
		Long:  "Create a new timestamped SQL migration file in the db/migrations directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateMigration(args[0])
		},
	}

	// Add the rollback command
	rollbackCmd := &cobra.Command{
		Use:   "rollback [migration_name]",
		Short: "Rollback a migration",
		Long:  "Revert a specific migration or the most recent one if none specified",
		RunE: func(cmd *cobra.Command, args []string) error {
			// If migration name is provided, roll back that specific migration
			if len(args) > 0 {
				return rollbackMigration(args[0])
			}
			// Otherwise, roll back the most recent migration
			return rollbackLastMigration()
		},
	}

	// Add the status command to show applied/pending migrations
	statusCmd := &cobra.Command{
		Use:   "status [--verbose]",
		Short: "Show migration status",
		Long: `Display a list of applied and pending migrations.
	
Use the --verbose flag to show SQL snippets of the UP and DOWN sections.

Examples:
  nyatictl db status            # Show basic migration status
  nyatictl db status --verbose  # Show status with SQL snippets`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showMigrationStatus()
		},
	}

	// Add commands to the db command
	dbCmd.AddCommand(migrateCmd)
	dbCmd.AddCommand(generateCmd)
	dbCmd.AddCommand(rollbackCmd)
	dbCmd.AddCommand(statusCmd)

	// Add the db command to the root command
	rootCmd.AddCommand(dbCmd)
}

// runMigrations runs all pending database migrations.
//
// It reads migration files from the migrations directory,
// tracks applied migrations in a migrations table,
// and executes pending migrations in order.
//
// Returns:
//   - error: If any migration fails
func runMigrations() error {
	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return fmt.Errorf("failed to enable foreign keys: %v", err)
	}

	// Ensure migrations table exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Get applied migrations
	rows, err := db.Query("SELECT name FROM migrations")
	if err != nil {
		return fmt.Errorf("failed to query migrations: %v", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("failed to scan migration: %v", err)
		}
		applied[name] = true
	}

	// Read migration files
	migrations, err := readMigrations()
	if err != nil {
		return fmt.Errorf("failed to read migrations: %v", err)
	}

	// Sort migrations by name (which includes timestamp)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	// Track whether any migrations were applied
	migrationsApplied := false

	// Apply pending migrations
	for _, migration := range migrations {
		if !applied[migration.Name] {
			// Validate the migration
			valid, errMsg := validateMigration(migration.SQL)
			if !valid {
				fmt.Printf("Skipping invalid migration %s: %s\n", migration.Name, errMsg)
				continue
			}

			fmt.Printf("Applying migration: %s\n", migration.Name)

			// Extract UP section
			upSQL := extractUPSection(migration.SQL)

			// Begin transaction
			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin transaction: %v", err)
			}

			// Execute each statement in the UP section
			statements := splitStatements(upSQL)
			for _, stmt := range statements {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}

				if _, err := tx.Exec(stmt); err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to apply migration %s: %v\nStatement: %s",
						migration.Name, err, stmt)
				}
			}

			// Record the migration as applied
			if _, err := tx.Exec(
				"INSERT INTO migrations (name) VALUES (?)",
				migration.Name); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to record migration %s: %v", migration.Name, err)
			}

			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit migration %s: %v", migration.Name, err)
			}

			fmt.Printf("Successfully applied migration: %s\n", migration.Name)
			migrationsApplied = true
		}
	}

	if migrationsApplied {
		fmt.Println("All migrations have been applied successfully")
	} else {
		fmt.Println("Database schema is already up to date")
	}

	return nil
}

// generateMigration creates a new migration file with the given name.
//
// Parameters:
//   - name: The descriptive name for the migration (will be prefixed with timestamp)
//
// Returns:
//   - error: If file creation fails
func generateMigration(name string) error {
	// Sanitize the name (replace spaces with underscores)
	sanitizedName := strings.ReplaceAll(name, " ", "_")
	sanitizedName = strings.ToLower(sanitizedName)

	// Create timestamp
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", timestamp, sanitizedName)
	path := filepath.Join(migrationsDir, filename)

	// Create migration content template with clear sections
	content := `-- UP
-- Write your SQL statements to apply the migration here
-- For example:
-- ALTER TABLE users ADD COLUMN email TEXT;
-- CREATE INDEX idx_users_email ON users(email);


-- DOWN
-- Write your SQL statements to revert the migration here
-- These statements will be executed when rolling back the migration
-- For example:
-- DROP INDEX IF EXISTS idx_users_email;
-- ALTER TABLE users DROP COLUMN email;
`

	// Ensure migrations directory exists
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %v", err)
	}

	// Write migration file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %v", err)
	}

	fmt.Printf("Created migration file: %s\n", path)
	fmt.Println("Edit this file to add your schema changes, then run 'nyatictl db migrate' to apply it.")
	return nil
}

// readMigrations reads all SQL migration files from the migrations directory.
//
// Returns:
//   - []Migration: List of migrations
//   - error: If directory reading fails
func readMigrations() ([]Migration, error) {
	var migrations []Migration

	// Check if migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		fmt.Printf("Migrations directory '%s' does not exist. Creating it...\n", migrationsDir)
		if err := os.MkdirAll(migrationsDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create migrations directory: %v", err)
		}
		return migrations, nil // Return empty list (no migrations yet)
	}

	err := filepath.WalkDir(migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), ".sql") {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read migration file %s: %v", path, err)
			}

			migrations = append(migrations, Migration{
				Name: d.Name(),
				SQL:  string(content),
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return migrations, nil
}

// extractUPSection extracts the SQL statements from the UP section of a migration.
//
// Parameters:
//   - sql: The complete SQL content of a migration file
//
// Returns:
//   - string: The UP section content
func extractUPSection(sql string) string {
	parts := strings.Split(sql, "-- DOWN")
	if len(parts) == 0 {
		return ""
	}

	upParts := strings.Split(parts[0], "-- UP")
	if len(upParts) < 2 {
		return parts[0] // No UP marker, assume everything is UP
	}

	return strings.TrimLeftFunc(upParts[1], unicode.IsSpace)
}

// extractDOWNSection extracts the SQL statements from the DOWN section of a migration.
//
// Parameters:
//   - sql: The complete SQL content of a migration file
//
// Returns:
//   - string: The DOWN section content
func extractDOWNSection(sql string) string {
	parts := strings.Split(sql, "-- DOWN")
	if len(parts) < 2 {
		return "" // No DOWN section found
	}

	return strings.TrimLeftFunc(parts[1], unicode.IsSpace)
}

// splitStatements splits a SQL string into individual statements by semicolons.
// This improved version handles multi-line statements and ignores semicolons in comments.
//
// Parameters:
//   - sql: SQL content to split
//
// Returns:
//   - []string: List of SQL statements
func splitStatements(sql string) []string {
	var statements []string
	var currentStmt strings.Builder
	lines := strings.Split(sql, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(trimmed, "--") || trimmed == "" {
			continue
		}

		currentStmt.WriteString(line)
		currentStmt.WriteString("\n")

		// If the line contains a semicolon, it might be the end of a statement
		if strings.Contains(line, ";") {
			stmt := currentStmt.String()
			statements = append(statements, stmt)
			currentStmt.Reset()
		}
	}

	// Don't forget any trailing statements without semicolons
	final := currentStmt.String()
	if strings.TrimSpace(final) != "" {
		statements = append(statements, final)
	}

	return statements
}

// prettyPrintSQL formats SQL statements for better readability.
// It removes excessive whitespace, preserves indentation, and
// formats the SQL to be more compact for display purposes.
//
// Parameters:
//   - sql: SQL content to format
//
// Returns:
//   - string: Formatted SQL
func prettyPrintSQL(sql string) string {
	if sql == "" {
		return ""
	}

	lines := strings.Split(sql, "\n")
	var result []string

	// Remove empty lines at the start
	startIdx := 0
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			startIdx = i
			break
		}
	}

	// Process each non-empty line
	for _, line := range lines[startIdx:] {
		// Skip empty lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Keep comments as-is
		if strings.HasPrefix(trimmed, "--") {
			result = append(result, trimmed)
			continue
		}

		// For SQL statements, preserve basic formatting
		indentLevel := countLeadingSpaces(line) / 2
		indent := strings.Repeat("  ", indentLevel) // 2 spaces per level

		// Add formatted line
		result = append(result, indent+trimmed)
	}

	// Join the result with newlines
	return strings.Join(result, "\n")
}

// countLeadingSpaces counts the number of spaces at the beginning of a string
func countLeadingSpaces(s string) int {
	count := 0
	for _, char := range s {
		if char == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}

// validateMigration checks if a migration file has valid UP/DOWN sections.
//
// Parameters:
//   - sql: The complete SQL content of a migration file
//
// Returns:
//   - bool: True if the migration is valid
//   - string: Error message if invalid
func validateMigration(sql string) (bool, string) {
	if !strings.Contains(sql, "-- UP") {
		return false, "Migration must contain '-- UP' section"
	}

	upSQL := extractUPSection(sql)
	if strings.TrimSpace(upSQL) == "" {
		return false, "UP section cannot be empty"
	}

	return true, ""
}

// RunMigrationsAPI provides a programmatic way to run migrations
// This can be called from other parts of the application (like server startup)
func RunMigrationsAPI() error {
	return runMigrations()
}

// rollbackMigration rolls back a specific migration.
//
// Parameters:
//   - migrationName: The name of the migration to roll back
//
// Returns:
//   - error: If rollback fails
func rollbackMigration(migrationName string) error {
	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Check if the migration exists and has been applied
	var exists bool
	err = db.QueryRow("SELECT 1 FROM migrations WHERE name = ?", migrationName).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("migration '%s' has not been applied or does not exist", migrationName)
		}
		return fmt.Errorf("failed to check migration status: %v", err)
	}

	// Read the migration file to get the DOWN section
	migrations, err := readMigrations()
	if err != nil {
		return fmt.Errorf("failed to read migrations: %v", err)
	}

	// Find the migration in the list
	var migrationSQL string
	for _, migration := range migrations {
		if migration.Name == migrationName {
			migrationSQL = migration.SQL
			break
		}
	}

	if migrationSQL == "" {
		return fmt.Errorf("migration file '%s' not found", migrationName)
	}

	// Extract the DOWN section
	downSQL := extractDOWNSection(migrationSQL)
	if downSQL == "" {
		return fmt.Errorf("no DOWN section found in migration '%s'", migrationName)
	}

	fmt.Printf("Rolling back migration: %s\n", migrationName)

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Execute each statement in the DOWN section
	statements := splitStatements(downSQL)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply rollback statement: %v\nStatement: %s", err, stmt)
		}
	}

	// Remove the migration from the migrations table
	if _, err := tx.Exec("DELETE FROM migrations WHERE name = ?", migrationName); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update migrations table: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("Successfully rolled back migration: %s\n", migrationName)
	return nil
}

// rollbackLastMigration rolls back the most recently applied migration.
//
// Returns:
//   - error: If rollback fails
func rollbackLastMigration() error {
	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Get the most recently applied migration
	var migrationName string
	err = db.QueryRow("SELECT name FROM migrations ORDER BY id DESC LIMIT 1").Scan(&migrationName)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no migrations have been applied yet")
		}
		return fmt.Errorf("failed to get the most recent migration: %v", err)
	}

	// Roll back the migration
	return rollbackMigration(migrationName)
}

// showMigrationStatus displays the status of all migrations with SQL snippets.
//
// Returns:
//   - error: If checking status fails
func showMigrationStatus() error {
	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure migrations table exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Get applied migrations
	rows, err := db.Query("SELECT name, applied_at FROM migrations ORDER BY id")
	if err != nil {
		return fmt.Errorf("failed to query migrations: %v", err)
	}
	defer rows.Close()

	applied := make(map[string]string) // name -> applied_at
	for rows.Next() {
		var name, appliedAt string
		if err := rows.Scan(&name, &appliedAt); err != nil {
			return fmt.Errorf("failed to scan migration: %v", err)
		}
		applied[name] = appliedAt
	}

	// Read migration files
	migrations, err := readMigrations()
	if err != nil {
		return fmt.Errorf("failed to read migrations: %v", err)
	}

	// Sort migrations by name
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	// Display status
	fmt.Println("===== Migration Status =====")

	if len(migrations) == 0 {
		fmt.Println("No migrations found.")
		return nil
	}

	// Get terminal width for formatting
	termWidth := 80 // default width

	// Check for flag to show SQL content
	detailedView := len(os.Args) > 3 && os.Args[3] == "--verbose"

	// Print header
	fmt.Printf("%-40s %-10s %s\n", "Migration", "Status", "Applied At")
	fmt.Printf("%-40s %-10s %s\n", strings.Repeat("-", 40), strings.Repeat("-", 10), strings.Repeat("-", 19))

	// Print migration status
	for _, migration := range migrations {
		appliedAt, isApplied := applied[migration.Name]
		status := "PENDING"
		if isApplied {
			status = "APPLIED"
		} else {
			appliedAt = "N/A"
		}
		fmt.Printf("%-40s %-10s %s\n", migration.Name, status, appliedAt)

		// Show SQL snippets in detailed view mode
		if detailedView {
			fmt.Println()
			upSQL := extractUPSection(migration.SQL)
			downSQL := extractDOWNSection(migration.SQL)

			// Display UP section (first few lines)
			upLines := strings.Split(prettyPrintSQL(upSQL), "\n")
			if len(upLines) > 0 {
				fmt.Println("  UP:")
				displayLines := min(len(upLines), 3) // Show at most 3 lines
				for i := 0; i < displayLines; i++ {
					if len(upLines[i]) > termWidth-6 {
						// Truncate long lines
						fmt.Printf("    %s...\n", upLines[i][:termWidth-10])
					} else {
						fmt.Printf("    %s\n", upLines[i])
					}
				}
				if len(upLines) > 3 {
					fmt.Println("    ...")
				}
			}

			// Display DOWN section (first few lines)
			if downSQL != "" {
				downLines := strings.Split(prettyPrintSQL(downSQL), "\n")
				if len(downLines) > 0 {
					fmt.Println("  DOWN:")
					displayLines := min(len(downLines), 3) // Show at most 3 lines
					for i := range displayLines {
						if len(downLines[i]) > termWidth-6 {
							// Truncate long lines
							fmt.Printf("    %s...\n", downLines[i][:termWidth-10])
						} else {
							fmt.Printf("    %s\n", downLines[i])
						}
					}
					if len(downLines) > 3 {
						fmt.Println("    ...")
					}
				}
			}
			fmt.Println(strings.Repeat("-", termWidth))
		}
	}

	fmt.Println()
	fmt.Println("Tip: Use 'nyatictl db status --verbose' to see SQL snippets")
	fmt.Println("     Use 'nyatictl db migrate' to apply pending migrations")
	fmt.Println("     Use 'nyatictl db rollback' to revert the most recent migration")

	return nil
}

// Helper function min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
