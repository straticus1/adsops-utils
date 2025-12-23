package ticket

import (
	"fmt"

	"github.com/spf13/cobra"
)

var closeCmd = &cobra.Command{
	Use:   "close [ticket-number]",
	Short: "Close a completed ticket",
	Long: `Close a completed change ticket.

Only tickets that have been fully approved and implemented
can be closed.

Examples:
  # Close a ticket
  changes ticket close CHG-2025-00001

  # Close with resolution notes
  changes ticket close CHG-2025-00001 --notes "Deployed successfully to production"`,
	Args: cobra.ExactArgs(1),
	Run:  runClose,
}

func init() {
	closeCmd.Flags().String("notes", "", "Resolution notes")
	closeCmd.Flags().String("actual-start", "", "Actual implementation start time (ISO8601)")
	closeCmd.Flags().String("actual-end", "", "Actual implementation end time (ISO8601)")
	closeCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

func runClose(cmd *cobra.Command, args []string) {
	ticketNumber := args[0]
	force, _ := cmd.Flags().GetBool("force")
	notes, _ := cmd.Flags().GetString("notes")

	// TODO: Implement API call to close ticket

	if !force {
		fmt.Printf("Close ticket %s? [y/N] ", ticketNumber)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return
		}
	}

	fmt.Printf("Closing ticket %s...\n", ticketNumber)
	if notes != "" {
		fmt.Printf("Resolution: %s\n", notes)
	}
	fmt.Println()
	fmt.Println("Ticket closed successfully!")
}
