package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/Wian47/GitSketch/internal/git"
	"github.com/Wian47/GitSketch/internal/tui"
)

func main() {
	// Verify we're inside a git repository.
	if !git.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "fatal: not a git repository (or any of the parent directories)")
		os.Exit(1)
	}

	// Initialize and run the Bubbletea program.
	model := tui.NewModel()
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running GitSketch: %v\n", err)
		os.Exit(1)
	}
}
