package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/reignx/reignx/pkg/config"
)

func main() {
	configPath := flag.String("config", "config/apiserver.yaml", "Path to configuration file")
	migrationsPath := flag.String("migrations", "migrations", "Path to migrations directory")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: migrate [up|down|reset|version|force] [args]")
		fmt.Println("Commands:")
		fmt.Println("  up [N]        - Apply all or N up migrations")
		fmt.Println("  down [N]      - Apply all or N down migrations")
		fmt.Println("  reset         - Down all migrations then up all")
		fmt.Println("  version       - Print current migration version")
		fmt.Println("  force VERSION - Set migration version without running migrations")
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create migration instance
	sourceURL := fmt.Sprintf("file://%s", *migrationsPath)
	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
		cfg.Database.SSLMode,
	)

	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	command := flag.Arg(0)

	switch command {
	case "up":
		err = m.Up()
		if err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run up migrations: %v", err)
		}
		if err == migrate.ErrNoChange {
			fmt.Println("No migrations to apply")
		} else {
			fmt.Println("Migrations applied successfully")
		}

	case "down":
		err = m.Down()
		if err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run down migrations: %v", err)
		}
		if err == migrate.ErrNoChange {
			fmt.Println("No migrations to rollback")
		} else {
			fmt.Println("Migrations rolled back successfully")
		}

	case "reset":
		err = m.Down()
		if err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run down migrations: %v", err)
		}
		err = m.Up()
		if err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run up migrations: %v", err)
		}
		fmt.Println("Database reset successfully")

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Current version: %d\n", version)
		fmt.Printf("Dirty: %v\n", dirty)

	case "force":
		if flag.NArg() < 2 {
			log.Fatal("force command requires version argument")
		}
		var version int
		fmt.Sscanf(flag.Arg(1), "%d", &version)
		err = m.Force(version)
		if err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}
		fmt.Printf("Forced version to %d\n", version)

	default:
		log.Fatalf("Unknown command: %s", command)
	}
}
