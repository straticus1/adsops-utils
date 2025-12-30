package ticket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var pdfCmd = &cobra.Command{
	Use:   "pdf [ticket-id...]",
	Short: "Generate PDF documents from local ticket JSON files",
	Long: `Generate PDF documents from local ticket JSON files.

This command reads ticket JSON files from the local tickets directory
and generates corresponding PDF documents for documentation or printing.

Examples:
  # Generate PDF for a single ticket
  changes ticket pdf CHG-2025-00001

  # Generate PDFs for all tickets
  changes ticket pdf --all

  # Generate PDF with custom output directory
  changes ticket pdf CHG-2025-00001 --output-dir ./pdfs

  # Force overwrite existing PDFs
  changes ticket pdf --all --overwrite`,
	Run: runPDF,
}

func init() {
	pdfCmd.Flags().Bool("all", false, "Generate PDFs for all local ticket files")
	pdfCmd.Flags().String("output-dir", "", "Output directory for PDFs (default: same as ticket file)")
	pdfCmd.Flags().Bool("overwrite", false, "Overwrite existing PDF files")

	// Register the command
	TicketCmd.AddCommand(pdfCmd)
}

func runPDF(cmd *cobra.Command, args []string) {
	generateAll, _ := cmd.Flags().GetBool("all")
	outputDir, _ := cmd.Flags().GetString("output-dir")
	overwrite, _ := cmd.Flags().GetBool("overwrite")

	ticketsDir := getTicketsDir()

	var files []string

	if generateAll {
		// Find all ticket JSON files
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
		for _, arg := range args {
			// Handle various input formats
			if !strings.HasSuffix(arg, ".json") {
				arg = arg + ".json"
			}
			if !strings.Contains(arg, string(os.PathSeparator)) {
				arg = filepath.Join(ticketsDir, arg)
			}
			files = append(files, arg)
		}
	} else {
		fmt.Println("Usage: changes ticket pdf [ticket-id...] or changes ticket pdf --all")
		fmt.Println("Run 'changes ticket pdf --help' for more information.")
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("No ticket files found.")
		os.Exit(0)
	}

	fmt.Printf("Generating PDFs for %d ticket(s)\n\n", len(files))

	var generated, skipped, failed int

	for _, file := range files {
		ticketID := strings.TrimSuffix(filepath.Base(file), ".json")
		fmt.Printf("Generating PDF for %s... ", ticketID)

		// Determine output path
		pdfPath := strings.TrimSuffix(file, ".json") + ".pdf"
		if outputDir != "" {
			pdfPath = filepath.Join(outputDir, ticketID+".pdf")
		}

		// Check if PDF already exists
		if !overwrite {
			if _, err := os.Stat(pdfPath); err == nil {
				fmt.Println("SKIPPED (exists)")
				skipped++
				continue
			}
		}

		// Read ticket JSON
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("FAILED (read error: %v)\n", err)
			failed++
			continue
		}

		var ticketData map[string]interface{}
		if err := json.Unmarshal(data, &ticketData); err != nil {
			fmt.Printf("FAILED (parse error: %v)\n", err)
			failed++
			continue
		}

		// Ensure output directory exists
		if outputDir != "" {
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				fmt.Printf("FAILED (mkdir error: %v)\n", err)
				failed++
				continue
			}
		}

		// Generate PDF
		if err := generateTicketPDF(ticketData, pdfPath); err != nil {
			fmt.Printf("FAILED (%v)\n", err)
			failed++
			continue
		}

		fmt.Printf("OK -> %s\n", pdfPath)
		generated++
	}

	fmt.Println()
	fmt.Printf("PDF generation complete: %d generated, %d skipped, %d failed\n", generated, skipped, failed)
}
