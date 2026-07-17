package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"gitsketch/internal/git"
	"gitsketch/internal/graph"
)

// ─── Messages ───────────────────────────────────────────────────────────────

// commitsParsedMsg is sent when the initial git log parse completes.
type commitsParsedMsg struct {
	commits []git.Commit
	err     error
}

// filesLoadedMsg is sent when changed files for a commit are loaded.
type filesLoadedMsg struct {
	files []git.FileChange
	hash  string
	err   error
}

// checkoutDoneMsg is sent when a git checkout completes.
type checkoutDoneMsg struct {
	result git.CheckoutResult
}

// clearNotifyMsg clears the notification bar after a timeout.
type clearNotifyMsg struct{}

// ─── Model ──────────────────────────────────────────────────────────────────

// Model holds the entire application state for the Bubbletea program.
type Model struct {
	// Data
	commits   []git.Commit
	graphRows []graph.GraphRow
	files     []git.FileChange
	filesHash string // hash of the commit whose files are loaded

	// UI State
	cursor   int // index into graphRows of the selected commit
	scrollOff int // vertical scroll offset for the graph pane
	width    int // terminal width
	height   int // terminal height

	// Mode
	confirmCheckout bool   // showing checkout confirmation dialog
	notification    string // transient status message
	notifyStyle     lipgloss.Style

	// Loading
	loading    bool
	loadingMsg string

	// Error
	err error
}

// NewModel creates a new Model with initial loading state.
func NewModel() Model {
	return Model{
		loading:    true,
		loadingMsg: "Loading git history…",
	}
}

// ─── Bubbletea Interface ────────────────────────────────────────────────────

// Init returns the initial command to parse the git log.
func (m Model) Init() tea.Cmd {
	return parseCommitsCmd()
}

// Update handles all incoming messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case commitsParsedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.commits = msg.commits
		m.graphRows = graph.BuildGraph(m.commits)
		m.cursor = 0
		m.scrollOff = 0
		// Load files for the first commit.
		if len(m.graphRows) > 0 {
			if c := m.selectedCommit(); c != nil {
				return m, loadFilesCmd(c.Hash)
			}
		}
		return m, nil

	case filesLoadedMsg:
		if msg.err == nil && msg.hash != "" {
			m.files = msg.files
			m.filesHash = msg.hash
		}
		return m, nil

	case checkoutDoneMsg:
		m.confirmCheckout = false
		if msg.result.Success {
			m.notification = fmt.Sprintf(" ✓ Checked out %s", msg.result.Hash)
			m.notifyStyle = NotifySuccessStyle
		} else {
			m.notification = fmt.Sprintf(" ✗ %s", msg.result.Message)
			m.notifyStyle = NotifyErrorStyle
		}
		// Refresh the DAG and clear notification after delay.
		return m, tea.Batch(
			parseCommitsCmd(),
			clearNotifyAfter(3*time.Second),
		)

	case clearNotifyMsg:
		m.notification = ""
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// View renders the full TUI layout.
func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}

	var content string

	if m.err != nil {
		content = lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			NotifyErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)),
		)
	} else if m.loading {
		content = lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			DimStyle.Render(m.loadingMsg),
		)
	} else if len(m.graphRows) == 0 {
		content = lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			DimStyle.Render("No commits found."),
		)
	} else {
		content = m.renderLayout()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// ─── Key Handling ───────────────────────────────────────────────────────────

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// If in checkout confirmation mode, only accept y/n/esc.
	if m.confirmCheckout {
		switch key {
		case KeyY:
			c := m.selectedCommit()
			if c != nil {
				return m, checkoutCmd(c.Hash)
			}
			m.confirmCheckout = false
			return m, nil
		case KeyN, KeyEsc:
			m.confirmCheckout = false
			return m, nil
		}
		return m, nil
	}

	switch key {
	case KeyQ, KeyCtrlC:
		return m, func() tea.Msg { return tea.Quit() }

	case KeyUp, KeyK:
		m.moveCursor(-1)
		return m, m.loadFilesIfNeeded()

	case KeyDown, KeyJ:
		m.moveCursor(1)
		return m, m.loadFilesIfNeeded()

	case KeyPgUp:
		m.moveCursor(-m.visibleRows())
		return m, m.loadFilesIfNeeded()

	case KeyPgDown:
		m.moveCursor(m.visibleRows())
		return m, m.loadFilesIfNeeded()

	case KeyG, KeyHome:
		m.cursor = 0
		m.scrollOff = 0
		return m, m.loadFilesIfNeeded()

	case KeyShiftG, KeyEnd:
		m.cursor = m.commitRowCount() - 1
		m.adjustScroll()
		return m, m.loadFilesIfNeeded()

	case KeyC:
		c := m.selectedCommit()
		if c != nil {
			m.confirmCheckout = true
		}
		return m, nil
	}

	return m, nil
}

