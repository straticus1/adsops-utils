package ticket

import (
	"fmt"

	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:     "reopen [ticket-number]",
	Aliases: []string{"open"},
	Short:   "Reopen a closed ticket",
	Long: `Reopen a previously closed change ticket.

This may be necessary if issues are discovered after
implementation that require additional changes.

Examples:
  # Reopen a ticket
  changes ticket reopen CHG-2025-00001

  # Reopen with reason
  changes ticket reopen CHG-2025-00001 --reason "Additional changes needed"`,
	Args: cobra.ExactArgs(1),
	Run:  runOpen,
}

func init() {
	openCmd.Flags().String("reason", "", "Reason for reopening (required)")
	openCmd.MarkFlagRequired("reason")
}

func runOpen(cmd *cobra.Command, args []string) {
	ticketNumber := args[0]
	reason, _ := cmd.Flags().GetString("reason")

	// TODO: Implement API call to reopen ticket

	fmt.Printf("Reopening ticket %s...\n", ticketNumber)
	fmt.Printf("Reason: %s\n", reason)
	fmt.Println()
	fmt.Println("Ticket reopened successfully!")
	fmt.Println("Status changed to: update_requested")
}

var cancelCmd = &cobra.Command{
	Use:   "cancel [ticket-number]",
	Short: "Cancel a ticket",
	Long: `Cancel a draft or submitted ticket.

Only tickets that have not yet been approved can be cancelled.

Examples:
  # Cancel a ticket
  changes ticket cancel CHG-2025-00001

  # Cancel with reason
  changes ticket cancel CHG-2025-00001 --reason "No longer needed"`,
	Args: cobra.ExactArgs(1),
	Run:  runCancel,
}

func init() {
	cancelCmd.Flags().String("reason", "", "Reason for cancellation (required)")
	cancelCmd.MarkFlagRequired("reason")
	cancelCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

func runCancel(cmd *cobra.Command, args []string) {
	ticketNumber := args[0]
	force, _ := cmd.Flags().GetBool("force")
	reason, _ := cmd.Flags().GetString("reason")

	// TODO: Implement API call to cancel ticket

	if !force {
		fmt.Printf("Cancel ticket %s? This cannot be undone. [y/N] ", ticketNumber)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return
		}
	}

	fmt.Printf("Cancelling ticket %s...\n", ticketNumber)
	fmt.Printf("Reason: %s\n", reason)
	fmt.Println()
	fmt.Println("Ticket cancelled successfully!")
}
