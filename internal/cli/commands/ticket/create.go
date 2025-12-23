package ticket

import (
	"fmt"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new change ticket",
	Long: `Create a new change ticket either interactively or by providing flags.

Interactive mode will guide you through the ticket creation process with
industry-specific compliance requirements.

Examples:
  # Create interactively
  changes ticket create

  # Create with minimal flags (will prompt for required fields)
  changes ticket create --title "Database migration"

  # Create with all required fields
  changes ticket create \
    --title "Update production database schema" \
    --description "Adding new index for performance" \
    --priority normal \
    --risk medium \
    --industry finance \
    --compliance glba,sox \
    --approval-types operations,it,security`,
	Run: runCreate,
}

func init() {
	createCmd.Flags().String("title", "", "Ticket title (required)")
	createCmd.Flags().String("description", "", "Ticket description")
	createCmd.Flags().StringP("priority", "p", "normal", "Priority (emergency, urgent, high, normal, low)")
	createCmd.Flags().StringP("risk", "r", "medium", "Risk level (critical, high, medium, low)")
	createCmd.Flags().StringP("industry", "i", "", "Industry (healthcare, it, government, insurance, finance)")
	createCmd.Flags().StringSlice("compliance", []string{}, "Compliance frameworks (glba, sox, hipaa, gdpr, banking_secrecy_act)")
	createCmd.Flags().StringSlice("approval-types", []string{}, "Required approval types")
	createCmd.Flags().StringSlice("affected-systems", []string{}, "Affected systems")
	createCmd.Flags().String("change-type", "", "Type of change")
	createCmd.Flags().String("impact", "", "Impact description")
	createCmd.Flags().String("rollback", "", "Rollback plan")
	createCmd.Flags().String("testing", "", "Testing plan")
	createCmd.Flags().Bool("submit", false, "Submit immediately instead of saving as draft")
	createCmd.Flags().Bool("interactive", true, "Use interactive mode")
}

func runCreate(cmd *cobra.Command, args []string) {
	interactive, _ := cmd.Flags().GetBool("interactive")

	if interactive {
		runInteractiveCreate(cmd)
		return
	}

	// Non-interactive creation
	title, _ := cmd.Flags().GetString("title")
	if title == "" {
		fmt.Println("Error: --title is required in non-interactive mode")
		return
	}

	// TODO: Implement API call to create ticket
	fmt.Printf("Creating ticket: %s\n", title)
	fmt.Println("Ticket created successfully: CHG-2025-00001")
}

func runInteractiveCreate(cmd *cobra.Command) {
	// TODO: Implement interactive TUI using bubbletea
	fmt.Println("Interactive ticket creation")
	fmt.Println("===========================")
	fmt.Println()
	fmt.Println("This will launch an interactive form to create a new ticket.")
	fmt.Println("(TUI implementation pending)")
}
