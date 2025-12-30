package ticket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var exportCmd = &cobra.Command{
	Use:   "export [ticket-id...]",
	Short: "Export tickets from the changes system to JSON files",
	Long: `Export change tickets from the changes management API to local JSON files.

This command fetches tickets from the API and saves them as JSON files locally.
Useful for backup, migration, or offline analysis.

Examples:
  # Export a single ticket
  changes ticket export CHG-2025-00001

  # Export multiple tickets
  changes ticket export CHG-2025-00001 CHG-2025-00002

  # Export all tickets from the API
  changes ticket export --all

  # Export to a specific directory
  changes ticket export --all --dir /path/to/backup

  # Export tickets matching a filter
  changes ticket export --status submitted,in_review

  # Export as PDF documents
  changes ticket export CHG-2025-00001 --format pdf

  # Export all tickets as both JSON and PDF
  changes ticket export --all --format all`,
	Run: runExport,
}

func init() {
	exportCmd.Flags().Bool("all", false, "Export all tickets from the API")
	exportCmd.Flags().String("dir", "", "Output directory (default: ./tickets)")
	exportCmd.Flags().StringSlice("status", []string{}, "Filter by status when using --all")
	exportCmd.Flags().String("format", "json", "Output format: json, pdf, or all")
	exportCmd.Flags().Bool("overwrite", false, "Overwrite existing files")
	exportCmd.Flags().String("api-url", "", "API URL (default: from config or https://api.changes.afterdarksys.com)")
}

