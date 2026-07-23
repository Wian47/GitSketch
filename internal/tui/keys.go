// internal/tui/keys.go
package tui

import (
	"fmt"

	"github.com/Wian47/GitSketch/internal/config"
)

// Key bindings. These are package-level vars, not consts, so ApplyKeyMap
// can override them from the user's loaded config at startup. Vim-style
// alternates (KeyK, KeyJ, KeyHome, KeyEnd) are always active and are not
// user-configurable.
var (
	KeyUp     = "up"
	KeyDown   = "down"
	KeyK      = "k"
	KeyJ      = "j"
	KeyG      = "g" // jump to top
	KeyShiftG = "G" // jump to bottom
	KeyHome   = "home"
	KeyEnd    = "end"
	KeyPgUp   = "pgup"
	KeyPgDown = "pgdown"
	KeyEnter  = "enter"
	KeyC      = "c" // checkout
	KeyY      = "y" // confirm
	KeyN      = "n" // cancel
	KeyEsc    = "escape"
	KeyQ      = "q"      // quit
	KeyCtrlC  = "ctrl+c" // quit
	KeyFilter = "/"
	KeyBranch = "b"
	KeyHelp   = "?"

	KeyStageFile   = "a"
	KeyUnstageFile = "u"
	KeyDiscard     = "x"
)

// ApplyKeyMap overrides the package-level key bindings from a loaded
// config.KeyMap. Empty fields are left untouched, so a config file only
// needs to specify the bindings it wants to change.
func ApplyKeyMap(km config.KeyMap) {
	setIfNotEmpty(&KeyUp, km.Up)
	setIfNotEmpty(&KeyDown, km.Down)
	setIfNotEmpty(&KeyG, km.Top)
	setIfNotEmpty(&KeyShiftG, km.Bottom)
	setIfNotEmpty(&KeyPgUp, km.PageUp)
	setIfNotEmpty(&KeyPgDown, km.PageDown)
	setIfNotEmpty(&KeyEnter, km.Enter)
	setIfNotEmpty(&KeyC, km.Checkout)
	setIfNotEmpty(&KeyFilter, km.Filter)
	setIfNotEmpty(&KeyBranch, km.Branch)
	setIfNotEmpty(&KeyHelp, km.Help)
	setIfNotEmpty(&KeyQ, km.Quit)
	setIfNotEmpty(&KeyStageFile, km.StageFile)
	setIfNotEmpty(&KeyUnstageFile, km.UnstageFile)
	setIfNotEmpty(&KeyDiscard, km.Discard)
}

func setIfNotEmpty(dst *string, val string) {
	if val != "" {
		*dst = val
	}
}

// HelpText returns the formatted help bar text
func HelpText() string {
	return "  ↑/k Up  ↓/j Down  g/G Top/Bottom  enter Expand  c Checkout  q Quit"
}

// ConfirmText returns the checkout confirmation prompt
func ConfirmText(hash string) string {
	return fmt.Sprintf("  Checkout %s? (y/n)", hash)
}
