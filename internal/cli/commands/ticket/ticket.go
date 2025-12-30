package ticket

import (
	"github.com/spf13/cobra"
)

// TicketCmd represents the ticket command group
var TicketCmd = &cobra.Command{
	Use:     "ticket",
	Aliases: []string{"tickets", "t"},
	Short:   "Manage change tickets",
	Long: `Create, view, edit, and manage change tickets.

Examples:
  # Create a new ticket interactively
  changes ticket create

  # Create a ticket with flags
  changes ticket create --title "Update database schema" --priority high

  # List all tickets
  changes ticket list

  # View a specific ticket
  changes ticket view CHG-2025-00001

  # Edit a ticket
  changes ticket edit CHG-2025-00001

  # Submit a draft ticket for approval
  changes ticket submit CHG-2025-00001

  # Close a completed ticket
  changes ticket close CHG-2025-00001

  # Import tickets from JSON files
  changes ticket import --all

  # Export tickets to JSON or PDF
  changes ticket export CHG-2025-00001 --format pdf`,
}

func init() {
	// Add subcommands
	TicketCmd.AddCommand(createCmd)
	TicketCmd.AddCommand(listCmd)
	TicketCmd.AddCommand(viewCmd)
	TicketCmd.AddCommand(editCmd)
	TicketCmd.AddCommand(submitCmd)
	TicketCmd.AddCommand(closeCmd)
	TicketCmd.AddCommand(openCmd)
	TicketCmd.AddCommand(cancelCmd)
	TicketCmd.AddCommand(importCmd)
	TicketCmd.AddCommand(exportCmd)
	// pdfCmd is registered in pdf.go init()
}
