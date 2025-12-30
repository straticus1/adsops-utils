package ticket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var importCmd = &cobra.Command{
	Use:   "import [file...]",
	Short: "Import tickets from JSON files to the changes system",
	Long: `Import change tickets from local JSON files to the changes management API.

This command reads JSON ticket files and creates them in the changes system
via the API. Existing tickets (by ID) will be skipped unless --update is specified.

Examples:
  # Import a single ticket
  changes ticket import CHG-2025-00001.json

  # Import all tickets from the tickets directory
  changes ticket import --all

  # Import multiple specific files
  changes ticket import CHG-2025-00001.json CHG-2025-00002.json

  # Import and update existing tickets
  changes ticket import --all --update

  # Import from a custom directory
  changes ticket import --dir /path/to/tickets --all

  # Dry run to see what would be imported
  changes ticket import --all --dry-run`,
	Run: runImport,
}

func init() {
	importCmd.Flags().Bool("all", false, "Import all JSON ticket files from the tickets directory")
	importCmd.Flags().Bool("update", false, "Update existing tickets instead of skipping them")
	importCmd.Flags().String("dir", "", "Directory containing ticket JSON files (default: ./tickets)")
	importCmd.Flags().Bool("dry-run", false, "Show what would be imported without actually importing")
	importCmd.Flags().String("api-url", "", "API URL (default: from config or https://api.changes.afterdarksys.com)")
	importCmd.Flags().String("token", "", "API authentication token (or set CHANGES_API_TOKEN env var)")
}

// Global token for API requests (set in runImport)
var apiToken string

func runImport(cmd *cobra.Command, args []string) {
	importAll, _ := cmd.Flags().GetBool("all")
	update, _ := cmd.Flags().GetBool("update")
	customDir, _ := cmd.Flags().GetString("dir")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	apiURL, _ := cmd.Flags().GetString("api-url")
	token, _ := cmd.Flags().GetString("token")

	// Determine API URL
	if apiURL == "" {
		apiURL = viper.GetString("api.url")
	}
	if apiURL == "" {
		apiURL = os.Getenv("CHANGES_API_URL")
	}
	if apiURL == "" {
		apiURL = "https://api.changes.afterdarksys.com"
	}

	// Determine API token
	apiToken = token
	if apiToken == "" {
		apiToken = viper.GetString("api.token")
	}
	if apiToken == "" {
		apiToken = os.Getenv("CHANGES_API_TOKEN")
	}

	// Get tickets directory
	ticketsDir := customDir
	if ticketsDir == "" {
		ticketsDir = getTicketsDir()
	}

	var files []string

	if importAll {
		// Scan directory for JSON files
		entries, err := os.ReadDir(ticketsDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading tickets directory: %v\n", err)
			os.Exit(1)
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") && strings.HasPrefix(entry.Name(), "CHG-") {
				files = append(files, filepath.Join(ticketsDir, entry.Name()))
			}
		}
	} else if len(args) > 0 {
		// Use provided file arguments
		for _, arg := range args {
			// Add .json extension if missing
			if !strings.HasSuffix(arg, ".json") {
				arg = arg + ".json"
			}
			// Check if it's just a filename or full path
			if !strings.Contains(arg, string(os.PathSeparator)) {
				arg = filepath.Join(ticketsDir, arg)
			}
			files = append(files, arg)
		}
	} else {
		fmt.Println("Usage: changes ticket import [file...] or changes ticket import --all")
		fmt.Println("Run 'changes ticket import --help' for more information.")
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("No ticket files found to import.")
		os.Exit(0)
	}

	fmt.Printf("Found %d ticket file(s) to import\n", len(files))
	if dryRun {
		fmt.Println("DRY RUN - no changes will be made")
	}
	fmt.Println()

	var imported, skipped, failed int

	for _, file := range files {
		ticketID := strings.TrimSuffix(filepath.Base(file), ".json")
		fmt.Printf("Processing %s... ", ticketID)

		// Read the file
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("FAILED (read error: %v)\n", err)
			failed++
			continue
		}

		// Parse to validate JSON
		var ticketData map[string]interface{}
		if err := json.Unmarshal(data, &ticketData); err != nil {
			fmt.Printf("FAILED (invalid JSON: %v)\n", err)
			failed++
			continue
		}

		if dryRun {
			fmt.Println("would import")
			imported++
			continue
		}

		// Check if ticket exists (GET request)
		exists := checkTicketExists(apiURL, ticketID)

		if exists && !update {
			fmt.Println("SKIPPED (already exists)")
			skipped++
			continue
		}

		// Import or update the ticket
		var err2 error
		if exists && update {
			err2 = updateTicketViaAPI(apiURL, ticketID, data)
		} else {
			err2 = createTicketViaAPI(apiURL, data)
		}

		if err2 != nil {
			fmt.Printf("FAILED (%v)\n", err2)
			failed++
			continue
		}

		if exists {
			fmt.Println("UPDATED")
		} else {
			fmt.Println("IMPORTED")
		}
		imported++
	}

	fmt.Println()
	fmt.Printf("Import complete: %d imported, %d skipped, %d failed\n", imported, skipped, failed)
}

func checkTicketExists(apiURL, ticketID string) bool {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/tickets/%s", apiURL, ticketID), nil)
	if err != nil {
		return false
	}
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func createTicketViaAPI(apiURL string, data []byte) error {
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/v1/tickets", apiURL),
		strings.NewReader(string(data)),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func updateTicketViaAPI(apiURL, ticketID string, data []byte) error {
	req, err := http.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("%s/v1/tickets/%s", apiURL, ticketID),
		strings.NewReader(string(data)),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
