package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const (
	// Default database connection
	defaultDBHost     = "localhost"
	defaultDBPort     = "5432"
	defaultDBName     = "apiproxy"
	defaultDBUser     = "apiproxy"
	defaultDBPassword = "apiproxy_secure_2026"

	// Blackout JSON export path
	blackoutJSONPath = "/var/lib/adsops/active-blackouts.json"

	// Version
	version = "1.0.0"
)

// Blackout represents a maintenance/blackout record
type Blackout struct {
	ID              int       `json:"id"`
	TicketNumber    string    `json:"ticket_number"`
	Hostname        string    `json:"hostname"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	ActualEndTime   *time.Time `json:"actual_end_time,omitempty"`
	Reason          string    `json:"reason"`
	CreatedBy       string    `json:"created_by"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

// ActiveBlackoutExport represents the JSON format for monitoring integration
type ActiveBlackoutExport struct {
	Hostname   string `json:"hostname"`
	Ticket     string `json:"ticket"`
	EndTime    string `json:"end_time"` // ISO8601 format
	Reason     string `json:"reason"`
	RemainingTime string `json:"remaining_time,omitempty"`
}

// DB manages database connections
type DB struct {
	conn *sql.DB
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Parse command
	command := os.Args[1]

	// Handle commands that don't need database connection
	if command == "version" || command == "--version" || command == "-v" {
		fmt.Printf("blackout version %s\n", version)
		return
	}

	if command == "help" || command == "--help" || command == "-h" {
		printUsage()
		return
	}

	// Get database connection for all other commands
	db, err := connectDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please ensure PostgreSQL is running and credentials are correct.\n")
		fmt.Fprintf(os.Stderr, "Set DB_* environment variables if needed (DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD)\n")
		os.Exit(1)
	}
	defer db.conn.Close()

	// Ensure schema exists
	if err := db.ensureSchema(); err != nil {
		fmt.Fprintf(os.Stderr, "Error ensuring schema: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "start":
		if len(os.Args) < 6 {
			fmt.Fprintf(os.Stderr, "Usage: blackout start <ticket> <hostname> <duration> \"reason\"\n")
			os.Exit(1)
		}
		handleStart(db, os.Args[2], os.Args[3], os.Args[4], os.Args[5])

	case "end":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: blackout end <hostname>\n")
			os.Exit(1)
		}
		handleEnd(db, os.Args[2])

	case "list":
		activeOnly := false
		hostname := ""
		for i := 2; i < len(os.Args); i++ {
			if os.Args[i] == "--active" {
				activeOnly = true
			} else if os.Args[i] == "--hostname" && i+1 < len(os.Args) {
				hostname = os.Args[i+1]
				i++
			}
		}
		handleList(db, activeOnly, hostname)

	case "show":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: blackout show <hostname>\n")
			os.Exit(1)
		}
		handleShow(db, os.Args[2])

	case "extend":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Usage: blackout extend <hostname> <additional_duration>\n")
			os.Exit(1)
		}
		handleExtend(db, os.Args[2], os.Args[3])

	case "export":
		// Manual export trigger
		if err := db.exportActiveBlackouts(); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting blackouts: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Active blackouts exported to %s\n", blackoutJSONPath)

	case "cleanup":
		// Expire old blackouts and clean up
		handleCleanup(db)

	default:
		// Default: assume shorthand format: blackout <ticket> <hostname> <duration> "reason"
		if len(os.Args) >= 5 {
			handleStart(db, os.Args[1], os.Args[2], os.Args[3], os.Args[4])
		} else {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
			printUsage()
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Printf(`blackout - Maintenance/Blackout Mode Management Tool v%s

USAGE:
    blackout <ticket> <hostname> <duration> "reason"
    blackout start <ticket> <hostname> <duration> "reason"
    blackout end <hostname>
    blackout list [--active] [--hostname <hostname>]
    blackout show <hostname>
    blackout extend <hostname> <additional_duration>
    blackout export
    blackout cleanup
    blackout version
    blackout help

EXAMPLES:
    # Start a 2.5 hour blackout
    blackout CHG-2024-001 api-server-1 2:30 "Database migration"

    # Start using explicit command
    blackout start CHG-2024-001 api-server-1 1:00 "Security patching"

    # End blackout early
    blackout end api-server-1

    # List all active blackouts
    blackout list --active

    # Show details for a host
    blackout show api-server-1

    # Extend blackout by 30 minutes
    blackout extend api-server-1 0:30

    # Export active blackouts to JSON (for monitoring)
    blackout export

    # Clean up expired blackouts
    blackout cleanup

DURATION FORMAT:
    H:MM    - Hours:Minutes (e.g., 2:30 for 2.5 hours)
    H       - Hours only (e.g., 2 for 2 hours)
    MMm     - Minutes only (e.g., 90m for 90 minutes)

ENVIRONMENT VARIABLES:
    DB_HOST       - Database host (default: localhost)
    DB_PORT       - Database port (default: 5432)
    DB_NAME       - Database name (default: apiproxy)
    DB_USER       - Database user (default: apiproxy)
    DB_PASSWORD   - Database password (default: apiproxy_secure_2026)

MONITORING INTEGRATION:
    Active blackouts are automatically exported to:
    %s

    This file is read by oci-observability for alert suppression.

FEATURES:
    - UTC timezone support
    - Auto-expiration checking
    - Audit trail with ticket numbers
    - Alert suppression integration
    - Host status management

`, version, blackoutJSONPath)
}