func runExport(cmd *cobra.Command, args []string) {
	exportAll, _ := cmd.Flags().GetBool("all")
	outputDir, _ := cmd.Flags().GetString("dir")
	statusFilter, _ := cmd.Flags().GetStringSlice("status")
	format, _ := cmd.Flags().GetString("format")
	overwrite, _ := cmd.Flags().GetBool("overwrite")
	apiURL, _ := cmd.Flags().GetString("api-url")

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

	// Get output directory
	if outputDir == "" {
		outputDir = getTicketsDir()
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	var ticketIDs []string

	if exportAll {
		// Fetch ticket list from API
		ids, err := fetchTicketIDsFromAPI(apiURL, statusFilter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching ticket list: %v\n", err)
			os.Exit(1)
		}
		ticketIDs = ids
	} else if len(args) > 0 {
		ticketIDs = args
	} else {
		fmt.Println("Usage: changes ticket export [ticket-id...] or changes ticket export --all")
		fmt.Println("Run 'changes ticket export --help' for more information.")
		os.Exit(1)
	}

	if len(ticketIDs) == 0 {
		fmt.Println("No tickets found to export.")
		os.Exit(0)
	}

	fmt.Printf("Exporting %d ticket(s) to %s\n", len(ticketIDs), outputDir)
	fmt.Println()

	var exported, skipped, failed int

	for _, ticketID := range ticketIDs {
		fmt.Printf("Exporting %s... ", ticketID)

		// Fetch ticket from API
		ticketData, err := fetchTicketFromAPI(apiURL, ticketID)
		if err != nil {
			fmt.Printf("FAILED (%v)\n", err)
			failed++
			continue
		}

		// Export JSON
		if format == "json" || format == "all" {
			jsonFile := filepath.Join(outputDir, ticketID+".json")
			if !overwrite {
				if _, err := os.Stat(jsonFile); err == nil {
					fmt.Print("SKIPPED (exists) ")
					skipped++
					continue
				}
			}

			// Pretty print JSON
			prettyJSON, err := json.MarshalIndent(ticketData, "", "  ")
			if err != nil {
				fmt.Printf("FAILED (marshal error: %v)\n", err)
				failed++
				continue
			}

			if err := os.WriteFile(jsonFile, prettyJSON, 0600); err != nil {
				fmt.Printf("FAILED (write error: %v)\n", err)
				failed++
				continue
			}

			if format == "json" {
				fmt.Println("OK")
			}
		}

		// Export PDF
		if format == "pdf" || format == "all" {
			pdfFile := filepath.Join(outputDir, ticketID+".pdf")
			if !overwrite {
				if _, err := os.Stat(pdfFile); err == nil {
					if format == "pdf" {
						fmt.Print("SKIPPED (exists) ")
						skipped++
						continue
					}
				}
			}

			if err := generateTicketPDF(ticketData, pdfFile); err != nil {
				if format == "pdf" {
					fmt.Printf("FAILED (PDF error: %v)\n", err)
					failed++
					continue
				} else {
					fmt.Printf("(PDF failed) ")
				}
			}

			if format == "all" {
				fmt.Println("OK (JSON+PDF)")
			} else {
				fmt.Println("OK")
			}
		}

		exported++
	}

	fmt.Println()
	fmt.Printf("Export complete: %d exported, %d skipped, %d failed\n", exported, skipped, failed)
}

func fetchTicketIDsFromAPI(apiURL string, statusFilter []string) ([]string, error) {
	url := fmt.Sprintf("%s/v1/tickets?per_page=1000", apiURL)
	if len(statusFilter) > 0 {
		url += "&status=" + strings.Join(statusFilter, ",")
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Tickets []struct {
			ID           string `json:"id"`
			TicketNumber string `json:"ticket_number"`
		} `json:"tickets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var ids []string
	for _, t := range result.Tickets {
		id := t.TicketNumber
		if id == "" {
			id = t.ID
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func fetchTicketFromAPI(apiURL, ticketID string) (map[string]interface{}, error) {
	resp, err := http.Get(fmt.Sprintf("%s/v1/tickets/%s", apiURL, ticketID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Ticket map[string]interface{} `json:"ticket"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Ticket, nil
}

// generateTicketPDF creates a PDF document from ticket data
// This generates a simple text-based PDF using basic PDF primitives
func generateTicketPDF(ticketData map[string]interface{}, outputPath string) error {
	// Extract ticket fields
	ticketID := getString(ticketData, "ticket_number", getString(ticketData, "id", "Unknown"))
	title := getString(ticketData, "title", "Untitled")
	description := getString(ticketData, "description", "No description")
	status := getString(ticketData, "status", "unknown")
	priority := getString(ticketData, "priority", "normal")
	risk := getString(ticketData, "risk_level", getString(ticketData, "risk", "medium"))
	industry := getString(ticketData, "industry", "")
	createdBy := getString(ticketData, "created_by", "Unknown")
	createdAt := getString(ticketData, "created_at", time.Now().Format(time.RFC3339))
	rollbackPlan := getString(ticketData, "rollback_plan", "")
	testingPlan := getString(ticketData, "testing_plan", "")

	// Get arrays
	compliance := getStringArray(ticketData, "compliance_frameworks")
	affectedSystems := getStringArray(ticketData, "affected_systems")
	approvalsRequired := getStringArrayAlt(ticketData, "approvals_required", "requires_approval_types")

	// Build PDF content
	var content strings.Builder

	// PDF Header
	content.WriteString("%PDF-1.4\n")
	content.WriteString("1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n")
	content.WriteString("2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n")
	content.WriteString("3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]/Contents 4 0 R/Resources<</Font<</F1 5 0 R/F2 6 0 R>>>>>>endobj\n")
	content.WriteString("5 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica-Bold>>endobj\n")
	content.WriteString("6 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj\n")

	// Build page content
	var pageContent strings.Builder
	y := 750

	// Title section
	pageContent.WriteString("BT\n")
	pageContent.WriteString(fmt.Sprintf("/F1 16 Tf %d %d Td (Change Request: %s) Tj\n", 50, y, escapePDF(ticketID)))
	y -= 25
	pageContent.WriteString(fmt.Sprintf("/F1 12 Tf %d %d Td (%s) Tj\n", 50, y, escapePDF(title)))
	y -= 30

	// Status and Priority
	pageContent.WriteString(fmt.Sprintf("/F2 10 Tf %d %d Td (Status: %s    Priority: %s    Risk: %s) Tj\n", 50, y, status, priority, risk))
	y -= 20
	pageContent.WriteString(fmt.Sprintf("%d %d Td (Created By: %s    Date: %s) Tj\n", 50, y, escapePDF(createdBy), formatDate(createdAt)))
	y -= 25

	// Compliance
	if industry != "" || len(compliance) > 0 {
		pageContent.WriteString(fmt.Sprintf("/F1 10 Tf %d %d Td (Compliance Information) Tj\n", 50, y))
		y -= 15
		pageContent.WriteString("/F2 10 Tf\n")
		if industry != "" {
			pageContent.WriteString(fmt.Sprintf("%d %d Td (Industry: %s) Tj\n", 50, y, industry))
			y -= 15
		}
		if len(compliance) > 0 {
			pageContent.WriteString(fmt.Sprintf("%d %d Td (Frameworks: %s) Tj\n", 50, y, strings.Join(compliance, ", ")))
			y -= 15
		}
		y -= 10
	}

	// Description
	pageContent.WriteString(fmt.Sprintf("/F1 10 Tf %d %d Td (Description) Tj\n", 50, y))
	y -= 15
	pageContent.WriteString("/F2 10 Tf\n")
	for _, line := range wrapText(description, 80) {
		if y < 100 {
			break // Don't overflow page
		}
		pageContent.WriteString(fmt.Sprintf("%d %d Td (%s) Tj\n", 50, y, escapePDF(line)))
		y -= 12
	}
	y -= 10

	// Affected Systems
	if len(affectedSystems) > 0 && y > 200 {
		pageContent.WriteString(fmt.Sprintf("/F1 10 Tf %d %d Td (Affected Systems) Tj\n", 50, y))
		y -= 15
		pageContent.WriteString(fmt.Sprintf("/F2 10 Tf %d %d Td (%s) Tj\n", 50, y, escapePDF(strings.Join(affectedSystems, ", "))))
		y -= 20
	}

	// Approvals Required
	if len(approvalsRequired) > 0 && y > 180 {
		pageContent.WriteString(fmt.Sprintf("/F1 10 Tf %d %d Td (Approvals Required) Tj\n", 50, y))
		y -= 15
		pageContent.WriteString(fmt.Sprintf("/F2 10 Tf %d %d Td (%s) Tj\n", 50, y, escapePDF(strings.Join(approvalsRequired, ", "))))
		y -= 20
	}

	// Rollback Plan
	if rollbackPlan != "" && y > 160 {
		pageContent.WriteString(fmt.Sprintf("/F1 10 Tf %d %d Td (Rollback Plan) Tj\n", 50, y))
		y -= 15
		pageContent.WriteString("/F2 10 Tf\n")
		for _, line := range wrapText(rollbackPlan, 80) {
			if y < 100 {
				break
			}
			pageContent.WriteString(fmt.Sprintf("%d %d Td (%s) Tj\n", 50, y, escapePDF(line)))
			y -= 12
		}
		y -= 10
	}

	// Testing Plan
	if testingPlan != "" && y > 140 {
		pageContent.WriteString(fmt.Sprintf("/F1 10 Tf %d %d Td (Testing Plan) Tj\n", 50, y))
		y -= 15
		pageContent.WriteString("/F2 10 Tf\n")
		for _, line := range wrapText(testingPlan, 80) {
			if y < 100 {
				break
			}
			pageContent.WriteString(fmt.Sprintf("%d %d Td (%s) Tj\n", 50, y, escapePDF(line)))
			y -= 12
		}
	}

	// Footer
	pageContent.WriteString(fmt.Sprintf("/F2 8 Tf %d 30 Td (Generated by After Dark Systems Change Management - %s) Tj\n", 50, time.Now().Format("2006-01-02 15:04:05")))

	pageContent.WriteString("ET\n")

	// Add content stream
	streamContent := pageContent.String()
	content.WriteString(fmt.Sprintf("4 0 obj<</Length %d>>stream\n%sendstream endobj\n", len(streamContent), streamContent))

	// PDF trailer
	content.WriteString("xref\n")
	content.WriteString("0 7\n")
	content.WriteString("0000000000 65535 f \n")
	content.WriteString("0000000009 00000 n \n")
	content.WriteString("0000000052 00000 n \n")
	content.WriteString("0000000101 00000 n \n")
	content.WriteString("0000000249 00000 n \n")
	content.WriteString("0000000350 00000 n \n")
	content.WriteString("0000000417 00000 n \n")
	content.WriteString("trailer<</Size 7/Root 1 0 R>>\n")
	content.WriteString("startxref\n")
	content.WriteString("478\n")
	content.WriteString("%%EOF\n")

	return os.WriteFile(outputPath, []byte(content.String()), 0600)
}

func getString(data map[string]interface{}, key, defaultVal string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func getStringArrayAlt(data map[string]interface{}, key1, key2 string) []string {
	result := getStringArray(data, key1)
	if len(result) == 0 {
		return getStringArray(data, key2)
	}
	return result
}

func getStringArray(data map[string]interface{}, key string) []string {
	if v, ok := data[key]; ok {
		if arr, ok := v.([]interface{}); ok {
			var result []string
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
		if arr, ok := v.([]string); ok {
			return arr
		}
	}
	return nil
}

func escapePDF(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

func wrapText(text string, maxLen int) []string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")

	var lines []string
	words := strings.Fields(text)
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len()+len(word)+1 > maxLen {
			if currentLine.Len() > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
			}
		}
		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
		}
		currentLine.WriteString(word)
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

func formatDate(dateStr string) string {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("2006-01-02")
}