// ─── Cursor & Scroll ────────────────────────────────────────────────────────

// commitRowCount returns the number of rows that have actual commits (not connectors).
func (m Model) commitRowCount() int {
	count := 0
	for _, r := range m.graphRows {
		if r.Commit != nil {
			count++
		}
	}
	return count
}

// moveCursor moves the selection cursor by delta, clamping to valid commit rows.
func (m *Model) moveCursor(delta int) {
	total := m.commitRowCount()
	if total == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= total {
		m.cursor = total - 1
	}
	m.adjustScroll()
}

// adjustScroll ensures the cursor is visible in the viewport.
func (m *Model) adjustScroll() {
	visible := m.visibleRows()
	if visible <= 0 {
		return
	}
	// Find the actual row index for the cursor'th commit.
	rowIdx := m.cursorRowIndex()
	if rowIdx < m.scrollOff {
		m.scrollOff = rowIdx
	}
	if rowIdx >= m.scrollOff+visible {
		m.scrollOff = rowIdx - visible + 1
	}
}

// cursorRowIndex returns the graphRows index for the cursor'th commit.
func (m Model) cursorRowIndex() int {
	commitIdx := 0
	for i, r := range m.graphRows {
		if r.Commit != nil {
			if commitIdx == m.cursor {
				return i
			}
			commitIdx++
		}
	}
	return 0
}

// visibleRows returns how many graph rows fit in the graph pane.
func (m Model) visibleRows() int {
	// height minus 3 (title bar + help bar + border overhead)
	h := m.height - 4
	if h < 1 {
		h = 1
	}
	return h
}

// selectedCommit returns the commit under the cursor.
func (m Model) selectedCommit() *git.Commit {
	commitIdx := 0
	for i := range m.graphRows {
		if m.graphRows[i].Commit != nil {
			if commitIdx == m.cursor {
				return m.graphRows[i].Commit
			}
			commitIdx++
		}
	}
	return nil
}

// loadFilesIfNeeded returns a command to load files if the selection changed.
func (m Model) loadFilesIfNeeded() tea.Cmd {
	c := m.selectedCommit()
	if c == nil {
		return nil
	}
	if c.Hash == m.filesHash {
		return nil
	}
	return loadFilesCmd(c.Hash)
}

// ─── Rendering ──────────────────────────────────────────────────────────────

// renderLayout builds the full split-pane layout.
func (m Model) renderLayout() string {
	// Calculate pane widths.
	leftWidth := m.width * 60 / 100
	rightWidth := m.width - leftWidth

	// Available height: total minus 1 for help bar.
	paneHeight := m.height - 2

	// Render panes.
	leftContent := m.renderGraphPane(leftWidth-4, paneHeight-2) // account for border + padding
	rightContent := m.renderDetailPane(rightWidth-4, paneHeight-2)

	// Style the panes with borders.
	leftPane := lipgloss.NewStyle().
		Width(leftWidth - 2).
		Height(paneHeight - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PaneBorderColor).
		Padding(0, 1).
		Render(leftContent)

	rightPane := lipgloss.NewStyle().
		Width(rightWidth - 2).
		Height(paneHeight - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PaneBorderColor).
		Padding(0, 1).
		Render(rightContent)

	// Join panes horizontally.
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Build help/status bar.
	helpBar := m.renderHelpBar()

	return lipgloss.JoinVertical(lipgloss.Left, mainView, helpBar)
}