func connectDB() (*DB, error) {
	host := getEnv("DB_HOST", defaultDBHost)
	port := getEnv("DB_PORT", defaultDBPort)
	dbname := getEnv("DB_NAME", defaultDBName)
	user := getEnv("DB_USER", defaultDBUser)
	password := getEnv("DB_PASSWORD", defaultDBPassword)

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func (db *DB) ensureSchema() error {
	// Create inventory_resources table if not exists
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS inventory_resources (
			id SERIAL PRIMARY KEY,
			hostname VARCHAR(255) UNIQUE NOT NULL,
			status VARCHAR(50) DEFAULT 'active',
			resource_type VARCHAR(100),
			environment VARCHAR(50),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create inventory_resources table: %w", err)
	}

	// Create inventory_blackouts table if not exists
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS inventory_blackouts (
			id SERIAL PRIMARY KEY,
			ticket_number VARCHAR(50) NOT NULL,
			hostname VARCHAR(255) NOT NULL,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP NOT NULL,
			actual_end_time TIMESTAMP,
			reason TEXT,
			created_by VARCHAR(255),
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create inventory_blackouts table: %w", err)
	}

	// Create indices for performance
	db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_blackouts_hostname ON inventory_blackouts(hostname)`)
	db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_blackouts_status ON inventory_blackouts(status)`)
	db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_blackouts_end_time ON inventory_blackouts(end_time)`)
	db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_resources_hostname ON inventory_resources(hostname)`)
	db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_resources_status ON inventory_resources(status)`)

	return nil
}

func (db *DB) exportActiveBlackouts() error {
	// Query active blackouts
	rows, err := db.conn.Query(`
		SELECT hostname, ticket_number, end_time, reason
		FROM inventory_blackouts
		WHERE status = 'active' AND end_time > NOW()
		ORDER BY end_time ASC
	`)
	if err != nil {
		return fmt.Errorf("failed to query active blackouts: %w", err)
	}
	defer rows.Close()

	var exports []ActiveBlackoutExport
	now := time.Now().UTC()

	for rows.Next() {
		var hostname, ticket, reason string
		var endTime time.Time

		if err := rows.Scan(&hostname, &ticket, &endTime, &reason); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		remaining := endTime.Sub(now)
		remainingStr := formatDuration(remaining)

		exports = append(exports, ActiveBlackoutExport{
			Hostname:      hostname,
			Ticket:        ticket,
			EndTime:       endTime.Format(time.RFC3339),
			Reason:        reason,
			RemainingTime: remainingStr,
		})
	}

	// Create directory if needed
	dir := filepath.Dir(blackoutJSONPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write JSON file
	data, err := json.MarshalIndent(exports, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(blackoutJSONPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

func handleStart(db *DB, ticket, hostname, durationStr, reason string) {
	// Parse duration
	duration, err := parseDuration(durationStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing duration '%s': %v\n", durationStr, err)
		fmt.Fprintf(os.Stderr, "Valid formats: 2:30 (2.5 hours), 1:00 (1 hour), 90m (90 minutes)\n")
		os.Exit(1)
	}

	// Get current user
	currentUser := getCurrentUser()

	// Calculate times (UTC)
	startTime := time.Now().UTC()
	endTime := startTime.Add(duration)

	// Check if host already in active blackout
	var existingID int
	err = db.conn.QueryRow(`
		SELECT id FROM inventory_blackouts
		WHERE hostname = $1 AND status = 'active' AND end_time > NOW()
	`, hostname).Scan(&existingID)

	if err == nil {
		fmt.Fprintf(os.Stderr, "⚠️  Host %s is already in an active blackout (ID: %d)\n", hostname, existingID)
		fmt.Fprintf(os.Stderr, "Use 'blackout end %s' first, or 'blackout extend %s <duration>' to extend\n", hostname, hostname)
		os.Exit(1)
	}

	// Start transaction
	tx, err := db.conn.Begin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting transaction: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback()

	// Insert blackout record
	var blackoutID int
	err = tx.QueryRow(`
		INSERT INTO inventory_blackouts (
			ticket_number, hostname, start_time, end_time, reason, created_by, status
		) VALUES ($1, $2, $3, $4, $5, $6, 'active')
		RETURNING id
	`, ticket, hostname, startTime, endTime, reason, currentUser).Scan(&blackoutID)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating blackout: %v\n", err)
		os.Exit(1)
	}

	// Ensure host exists in inventory_resources
	_, err = tx.Exec(`
		INSERT INTO inventory_resources (hostname, status, updated_at)
		VALUES ($1, 'blackout', NOW())
		ON CONFLICT (hostname) DO UPDATE
		SET status = 'blackout', updated_at = NOW()
	`, hostname)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating host status: %v\n", err)
		os.Exit(1)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		fmt.Fprintf(os.Stderr, "Error committing transaction: %v\n", err)
		os.Exit(1)
	}

	// Export active blackouts
	if err := db.exportActiveBlackouts(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to export blackouts: %v\n", err)
	}

	// Print success
	fmt.Printf("✅ Blackout started for %s\n", hostname)
	fmt.Printf("   Ticket:     %s\n", ticket)
	fmt.Printf("   Start:      %s\n", startTime.Format(time.RFC3339))
	fmt.Printf("   End:        %s\n", endTime.Format(time.RFC3339))
	fmt.Printf("   Duration:   %s\n", formatDuration(duration))
	fmt.Printf("   Reason:     %s\n", reason)
	fmt.Printf("   Created by: %s\n", currentUser)
	fmt.Printf("   ID:         %d\n", blackoutID)
	fmt.Printf("\n")
	fmt.Printf("🔕 Alerts suppressed for %s until %s\n", hostname, endTime.Format("2006-01-02 15:04 MST"))
}

func handleEnd(db *DB, hostname string) {
	// Find active blackout
	var blackoutID int
	var ticket string
	var startTime, endTime time.Time

	err := db.conn.QueryRow(`
		SELECT id, ticket_number, start_time, end_time
		FROM inventory_blackouts
		WHERE hostname = $1 AND status = 'active' AND end_time > NOW()
		ORDER BY start_time DESC
		LIMIT 1
	`, hostname).Scan(&blackoutID, &ticket, &startTime, &endTime)

	if err == sql.ErrNoRows {
		fmt.Fprintf(os.Stderr, "⚠️  No active blackout found for %s\n", hostname)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying blackout: %v\n", err)
		os.Exit(1)
	}

	now := time.Now().UTC()

	// Start transaction
	tx, err := db.conn.Begin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting transaction: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback()

	// Update blackout status
	_, err = tx.Exec(`
		UPDATE inventory_blackouts
		SET status = 'completed', actual_end_time = $1
		WHERE id = $2
	`, now, blackoutID)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating blackout: %v\n", err)
		os.Exit(1)
	}

	// Restore host status to active
	_, err = tx.Exec(`
		UPDATE inventory_resources
		SET status = 'active', updated_at = NOW()
		WHERE hostname = $1
	`, hostname)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating host status: %v\n", err)
		os.Exit(1)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		fmt.Fprintf(os.Stderr, "Error committing transaction: %v\n", err)
		os.Exit(1)
	}

	// Export active blackouts
	if err := db.exportActiveBlackouts(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to export blackouts: %v\n", err)
	}

	duration := now.Sub(startTime)
	scheduledDuration := endTime.Sub(startTime)

	fmt.Printf("✅ Blackout ended for %s\n", hostname)
	fmt.Printf("   Ticket:            %s\n", ticket)
	fmt.Printf("   Actual duration:   %s\n", formatDuration(duration))
	fmt.Printf("   Scheduled:         %s\n", formatDuration(scheduledDuration))
	if now.Before(endTime) {
		early := endTime.Sub(now)
		fmt.Printf("   Ended early by:    %s\n", formatDuration(early))
	} else {
		overrun := now.Sub(endTime)
		fmt.Printf("   ⚠️  Overran by:      %s\n", formatDuration(overrun))
	}
	fmt.Printf("\n")
	fmt.Printf("🔔 Alerts re-enabled for %s\n", hostname)
}

func handleList(db *DB, activeOnly bool, hostname string) {
	query := `
		SELECT id, ticket_number, hostname, start_time, end_time,
		       actual_end_time, reason, created_by, status, created_at
		FROM inventory_blackouts
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	if activeOnly {
		query += fmt.Sprintf(" AND status = 'active' AND end_time > NOW()")
	}

	if hostname != "" {
		query += fmt.Sprintf(" AND hostname = $%d", argCount)
		args = append(args, hostname)
		argCount++
	}

	query += " ORDER BY start_time DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying blackouts: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	blackouts := []Blackout{}
	for rows.Next() {
		var b Blackout
		err := rows.Scan(&b.ID, &b.TicketNumber, &b.Hostname, &b.StartTime,
			&b.EndTime, &b.ActualEndTime, &b.Reason, &b.CreatedBy,
			&b.Status, &b.CreatedAt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning row: %v\n", err)
			continue
		}
		blackouts = append(blackouts, b)
	}

	if len(blackouts) == 0 {
		if activeOnly {
			fmt.Println("No active blackouts found")
		} else {
			fmt.Println("No blackouts found")
		}
		return
	}

	// Print header
	fmt.Printf("\n%-6s %-15s %-20s %-20s %-20s %-10s\n",
		"ID", "TICKET", "HOSTNAME", "START", "END", "STATUS")
	fmt.Println(strings.Repeat("-", 100))

	now := time.Now().UTC()

	for _, b := range blackouts {
		startStr := b.StartTime.Format("2006-01-02 15:04")
		endStr := b.EndTime.Format("2006-01-02 15:04")

		// Add indicator for active blackouts
		statusDisplay := b.Status
		if b.Status == "active" && b.EndTime.After(now) {
			remaining := b.EndTime.Sub(now)
			statusDisplay = fmt.Sprintf("active (%s left)", formatDuration(remaining))
		}

		fmt.Printf("%-6d %-15s %-20s %-20s %-20s %-10s\n",
			b.ID, b.TicketNumber, b.Hostname, startStr, endStr, statusDisplay)

		if b.Reason != "" {
			fmt.Printf("       Reason: %s\n", b.Reason)
		}
	}

	fmt.Printf("\nTotal: %d blackout(s)\n", len(blackouts))
}

