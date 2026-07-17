package tui

import "fmt"

// Key binding constants
const (
	KeyUp     = "up"
	KeyDown   = "down"
	KeyK      = "k"
	KeyJ      = "j"
	KeyG      = "g"       // jump to top
	KeyShiftG = "G"       // jump to bottom
	KeyHome   = "home"
	KeyEnd    = "end"
	KeyPgUp   = "pgup"
	KeyPgDown = "pgdown"
	KeyEnter  = "enter"
	KeyC      = "c"       // checkout
	KeyY      = "y"       // confirm
	KeyN      = "n"       // cancel
	KeyEsc    = "escape"
	KeyQ      = "q"       // quit
	KeyCtrlC  = "ctrl+c"  // quit
)

// HelpText returns the formatted help bar text
func HelpText() string {
	return "  ↑/k Up  ↓/j Down  g/G Top/Bottom  enter Expand  c Checkout  q Quit"
}

// ConfirmText returns the checkout confirmation prompt
func ConfirmText(hash string) string {
	return fmt.Sprintf("  Checkout %s? (y/n)", hash)
}