// renderGraphPane renders the scrollable ASCII graph with commit metadata.
func (m Model) renderGraphPane(width, height int) string {
	if len(m.graphRows) == 0 {
		return DimStyle.Render("No commits.")
	}

	maxGraphCols := graph.MaxColumns(m.graphRows)
	graphWidth := maxGraphCols*2 + 1 // each cell is char + space
	metaWidth := width - graphWidth - 2
	if metaWidth < 10 {
		metaWidth = 10
	}

	var lines []string
	commitIdx := 0

	// Only render visible rows (scroll window).
	endRow := m.scrollOff + height
	if endRow > len(m.graphRows) {
		endRow = len(m.graphRows)
	}

	for rowIdx := m.scrollOff; rowIdx < endRow; rowIdx++ {
		row := m.graphRows[rowIdx]

		// Build the graph prefix.
		graphStr := m.renderGraphCells(row, maxGraphCols)

		if row.Commit == nil {
			// Connector row — just the graph lines.
			lines = append(lines, graphStr)
			continue
		}

		// Is this the selected commit?
		// We need to track commit indices across all rows, not just visible ones.
		// Recalculate: count commits before scrollOff.
		actualCommitIdx := 0
		for ci := 0; ci < rowIdx; ci++ {
			if m.graphRows[ci].Commit != nil {
				actualCommitIdx++
			}
		}
		isSelected := actualCommitIdx == m.cursor

		// Build the metadata portion.
		meta := m.renderCommitMeta(row.Commit, metaWidth)

		// Combine graph + metadata.
		line := graphStr + "  " + meta

		if isSelected {
			line = SelectedRowStyle.Width(width).Render(line)
		}

		lines = append(lines, line)
		commitIdx++
	}

	return strings.Join(lines, "\n")
}

// renderGraphCells converts a row's cells into a styled string.
func (m Model) renderGraphCells(row graph.GraphRow, maxCols int) string {
	var sb strings.Builder
	for i := 0; i < maxCols; i++ {
		if i > 0 {
			sb.WriteRune(' ')
		}
		if i < len(row.Cells) {
			cell := row.Cells[i]
			charStr := string(cell.Char)
			switch cell.Char {
			case '*':
				colorIdx := cell.Color % len(BranchColors)
				style := lipgloss.NewStyle().
					Foreground(BranchColors[colorIdx]).
					Bold(true)
				sb.WriteString(style.Render("●"))
			case '│':
				colorIdx := cell.Color % len(BranchColors)
				style := lipgloss.NewStyle().
					Foreground(BranchColors[colorIdx])
				sb.WriteString(style.Render("│"))
			case '/', '\\':
				colorIdx := cell.Color % len(BranchColors)
				style := lipgloss.NewStyle().
					Foreground(BranchColors[colorIdx])
				sb.WriteString(style.Render(charStr))
			default:
				sb.WriteRune(' ')
			}
		} else {
			sb.WriteRune(' ')
		}
	}
	return sb.String()
}

// renderCommitMeta renders the inline commit metadata (refs, hash, subject, date).
func (m Model) renderCommitMeta(c *git.Commit, width int) string {
	var parts []string

	// Render ref badges.
	for _, ref := range c.Refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if strings.Contains(ref, "HEAD") {
			parts = append(parts, HeadRefStyle.Render(ref))
		} else if strings.HasPrefix(ref, "tag:") {
			parts = append(parts, TagRefStyle.Render(strings.TrimPrefix(ref, "tag: ")))
		} else {
			parts = append(parts, BranchRefStyle.Render(ref))
		}
	}

	// Hash.
	parts = append(parts, HashStyle.Render(c.ShortHash))

	// Subject (truncated to fit).
	subject := c.Subject
	usedWidth := 0
	for _, p := range parts {
		usedWidth += lipgloss.Width(p) + 1
	}
	// Reserve space for date.
	dateStr := c.RelDate
	remaining := width - usedWidth - lipgloss.Width(dateStr) - 2
	if remaining < 0 {
		remaining = 10
	}
	if len(subject) > remaining {
		if remaining > 3 {
			subject = subject[:remaining-3] + "…"
		} else {
			subject = subject[:remaining]
		}
	}
	parts = append(parts, SubjectStyle.Render(subject))

	// Date (right-aligned effect via padding).
	parts = append(parts, DateStyle.Render(dateStr))

	return strings.Join(parts, " ")
}