func handleShow(db *DB, hostname string) {
	// Get current/most recent blackout
	var b Blackout
	err := db.conn.QueryRow(`
		SELECT id, ticket_number, hostname, start_time, end_time,
		       actual_end_time, reason, created_by, status, created_at
		FROM inventory_blackouts
		WHERE hostname = $1
		ORDER BY start_time DESC
		LIMIT 1
	`, hostname).Scan(&b.ID, &b.TicketNumber, &b.Hostname, &b.StartTime,
		&b.EndTime, &b.ActualEndTime, &b.Reason, &b.CreatedBy,
		&b.Status, &b.CreatedAt)

	if err == sql.ErrNoRows {
		fmt.Printf("No blackout history found for %s\n", hostname)
		os.Exit(0)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying blackout: %v\n", err)
		os.Exit(1)
	}

	now := time.Now().UTC()

	// Print details
	fmt.Printf("\n=== Blackout Details for %s ===\n\n", hostname)
	fmt.Printf("ID:             %d\n", b.ID)
	fmt.Printf("Ticket:         %s\n", b.TicketNumber)
	fmt.Printf("Hostname:       %s\n", b.Hostname)
	fmt.Printf("Status:         %s\n", b.Status)
	fmt.Printf("\n")
	fmt.Printf("Start Time:     %s\n", b.StartTime.Format(time.RFC3339))
	fmt.Printf("End Time:       %s\n", b.EndTime.Format(time.RFC3339))

	if b.ActualEndTime != nil {
		fmt.Printf("Actual End:     %s\n", b.ActualEndTime.Format(time.RFC3339))
	}

	scheduledDuration := b.EndTime.Sub(b.StartTime)
	fmt.Printf("Duration:       %s\n", formatDuration(scheduledDuration))

	if b.Status == "active" {
		if b.EndTime.After(now) {
			remaining := b.EndTime.Sub(now)
			elapsed := now.Sub(b.StartTime)
			fmt.Printf("\n")
			fmt.Printf("⏱️  Elapsed:       %s\n", formatDuration(elapsed))
			fmt.Printf("⏱️  Remaining:     %s\n", formatDuration(remaining))
		} else {
			overrun := now.Sub(b.EndTime)
			fmt.Printf("\n")
			fmt.Printf("⚠️  OVERRUN:       %s past scheduled end\n", formatDuration(overrun))
		}
	} else if b.ActualEndTime != nil {
		actualDuration := b.ActualEndTime.Sub(b.StartTime)
		fmt.Printf("\n")
		fmt.Printf("Actual Duration: %s\n", formatDuration(actualDuration))
		if b.ActualEndTime.Before(b.EndTime) {
			early := b.EndTime.Sub(*b.ActualEndTime)
			fmt.Printf("Ended Early:     %s\n", formatDuration(early))
		} else if b.ActualEndTime.After(b.EndTime) {
			overrun := b.ActualEndTime.Sub(b.EndTime)
			fmt.Printf("Overrun:         %s\n", formatDuration(overrun))
		}
	}

	fmt.Printf("\n")
	fmt.Printf("Reason:         %s\n", b.Reason)
	fmt.Printf("Created By:     %s\n", b.CreatedBy)
	fmt.Printf("Created At:     %s\n", b.CreatedAt.Format(time.RFC3339))
	fmt.Printf("\n")
}

