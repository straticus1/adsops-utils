package approval

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// ApprovalCmd represents the approval command group
var ApprovalCmd = &cobra.Command{
	Use:     "approval",
	Aliases: []string{"approvals", "approve", "a"},
	Short:   "Manage approvals",
	Long: `View and manage change ticket approvals.

Examples:
  # List pending approvals
  changes approval list

  # Approve a ticket
  changes approval approve CHG-2025-00001

  # Deny a ticket
  changes approval deny CHG-2025-00001 --reason "Missing rollback plan"

  # Request an update
  changes approval request-update CHG-2025-00001 --comment "Please add testing plan"`,
}

func init() {
	ApprovalCmd.AddCommand(listCmd)
	ApprovalCmd.AddCommand(approveCmd)
	ApprovalCmd.AddCommand(denyCmd)
	ApprovalCmd.AddCommand(requestUpdateCmd)
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List pending approvals",
	Long: `List approvals waiting for your action.

Examples:
  # List all pending approvals
  changes approval list

  # List all approvals (including decided)
  changes approval list --all

  # Filter by approval type
  changes approval list --type security`,
	Run: runList,
}

func init() {
	listCmd.Flags().Bool("all", false, "Show all approvals, not just pending")
	listCmd.Flags().String("type", "", "Filter by approval type")
	listCmd.Flags().String("status", "", "Filter by status")
}

func runList(cmd *cobra.Command, args []string) {
	// TODO: Implement API call to list approvals

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TICKET\tTYPE\tSTATUS\tTITLE\tEXPIRES")
	fmt.Fprintln(w, "------\t----\t------\t-----\t-------")
	fmt.Fprintln(w, "CHG-2025-00001\toperations\tpending\tDatabase schema update\t2025-12-27")
	fmt.Fprintln(w, "CHG-2025-00002\tsecurity\tpending\tNetwork configuration\t2025-12-28")
	fmt.Fprintln(w, "CHG-2025-00003\tit\tapproved\tSecurity patch\t-")
	w.Flush()
}

var approveCmd = &cobra.Command{
	Use:   "approve [ticket-number]",
	Short: "Approve a change ticket",
	Long: `Approve a change ticket for your approval type.

Examples:
  # Approve a ticket
  changes approval approve CHG-2025-00001

  # Approve with comment
  changes approval approve CHG-2025-00001 --comment "Looks good"

  # Approve with conditions
  changes approval approve CHG-2025-00001 --conditions "Deploy during maintenance window only"`,
	Args: cobra.ExactArgs(1),
	Run:  runApprove,
}

func init() {
	approveCmd.Flags().String("comment", "", "Comment for the approval")
	approveCmd.Flags().String("conditions", "", "Conditions for the approval")
	approveCmd.Flags().Bool("force", false, "Skip confirmation")
}

func runApprove(cmd *cobra.Command, args []string) {
	ticketNumber := args[0]
	force, _ := cmd.Flags().GetBool("force")
	comment, _ := cmd.Flags().GetString("comment")
	conditions, _ := cmd.Flags().GetString("conditions")

	if !force {
		fmt.Printf("Approve ticket %s? [y/N] ", ticketNumber)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return
		}
	}

	// TODO: Implement API call to approve

	fmt.Printf("Approving ticket %s...\n", ticketNumber)
	if comment != "" {
		fmt.Printf("Comment: %s\n", comment)
	}
	if conditions != "" {
		fmt.Printf("Conditions: %s\n", conditions)
	}
	fmt.Println()
	fmt.Println("Ticket approved successfully!")
}

var denyCmd = &cobra.Command{
	Use:   "deny [ticket-number]",
	Short: "Deny a change ticket",
	Long: `Deny a change ticket for your approval type.

A reason is required when denying a ticket.

Examples:
  # Deny a ticket
  changes approval deny CHG-2025-00001 --reason "Missing security review"`,
	Args: cobra.ExactArgs(1),
	Run:  runDeny,
}

func init() {
	denyCmd.Flags().String("reason", "", "Reason for denial (required)")
	denyCmd.Flags().String("comment", "", "Additional comment")
	denyCmd.MarkFlagRequired("reason")
}

func runDeny(cmd *cobra.Command, args []string) {
	ticketNumber := args[0]
	reason, _ := cmd.Flags().GetString("reason")
	comment, _ := cmd.Flags().GetString("comment")

	// TODO: Implement API call to deny

	fmt.Printf("Denying ticket %s...\n", ticketNumber)
	fmt.Printf("Reason: %s\n", reason)
	if comment != "" {
		fmt.Printf("Comment: %s\n", comment)
	}
	fmt.Println()
	fmt.Println("Ticket denied.")
}

var requestUpdateCmd = &cobra.Command{
	Use:     "request-update [ticket-number]",
	Aliases: []string{"update"},
	Short:   "Request an update to a ticket",
	Long: `Request changes to a ticket before approving.

The ticket creator will be notified and can update the ticket
to address your concerns.

Examples:
  # Request an update
  changes approval request-update CHG-2025-00001 --comment "Please add rollback plan"`,
	Args: cobra.ExactArgs(1),
	Run:  runRequestUpdate,
}

func init() {
	requestUpdateCmd.Flags().String("comment", "", "Comment explaining requested changes (required)")
	requestUpdateCmd.Flags().String("required-changes", "", "Specific changes required")
	requestUpdateCmd.MarkFlagRequired("comment")
}

func runRequestUpdate(cmd *cobra.Command, args []string) {
	ticketNumber := args[0]
	comment, _ := cmd.Flags().GetString("comment")
	changes, _ := cmd.Flags().GetString("required-changes")

	// TODO: Implement API call to request update

	fmt.Printf("Requesting update for ticket %s...\n", ticketNumber)
	fmt.Printf("Comment: %s\n", comment)
	if changes != "" {
		fmt.Printf("Required changes: %s\n", changes)
	}
	fmt.Println()
	fmt.Println("Update requested. The ticket creator has been notified.")
}
