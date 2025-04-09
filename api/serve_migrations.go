package api

import (
	"fmt"
	"log"

	"github.com/zechtz/nyatictl/cli"
)

// EnsureDatabaseMigrated checks for and applies any pending migrations
// during server startup. This ensures the database schema is up to date.
//
// This function is called from NewServer() to ensure migrations are applied
// before the server is fully initialized.
//
// Returns:
//   - error: If applying migrations fails
func EnsureDatabaseMigrated() error {
	log.Println("Checking for pending database migrations...")

	// Run migrations using the CLI migration function
	if err := cli.RunMigrationsAPI(); err != nil {
		return fmt.Errorf("failed to apply migrations: %v", err)
	}

	log.Println("Database schema is up to date")
	return nil
}