func handleExtend(db *DB, hostname, durationStr string) {
	// Parse additional duration
	additionalDuration, err := parseDuration(durationStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing duration '%s': %v\n", durationStr, err)
		os.Exit(1)
	}

	// Find active blackout
	var blackoutID int
	var ticket string
	var currentEndTime time.Time

	err = db.conn.QueryRow(`
		SELECT id, ticket_number, end_time
		FROM inventory_blackouts
		WHERE hostname = $1 AND status = 'active' AND end_time > NOW()
		ORDER BY start_time DESC
		LIMIT 1
	`, hostname).Scan(&blackoutID, &ticket, &currentEndTime)

	if err == sql.ErrNoRows {
		fmt.Fprintf(os.Stderr, "⚠️  No active blackout found for %s\n", hostname)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying blackout: %v\n", err)
		os.Exit(1)
	}

	newEndTime := currentEndTime.Add(additionalDuration)

	// Update end time
	_, err = db.conn.Exec(`
		UPDATE inventory_blackouts
		SET end_time = $1
		WHERE id = $2
	`, newEndTime, blackoutID)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extending blackout: %v\n", err)
		os.Exit(1)
	}

	// Export active blackouts
	if err := db.exportActiveBlackouts(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to export blackouts: %v\n", err)
	}

	fmt.Printf("✅ Blackout extended for %s\n", hostname)
	fmt.Printf("   Ticket:          %s\n", ticket)
	fmt.Printf("   Previous End:    %s\n", currentEndTime.Format(time.RFC3339))
	fmt.Printf("   New End:         %s\n", newEndTime.Format(time.RFC3339))
	fmt.Printf("   Extended By:     %s\n", formatDuration(additionalDuration))
	fmt.Printf("\n")
	fmt.Printf("🔕 Alerts suppressed until %s\n", newEndTime.Format("2006-01-02 15:04 MST"))
}

