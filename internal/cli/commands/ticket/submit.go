package ticket

import (
	"fmt"

	"github.com/spf13/cobra"
)

var submitCmd = &cobra.Command{
	Use:   "submit [ticket-number]",
	Short: "Submit a draft ticket for approval",
	Long: `Submit a draft ticket for approval.

This will trigger the approval workflow and send notifications
to all required approvers.

Examples:
  # Submit a ticket
  changes ticket submit CHG-2025-00001

  # Submit with a note to approvers
  changes ticket submit CHG-2025-00001 --note "Please prioritize - needed for release"`,
	Args: cobra.ExactArgs(1),
	Run:  runSubmit,
}

func init() {
	submitCmd.Flags().String("note", "", "Note to include with approval requests")
	submitCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

func runSubmit(cmd *cobra.Command, args []string) {
	ticketNumber := args[0]
	force, _ := cmd.Flags().GetBool("force")
	note, _ := cmd.Flags().GetString("note")

	// TODO: Implement API call to submit ticket

	if !force {
		fmt.Printf("Submit ticket %s for approval? [y/N] ", ticketNumber)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return
		}
	}

	fmt.Printf("Submitting ticket %s...\n", ticketNumber)
	if note != "" {
		fmt.Printf("Note: %s\n", note)
	}
	fmt.Println()
	fmt.Println("Ticket submitted successfully!")
	fmt.Println()
	fmt.Println("Approvals requested from:")
	fmt.Println("  - Operations Team (ops-team@example.com)")
	fmt.Println("  - IT Team (it-team@example.com)")
	fmt.Println("  - Security Team (security@example.com)")
}
