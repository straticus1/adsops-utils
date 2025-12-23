package ticket

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List change tickets",
	Long: `List change tickets with optional filters.

Examples:
  # List all tickets
  changes ticket list

  # List only pending tickets
  changes ticket list --status submitted,in_review

  # List high priority tickets
  changes ticket list --priority high,urgent,emergency

  # List tickets assigned to you
  changes ticket list --mine

  # List tickets with JSON output
  changes ticket list --output json`,
	Run: runList,
}

func init() {
	listCmd.Flags().StringSlice("status", []string{}, "Filter by status")
	listCmd.Flags().StringSlice("priority", []string{}, "Filter by priority")
	listCmd.Flags().StringSlice("risk", []string{}, "Filter by risk level")
	listCmd.Flags().Bool("mine", false, "Show only tickets created by me")
	listCmd.Flags().Bool("assigned", false, "Show only tickets assigned to me")
	listCmd.Flags().Int("limit", 50, "Maximum number of tickets to display")
	listCmd.Flags().String("sort", "created_at", "Sort field (created_at, updated_at, priority)")
	listCmd.Flags().Bool("desc", true, "Sort descending")
}

func runList(cmd *cobra.Command, args []string) {
	// TODO: Implement API call to list tickets

	// Sample output format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TICKET\tSTATUS\tPRIORITY\tTITLE\tCREATED")
	fmt.Fprintln(w, "------\t------\t--------\t-----\t-------")
	fmt.Fprintln(w, "CHG-2025-00001\tsubmitted\thigh\tDatabase schema update\t2025-12-20")
	fmt.Fprintln(w, "CHG-2025-00002\tdraft\tnormal\tNetwork configuration change\t2025-12-21")
	fmt.Fprintln(w, "CHG-2025-00003\tapproved\turgent\tSecurity patch deployment\t2025-12-22")
	w.Flush()
}
