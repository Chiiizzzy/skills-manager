package main

import (
	"os"

	"github.com/your-org/skills-manager/internal/commands"
)

func main() {
	if err := commands.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
