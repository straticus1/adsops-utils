package main

import (
	"os"

	"github.com/afterdarksys/adsops-utils/internal/cli/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
