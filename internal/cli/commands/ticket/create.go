package ticket

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// CreateTicketData represents the full ticket data for JSON storage
type CreateTicketData struct {
	ID                   string   `json:"id"`
	Title                string   `json:"title"`
	Description          string   `json:"description"`
	Status               string   `json:"status"`
	Priority             string   `json:"priority"`
	Risk                 string   `json:"risk"`
	Type                 string   `json:"type"`
	Industry             string   `json:"industry"`
	ComplianceFrameworks []string `json:"compliance_frameworks"`
	AffectedSystems      []string `json:"affected_systems"`
	AcceptanceCriteria   []string `json:"acceptance_criteria"`
	TestingPlan          string   `json:"testing_plan"`
	RollbackPlan         string   `json:"rollback_plan"`
	CreatedBy            string   `json:"created_by"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
	Sprint               string   `json:"sprint"`
	Assignee             *string  `json:"assignee"`
	ApprovalsRequired    []string `json:"approvals_required"`
	Approvals            []string `json:"approvals"`
	Dependencies         []string `json:"dependencies"`
	Comments             []struct {
		Author    string `json:"author"`
		Timestamp string `json:"timestamp"`
		Text      string `json:"text"`
	} `json:"comments"`
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new change ticket",
	Long: `Create a new change ticket either interactively or by providing flags.

Interactive mode will guide you through the ticket creation process with
industry-specific compliance requirements.

Examples:
  # Create interactively
  changes ticket create

  # Create with minimal flags (will prompt for required fields)
  changes ticket create --title "Database migration"

  # Create with all required fields
  changes ticket create \
    --title "Update production database schema" \
    --description "Adding new index for performance" \
    --priority normal \
    --risk medium \
    --industry finance \
    --compliance glba,sox \
    --approval-types operations,it,security`,
	Run: runCreate,
}

func init() {
	createCmd.Flags().String("title", "", "Ticket title (required)")
	createCmd.Flags().String("description", "", "Ticket description")
	createCmd.Flags().StringP("priority", "p", "normal", "Priority (emergency, urgent, high, normal, low)")
	createCmd.Flags().StringP("risk", "r", "medium", "Risk level (critical, high, medium, low)")
	createCmd.Flags().StringP("industry", "i", "", "Industry (healthcare, it, government, insurance, finance)")
	createCmd.Flags().StringSlice("compliance", []string{}, "Compliance frameworks (glba, sox, hipaa, gdpr, banking_secrecy_act)")
	createCmd.Flags().StringSlice("approval-types", []string{}, "Required approval types")
	createCmd.Flags().StringSlice("affected-systems", []string{}, "Affected systems")
	createCmd.Flags().String("change-type", "", "Type of change")
	createCmd.Flags().String("impact", "", "Impact description")
	createCmd.Flags().String("rollback", "", "Rollback plan")
	createCmd.Flags().String("testing", "", "Testing plan")
	createCmd.Flags().Bool("submit", false, "Submit immediately instead of saving as draft")
	createCmd.Flags().Bool("interactive", true, "Use interactive mode")
}

// getMaxTicketNumFromDB attempts to get the max ticket number from the database
// Returns 0 if database is unavailable or query fails (graceful degradation)
func getMaxTicketNumFromDB(year int) int {
	// Try to load database config from viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/adsops-utils")
	viper.SetEnvPrefix("ADSOPS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	_ = viper.ReadInConfig() // Ignore error, env vars may be enough

	host := viper.GetString("database.host")
	port := viper.GetInt("database.port")
	user := viper.GetString("database.user")
	password := viper.GetString("database.password")
	dbname := viper.GetString("database.dbname")
	sslmode := viper.GetString("database.sslmode")

	// If no database config, return 0 (will use local files only)
	if host == "" || user == "" || dbname == "" {
		return 0
	}

	// Set defaults
	if port == 0 {
		port = 5432
	}
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=5",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return 0
	}
	defer db.Close()

	// Quick connection test with timeout
	if err := db.Ping(); err != nil {
		return 0
	}

	// Query for max ticket number this year
	var maxNum sql.NullInt64
	query := `
		SELECT MAX(
			CAST(SUBSTRING(ticket_number FROM '[0-9]+$') AS INTEGER)
		)
		FROM change_tickets
		WHERE EXTRACT(YEAR FROM created_at) = $1
	`
	if err := db.QueryRow(query, year).Scan(&maxNum); err != nil {
		return 0
	}

	if maxNum.Valid {
		return int(maxNum.Int64)
	}
	return 0
}

// getMaxTicketNumFromFiles scans local JSON files for the max ticket number
func getMaxTicketNumFromFiles(ticketsDir string, year int) int {
	entries, err := os.ReadDir(ticketsDir)
	if err != nil {
		return 0
	}

	var maxNum int
	prefix := fmt.Sprintf("CHG-%d-", year)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".json") {
			continue
		}
		// Extract number from CHG-2025-00001.json -> 1
		numStr := strings.TrimPrefix(name, prefix)
		numStr = strings.TrimSuffix(numStr, ".json")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}
		if num > maxNum {
			maxNum = num
		}
	}
	return maxNum
}

// getNextTicketNumber determines the next available ticket number
// Checks BOTH local JSON files AND the database (if available) to prevent ID collisions
func getNextTicketNumber() (string, error) {
	ticketsDir := getTicketsDir()
	year := time.Now().Year()

	// Ensure tickets directory exists
	if err := os.MkdirAll(ticketsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tickets directory: %w", err)
	}

	// Get max from local JSON files
	maxFromFiles := getMaxTicketNumFromFiles(ticketsDir, year)

	// Get max from database (gracefully handles unavailable DB)
	maxFromDB := getMaxTicketNumFromDB(year)

	// Use the higher of the two
	maxNum := maxFromFiles
	if maxFromDB > maxNum {
		maxNum = maxFromDB
	}

	// Generate next ID
	nextNum := maxNum + 1
	ticketID := fmt.Sprintf("CHG-%d-%05d", year, nextNum)

	// Final safety check: ensure file doesn't already exist (race condition protection)
	for {
		filename := filepath.Join(ticketsDir, ticketID+".json")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			break // File doesn't exist, safe to use this ID
		}
		// File exists, increment and try again
		nextNum++
		ticketID = fmt.Sprintf("CHG-%d-%05d", year, nextNum)
	}

	return ticketID, nil
}

// saveTicket saves a ticket to the local tickets directory
func saveTicket(ticket *CreateTicketData) error {
	ticketsDir := getTicketsDir()
	if err := os.MkdirAll(ticketsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tickets directory: %w", err)
	}

	filename := filepath.Join(ticketsDir, ticket.ID+".json")
	data, err := json.MarshalIndent(ticket, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal ticket: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write ticket file: %w", err)
	}

	return nil
}

func runCreate(cmd *cobra.Command, args []string) {
	interactive, _ := cmd.Flags().GetBool("interactive")

	// If interactive is explicitly false or we have a title, use non-interactive
	title, _ := cmd.Flags().GetString("title")
	if title != "" {
		interactive = false
	}

	if interactive {
		runInteractiveCreate(cmd)
		return
	}

	// Non-interactive creation
	if title == "" {
		fmt.Println("Error: --title is required in non-interactive mode")
		fmt.Println("Usage: changes ticket create --title \"Your ticket title\" [other flags]")
		os.Exit(1)
	}

	// Get next ticket number
	ticketID, err := getNextTicketNumber()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating ticket ID: %v\n", err)
		os.Exit(1)
	}

	// Collect all flags
	description, _ := cmd.Flags().GetString("description")
	priority, _ := cmd.Flags().GetString("priority")
	risk, _ := cmd.Flags().GetString("risk")
	industry, _ := cmd.Flags().GetString("industry")
	compliance, _ := cmd.Flags().GetStringSlice("compliance")
	approvalTypes, _ := cmd.Flags().GetStringSlice("approval-types")
	affectedSystems, _ := cmd.Flags().GetStringSlice("affected-systems")
	changeType, _ := cmd.Flags().GetString("change-type")
	rollback, _ := cmd.Flags().GetString("rollback")
	testing, _ := cmd.Flags().GetString("testing")
	submit, _ := cmd.Flags().GetBool("submit")

	now := time.Now().UTC()
	status := "draft"
	if submit {
		status = "submitted"
	}

	// Determine current user
	user := os.Getenv("USER")
	if user == "" {
		user = "unknown"
	}
	createdBy := user + "@afterdarksys.com"

	// Calculate sprint
	_, week := now.ISOWeek()
	quarter := (now.Month()-1)/3 + 1
	sprint := fmt.Sprintf("%d-Q%d-Sprint-%d", now.Year(), quarter, (week-1)%2+1)

	ticket := &CreateTicketData{
		ID:                   ticketID,
		Title:                title,
		Description:          description,
		Status:               status,
		Priority:             priority,
		Risk:                 risk,
		Type:                 changeType,
		Industry:             industry,
		ComplianceFrameworks: compliance,
		AffectedSystems:      affectedSystems,
		AcceptanceCriteria:   []string{},
		TestingPlan:          testing,
		RollbackPlan:         rollback,
		CreatedBy:            createdBy,
		CreatedAt:            now.Format(time.RFC3339),
		UpdatedAt:            now.Format(time.RFC3339),
		Sprint:               sprint,
		Assignee:             nil,
		ApprovalsRequired:    approvalTypes,
		Approvals:            []string{},
		Dependencies:         []string{},
		Comments: []struct {
			Author    string `json:"author"`
			Timestamp string `json:"timestamp"`
			Text      string `json:"text"`
		}{
			{
				Author:    createdBy,
				Timestamp: now.Format(time.RFC3339),
				Text:      "Ticket created via CLI.",
			},
		},
	}

	if err := saveTicket(ticket); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving ticket: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Creating ticket: %s\n", title)
	fmt.Printf("Ticket created successfully: %s\n", ticketID)
	if submit {
		fmt.Println("Status: submitted (awaiting approval)")
	} else {
		fmt.Println("Status: draft")
		fmt.Println("Use 'changes ticket submit " + ticketID + "' to submit for approval.")
	}
}

func runInteractiveCreate(cmd *cobra.Command) {
	// For now, provide a simple interactive flow using standard input
	fmt.Println("Interactive ticket creation")
	fmt.Println("===========================")
	fmt.Println()
	fmt.Println("Use --interactive=false with --title to create non-interactively.")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  changes ticket create --interactive=false --title \"My ticket\" --priority high")
	fmt.Println()

	// List existing tickets
	tickets, err := loadLocalTickets()
	if err == nil && len(tickets) > 0 {
		fmt.Println("Recent tickets:")
		// Sort by created date descending
		sort.Slice(tickets, func(i, j int) bool {
			return tickets[i].CreatedAt.After(tickets[j].CreatedAt)
		})
		for i, t := range tickets {
			if i >= 5 {
				break
			}
			fmt.Printf("  %s: %s (%s)\n", t.ID, t.Title, t.Status)
		}
	}
}
