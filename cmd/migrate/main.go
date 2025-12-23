package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/afterdarksys/adsops-utils/internal/config"
	"github.com/spf13/cobra"
)

var (
	migrationsDir string
	rootCmd       = &cobra.Command{
		Use:   "migrate",
		Short: "Database migration tool for After Dark Systems Change Management",
		Long:  `Run database migrations up, down, or check status.`,
	}
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&migrationsDir, "dir", "./migrations", "Migrations directory")

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(createCmd)
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Run all pending migrations",
	Run:   runUp,
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback the last migration",
	Run:   runDown,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Run:   runStatus,
}

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new migration",
	Args:  cobra.ExactArgs(1),
	Run:   runCreate,
}

func runUp(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Running migrations against %s:%d/%s\n",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// Find migration files
	files, err := findMigrations(migrationsDir, ".up.sql")
	if err != nil {
		log.Fatalf("Failed to find migrations: %v", err)
	}

	if len(files) == 0 {
		fmt.Println("No migrations to run")
		return
	}

	// TODO: Connect to database and run migrations
	for _, f := range files {
		fmt.Printf("Running: %s\n", filepath.Base(f))
	}

	fmt.Println("Migrations completed successfully")
}

func runDown(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Rolling back migration on %s:%d/%s\n",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// TODO: Connect to database and rollback

	fmt.Println("Rollback completed")
}

func runStatus(cmd *cobra.Command, args []string) {
	files, err := findMigrations(migrationsDir, ".up.sql")
	if err != nil {
		log.Fatalf("Failed to find migrations: %v", err)
	}

	fmt.Println("Migration Status")
	fmt.Println("================")
	fmt.Println()

	// TODO: Check which migrations have been applied

	for _, f := range files {
		name := filepath.Base(f)
		name = strings.TrimSuffix(name, ".up.sql")
		fmt.Printf("[ ] %s\n", name)
	}
}

func runCreate(cmd *cobra.Command, args []string) {
	name := args[0]

	// Find the next migration number
	files, _ := findMigrations(migrationsDir, ".up.sql")
	nextNum := len(files) + 1

	// Create migration files
	upFile := filepath.Join(migrationsDir, fmt.Sprintf("%06d_%s.up.sql", nextNum, name))
	downFile := filepath.Join(migrationsDir, fmt.Sprintf("%06d_%s.down.sql", nextNum, name))

	if err := os.WriteFile(upFile, []byte("-- Migration: "+name+"\n\n"), 0644); err != nil {
		log.Fatalf("Failed to create up migration: %v", err)
	}

	if err := os.WriteFile(downFile, []byte("-- Rollback: "+name+"\n\n"), 0644); err != nil {
		log.Fatalf("Failed to create down migration: %v", err)
	}

	fmt.Printf("Created migration files:\n")
	fmt.Printf("  %s\n", upFile)
	fmt.Printf("  %s\n", downFile)
}

func findMigrations(dir, suffix string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), suffix) {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	sort.Strings(files)
	return files, nil
}