func handleCleanup(db *DB) {
	// Auto-expire blackouts that have passed their end time
	result, err := db.conn.Exec(`
		UPDATE inventory_blackouts
		SET status = 'expired'
		WHERE status = 'active' AND end_time < NOW()
	`)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error expiring blackouts: %v\n", err)
		os.Exit(1)
	}

	expiredCount, _ := result.RowsAffected()

	// Restore hosts to active if they have no other active blackouts
	result, err = db.conn.Exec(`
		UPDATE inventory_resources
		SET status = 'active', updated_at = NOW()
		WHERE status = 'blackout'
		AND hostname NOT IN (
			SELECT hostname FROM inventory_blackouts
			WHERE status = 'active' AND end_time > NOW()
		)
	`)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error restoring host status: %v\n", err)
		os.Exit(1)
	}

	restoredCount, _ := result.RowsAffected()

	// Export active blackouts
	if err := db.exportActiveBlackouts(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to export blackouts: %v\n", err)
	}

	fmt.Printf("✅ Cleanup completed\n")
	fmt.Printf("   Expired blackouts:  %d\n", expiredCount)
	fmt.Printf("   Hosts restored:     %d\n", restoredCount)
}

// Helper functions

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	// Format: H:MM (e.g., 2:30)
	if strings.Contains(s, ":") {
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid duration format")
		}

		var hours, minutes int
		if _, err := fmt.Sscanf(parts[0], "%d", &hours); err != nil {
			return 0, fmt.Errorf("invalid hours")
		}
		if _, err := fmt.Sscanf(parts[1], "%d", &minutes); err != nil {
			return 0, fmt.Errorf("invalid minutes")
		}

		return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute, nil
	}

	// Format: Hh or just H (hours)
	if strings.HasSuffix(s, "h") || !strings.HasSuffix(s, "m") {
		s = strings.TrimSuffix(s, "h")
		var hours int
		if _, err := fmt.Sscanf(s, "%d", &hours); err != nil {
			return 0, fmt.Errorf("invalid hours")
		}
		return time.Duration(hours) * time.Hour, nil
	}

	// Format: MMm (minutes)
	if strings.HasSuffix(s, "m") {
		s = strings.TrimSuffix(s, "m")
		var minutes int
		if _, err := fmt.Sscanf(s, "%d", &minutes); err != nil {
			return 0, fmt.Errorf("invalid minutes")
		}
		return time.Duration(minutes) * time.Minute, nil
	}

	return 0, fmt.Errorf("invalid duration format")
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return fmt.Sprintf("-%s", formatDuration(-d))
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func getCurrentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return "unknown"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