// renderDetailPane renders the right-side detail pane for the selected commit.
func (m Model) renderDetailPane(width, height int) string {
	c := m.selectedCommit()
	if c == nil {
		return DimStyle.Render("No commit selected.")
	}

	var lines []string

	// Title.
	lines = append(lines, SectionHeaderStyle.Render("Commit Details"))
	lines = append(lines, "")

	// Metadata.
	lines = append(lines, DetailLabelStyle.Render("Commit  ")+HashStyle.Render(c.Hash))
	lines = append(lines, DetailLabelStyle.Render("Author  ")+AuthorStyle.Render(c.Author)+" "+DimStyle.Render("<"+c.Email+">"))
	lines = append(lines, DetailLabelStyle.Render("Date    ")+DateStyle.Render(c.RelDate))

	// Refs.
	if len(c.Refs) > 0 {
		var refBadges []string
		for _, ref := range c.Refs {
			ref = strings.TrimSpace(ref)
			if ref == "" {
				continue
			}
			if strings.Contains(ref, "HEAD") {
				refBadges = append(refBadges, HeadRefStyle.Render(ref))
			} else if strings.HasPrefix(ref, "tag:") {
				refBadges = append(refBadges, TagRefStyle.Render(strings.TrimPrefix(ref, "tag: ")))
			} else {
				refBadges = append(refBadges, BranchRefStyle.Render(ref))
			}
		}
		lines = append(lines, DetailLabelStyle.Render("Refs    ")+strings.Join(refBadges, " "))
	}

	// Parents.
	if len(c.Parents) > 0 {
		parentStrs := make([]string, len(c.Parents))
		for i, p := range c.Parents {
			short := p
			if len(short) > 7 {
				short = short[:7]
			}
			parentStrs[i] = HashStyle.Render(short)
		}
		lines = append(lines, DetailLabelStyle.Render("Parents ")+strings.Join(parentStrs, " "))
	}

	lines = append(lines, "")

	// Subject & body.
	lines = append(lines, SectionHeaderStyle.Render("Message"))
	lines = append(lines, "")
	lines = append(lines, SubjectStyle.Render(c.Subject))
	if c.Body != "" {
		lines = append(lines, "")
		// Wrap body text.
		bodyLines := strings.Split(c.Body, "\n")
		for _, bl := range bodyLines {
			if len(bl) > width {
				bl = bl[:width-1] + "…"
			}
			lines = append(lines, BodyStyle.Render(bl))
		}
	}

	// Changed files.
	if len(m.files) > 0 && m.filesHash == c.Hash {
		lines = append(lines, "")
		lines = append(lines, SectionHeaderStyle.Render(fmt.Sprintf("Changed Files (%d)", len(m.files))))
		lines = append(lines, "")

		maxFiles := height - len(lines) - 1
		if maxFiles < 1 {
			maxFiles = 1
		}
		for i, f := range m.files {
			if i >= maxFiles {
				remaining := len(m.files) - maxFiles
				lines = append(lines, DimStyle.Render(fmt.Sprintf("  … and %d more", remaining)))
				break
			}
			var statusStyle lipgloss.Style
			switch f.Status {
			case "M":
				statusStyle = FileModifiedStyle
			case "A":
				statusStyle = FileAddedStyle
			case "D":
				statusStyle = FileDeletedStyle
			default:
				statusStyle = DimStyle
			}
			filePath := f.Path
			if len(filePath) > width-6 {
				filePath = "…" + filePath[len(filePath)-(width-7):]
			}
			lines = append(lines, "  "+statusStyle.Render(f.Status)+" "+DimStyle.Render(filePath))
		}
	} else if m.filesHash != c.Hash {
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render("Loading files…"))
	}

	return strings.Join(lines, "\n")
}

// renderHelpBar renders the bottom status/help bar.
func (m Model) renderHelpBar() string {
	var text string

	if m.confirmCheckout {
		c := m.selectedCommit()
		hash := "unknown"
		if c != nil {
			hash = c.ShortHash
		}
		text = NotifyErrorStyle.Render(ConfirmText(hash))
	} else if m.notification != "" {
		text = m.notifyStyle.Render(m.notification)
	} else {
		text = HelpText()
	}

	return HelpBarStyle.Width(m.width).Render(text)
}

// ─── Commands ───────────────────────────────────────────────────────────────

// parseCommitsCmd returns a tea.Cmd that parses the git log.
func parseCommitsCmd() tea.Cmd {
	return func() tea.Msg {
		commits, err := git.ParseLog()
		return commitsParsedMsg{commits: commits, err: err}
	}
}

// loadFilesCmd returns a tea.Cmd that loads changed files for a commit.
func loadFilesCmd(hash string) tea.Cmd {
	return func() tea.Msg {
		files, err := git.GetChangedFiles(hash)
		return filesLoadedMsg{files: files, hash: hash, err: err}
	}
}

// checkoutCmd returns a tea.Cmd that runs git checkout.
func checkoutCmd(hash string) tea.Cmd {
	return func() tea.Msg {
		result := git.Checkout(hash)
		return checkoutDoneMsg{result: result}
	}
}

// clearNotifyAfter returns a tea.Cmd that clears the notification after a delay.
func clearNotifyAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearNotifyMsg{}
	})
}
// TODO: implement search filter
