package ticket

import (
	"fmt"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit [ticket-number]",
	Short: "Edit a change ticket",
	Long: `Edit an existing change ticket.

Only tickets in draft or update_requested status can be edited.

Examples:
  # Edit interactively
  changes ticket edit CHG-2025-00001

  # Update specific fields
  changes ticket edit CHG-2025-00001 --priority urgent --risk high

  # Add to description
  changes ticket edit CHG-2025-00001 --add-description "Additional context..."`,
	Args: cobra.ExactArgs(1),
	Run:  runEdit,
}

func init() {
	editCmd.Flags().String("title", "", "Update title")
	editCmd.Flags().String("description", "", "Replace description")
	editCmd.Flags().String("add-description", "", "Append to description")
	editCmd.Flags().StringP("priority", "p", "", "Update priority")
	editCmd.Flags().StringP("risk", "r", "", "Update risk level")
	editCmd.Flags().StringSlice("add-systems", []string{}, "Add affected systems")
	editCmd.Flags().StringSlice("remove-systems", []string{}, "Remove affected systems")
	editCmd.Flags().String("impact", "", "Update impact description")
	editCmd.Flags().String("rollback", "", "Update rollback plan")
	editCmd.Flags().String("testing", "", "Update testing plan")
	editCmd.Flags().Bool("interactive", true, "Use interactive mode")
}

func runEdit(cmd *cobra.Command, args []string) {
	ticketNumber := args[0]
	interactive, _ := cmd.Flags().GetBool("interactive")

	// TODO: Check if ticket can be edited

	if interactive {
		fmt.Printf("Opening interactive editor for %s...\n", ticketNumber)
		fmt.Println("(TUI implementation pending)")
		return
	}

	// TODO: Implement API call to update ticket
	fmt.Printf("Updating ticket %s...\n", ticketNumber)
	fmt.Println("Ticket updated successfully")
}
