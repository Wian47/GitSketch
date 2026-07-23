package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Wian47/GitSketch/internal/git"
	"github.com/Wian47/GitSketch/internal/graph"
)

// filterDebounceDelay controls how long to wait after the last keystroke in
// the filter box before recomputing the filtered commit list and graph. This
// avoids rebuilding the entire graph on every keypress for large histories.
const filterDebounceDelay = 120 * time.Millisecond

// ─── Messages ───────────────────────────────────────────────────────────────

type commitsParsedMsg struct {
	commits []git.Commit
	err     error
}

type filesLoadedMsg struct {
	files []git.FileChange
	hash  string
	err   error
}

type diffLoadedMsg struct {
	hash string
	diff string
	err  error
}

type checkoutDoneMsg struct {
	result git.CheckoutResult
}

type commitDoneMsg struct {
	result git.CommitResult
}

type statusLoadedMsg struct {
	status git.Status
	err    error
}

type clearNotifyMsg struct{}

type filterDebounceMsg struct{ gen int }

type stagingDoneMsg struct {
	action string // "staged", "unstaged", or "discarded"
	path   string
	err    error
}

type stagingDiffLoadedMsg struct {
	path   string
	staged bool
	header string
	hunks  []git.Hunk
	err    error
}

type hunkStagingDoneMsg struct {
	path   string
	staged bool // which view (staged/unstaged) we were operating against
	err    error
}

// ─── Model ──────────────────────────────────────────────────────────────────

// Model holds the entire application state for the Bubbletea program.
type Model struct {
	// Data
	allCommits      []git.Commit
	filteredCommits []git.Commit
	graphRows       []graph.GraphRow
	files           []git.FileChange
	filesHash       string     // hash of the commit whose files are loaded
	wtStatus        git.Status // last-loaded working tree status (staged/unstaged files)

	// Status bar state
	repoBranch    string
	repoAhead     int
	repoBehind    int
	dirtyStaged   int
	dirtyUnstaged int

	// Fullscreen Diff View State
	showDiff    bool
	diffContent string
	diffScroll  int

	stagingDiffMode   bool // true when the fullscreen view (showDiff) is hunk-staging mode, not a plain commit diff
	stagingFileHeader string
	stagingHunks      []git.Hunk
	stagingHunkCursor int
	stagingScroll     int // first visible line within the rendered hunk body, kept in sync with stagingHunkCursor
	stagingFilePath   string
	stagingFileStaged bool // whether the currently-open diff is the staged or unstaged view of stagingFilePath

	// Search / Filter State
	searchMode  bool
	searchQuery string
	filterGen   int // incremented per keystroke; guards debounced recompute

	// historyLoaded tracks whether the initial commit load has completed, so
	// later refreshes (after checkout/branch actions) can preserve cursor
	// position instead of resetting to the top every time.
	historyLoaded bool

	// Branch Manager State
	branchMode    bool
	branchSubMode string // "", "create", "delete"
	branchInput   string

	// UI State
	cursor    int  // index into filteredCommits of the selected commit
	scrollOff int  // vertical scroll offset for the graph pane
	width     int  // terminal width
	height    int  // terminal height
	helpMode  bool // showing the full keybinding help overlay

	wtSelected   bool // true when focus is on the working-tree row/file list, not a commit
	wtFileCursor int  // index into wtFileEntries() when wtSelected is true

	// Mode
	confirmCheckout bool   // showing checkout confirmation dialog
	confirmDiscard  bool   // showing discard-changes confirmation for the focused working-tree file
	notification    string // transient status message
	notifyStyle     lipgloss.Style
	commitInputMode bool
	commitMessage   string

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
	return tea.Batch(parseCommitsCmd(), loadStatusCmd())
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
		m.allCommits = msg.commits
		m.applyFilter()
		// Only reset the cursor to the top on the very first load. Checkout
		// and branch actions re-parse the log to refresh ref decorations,
		// but don't change commit order/count, so keep the cursor where the
		// user left it instead of jumping back to the top each time.
		if !m.historyLoaded {
			m.cursor = 0
			m.scrollOff = 0
			m.historyLoaded = true
		}
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

	case statusLoadedMsg:
		if msg.err == nil {
			m.repoBranch = msg.status.Branch
			m.repoAhead = msg.status.Ahead
			m.repoBehind = msg.status.Behind
			m.dirtyStaged = len(msg.status.Staged)
			m.dirtyUnstaged = len(msg.status.Unstaged)
			m.wtStatus = msg.status
			total := m.dirtyStaged + m.dirtyUnstaged
			if m.wtFileCursor >= total {
				m.wtFileCursor = total - 1
			}
			if m.wtFileCursor < 0 {
				m.wtFileCursor = 0
			}
		}
		return m, nil

	case diffLoadedMsg:
		if msg.err == nil && msg.hash != "" {
			m.diffContent = msg.diff
			m.diffScroll = 0
		} else if msg.err != nil {
			m.diffContent = fmt.Sprintf("Error loading diff: %v", msg.err)
		}
		return m, nil

	case stagingDiffLoadedMsg:
		if msg.err != nil {
			m.notification = fmt.Sprintf(" ✗ Error loading diff: %v", msg.err)
			m.notifyStyle = NotifyErrorStyle
			return m, clearNotifyAfter(3 * time.Second)
		}
		m.stagingFileHeader = msg.header
		m.stagingHunks = msg.hunks
		if m.stagingHunkCursor >= len(m.stagingHunks) {
			m.stagingHunkCursor = 0
		}
		m.adjustStagingScroll()
		return m, nil

	case hunkStagingDoneMsg:
		if msg.err != nil {
			m.notification = fmt.Sprintf(" ✗ %s", msg.err.Error())
			m.notifyStyle = NotifyErrorStyle
			return m, clearNotifyAfter(3 * time.Second)
		}
		action := "staged"
		if msg.staged {
			action = "unstaged"
		}
		m.notification = fmt.Sprintf(" ✓ hunk %s in %s", action, msg.path)
		m.notifyStyle = NotifySuccessStyle
		return m, tea.Batch(
			loadStagingDiffCmd(msg.path, msg.staged),
			loadStatusCmd(),
			clearNotifyAfter(3*time.Second),
		)

	case checkoutDoneMsg:
		m.confirmCheckout = false
		if msg.result.Success {
			m.notification = msg.result.Message
			m.notifyStyle = NotifySuccessStyle
		} else {
			m.notification = fmt.Sprintf(" ✗ %s", msg.result.Message)
			m.notifyStyle = NotifyErrorStyle
		}
		// Refresh the DAG and clear notification after delay.
		return m, tea.Batch(
			parseCommitsCmd(),
			loadStatusCmd(),
			clearNotifyAfter(3*time.Second),
		)

	case commitDoneMsg:
		if msg.result.Success {
			text := msg.result.Message
			if text == "" {
				text = "Committed"
			}
			m.notification = " ✓ " + text
			m.notifyStyle = NotifySuccessStyle
		} else {
			m.notification = fmt.Sprintf(" ✗ %s", msg.result.Message)
			m.notifyStyle = NotifyErrorStyle
		}
		return m, tea.Batch(
			parseCommitsCmd(),
			loadStatusCmd(),
			clearNotifyAfter(3*time.Second),
		)

	case stagingDoneMsg:
		if msg.err != nil {
			m.notification = fmt.Sprintf(" ✗ %s", msg.err.Error())
			m.notifyStyle = NotifyErrorStyle
		} else {
			m.notification = fmt.Sprintf(" ✓ %s %s", msg.action, msg.path)
			m.notifyStyle = NotifySuccessStyle
		}
		return m, tea.Batch(loadStatusCmd(), clearNotifyAfter(3*time.Second))

	case clearNotifyMsg:
		m.notification = ""
		return m, nil

	case filterDebounceMsg:
		if msg.gen == m.filterGen {
			m.applyFilter()
		}
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
	} else if m.helpMode {
		content = m.renderHelpOverlay()
	} else if m.showDiff {
		if m.stagingDiffMode {
			content = m.renderStagingDiffView()
		} else {
			content = m.renderDiffView()
		}
	} else if len(m.graphRows) == 0 {
		content = lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			DimStyle.Render("No matching commits found."),
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

	// ── Mode: Fullscreen Diff Viewer ──
	if m.showDiff {
		if m.stagingDiffMode {
			switch key {
			case KeyEsc, KeyQ:
				m.showDiff = false
				m.stagingDiffMode = false
				m.stagingHunks = nil
				return m, nil
			case KeyUp, KeyK:
				if m.stagingHunkCursor > 0 {
					m.stagingHunkCursor--
				}
				m.adjustStagingScroll()
				return m, nil
			case KeyDown, KeyJ:
				if m.stagingHunkCursor < len(m.stagingHunks)-1 {
					m.stagingHunkCursor++
				}
				m.adjustStagingScroll()
				return m, nil
			case "space":
				return m, m.toggleSelectedHunk()
			}
			return m, nil
		}
		switch key {
		case KeyEsc, KeyEnter, KeyQ:
			m.showDiff = false
			m.diffContent = ""
			return m, nil
		case KeyUp, KeyK:
			if m.diffScroll > 0 {
				m.diffScroll--
			}
			return m, nil
		case KeyDown, KeyJ:
			m.diffScroll++
			return m, nil
		case KeyPgUp:
			m.diffScroll -= m.height - 4
			if m.diffScroll < 0 {
				m.diffScroll = 0
			}
			return m, nil
		case KeyPgDown:
			m.diffScroll += m.height - 4
			return m, nil
		}
		return m, nil
	}

	// ── Mode: Help Overlay ──
	if m.helpMode {
		switch key {
		case KeyEsc, KeyHelp, KeyQ:
			m.helpMode = false
			return m, nil
		}
		return m, nil
	}

	// ── Mode: Search / Filter Input ──
	if m.searchMode {
		switch key {
		case KeyEsc:
			m.searchMode = false
			m.searchQuery = ""
			m.filterGen++ // invalidate any pending debounced recompute
			m.applyFilter()
			return m, nil
		case KeyEnter:
			m.searchMode = false
			m.filterGen++ // invalidate any pending debounced recompute
			m.applyFilter()
			return m, nil
		case "backspace":
			if len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				m.filterGen++
				return m, filterDebounceCmd(m.filterGen)
			}
			return m, nil
		default:
			// Capture printable characters
			if len(key) == 1 {
				m.searchQuery += key
				m.filterGen++
				return m, filterDebounceCmd(m.filterGen)
			} else if key == "space" {
				m.searchQuery += " "
				m.filterGen++
				return m, filterDebounceCmd(m.filterGen)
			}
			return m, nil
		}
	}

	// ── Mode: Branch Manager ──
	if m.branchMode {
		if m.branchSubMode == "" {
			switch key {
			case "c":
				m.branchSubMode = "create"
				m.branchInput = ""
				return m, nil
			case "d":
				m.branchSubMode = "delete"
				m.branchInput = ""
				return m, nil
			case KeyEsc:
				m.branchMode = false
				return m, nil
			}
			return m, nil
		} else {
			// Capturing branch input name
			switch key {
			case KeyEsc:
				m.branchSubMode = ""
				return m, nil
			case KeyEnter:
				if m.branchInput == "" {
					m.branchSubMode = ""
					return m, nil
				}
				cmd := m.executeBranchAction()
				m.branchMode = false
				m.branchSubMode = ""
				return m, cmd
			case "backspace":
				if len(m.branchInput) > 0 {
					m.branchInput = m.branchInput[:len(m.branchInput)-1]
				}
				return m, nil
			default:
				if len(key) == 1 {
					m.branchInput += key
				} else if key == "space" {
					m.branchInput += " "
				}
				return m, nil
			}
		}
	}

	// ── Mode: Commit Message Input ──
	if m.commitInputMode {
		switch key {
		case KeyEsc:
			m.commitInputMode = false
			m.commitMessage = ""
			return m, nil
		case KeyEnter:
			if m.commitMessage == "" {
				return m, nil
			}
			msg := m.commitMessage
			m.commitInputMode = false
			m.commitMessage = ""
			return m, commitCmd(msg)
		case "backspace":
			if len(m.commitMessage) > 0 {
				m.commitMessage = m.commitMessage[:len(m.commitMessage)-1]
			}
			return m, nil
		default:
			if len(key) == 1 {
				m.commitMessage += key
			} else if key == "space" {
				m.commitMessage += " "
			}
			return m, nil
		}
	}

	// ── Mode: Checkout Confirmation ──
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

	// ── Mode: Discard Confirmation ──
	if m.confirmDiscard {
		switch key {
		case KeyY:
			m.confirmDiscard = false
			if ref, ok := m.selectedWorkingTreeFile(); ok {
				return m, discardFileCmd(ref.entry.Path)
			}
			return m, nil
		case KeyN, KeyEsc:
			m.confirmDiscard = false
			return m, nil
		}
		return m, nil
	}

	// ── Normal Mode ──
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
		m.wtSelected = false
		m.cursor = 0
		m.scrollOff = 0
		return m, m.loadFilesIfNeeded()

	case KeyShiftG, KeyEnd:
		m.wtSelected = false
		m.cursor = m.commitRowCount() - 1
		m.adjustScroll()
		return m, m.loadFilesIfNeeded()

	case KeyC:
		if m.wtSelected {
			if m.dirtyStaged == 0 {
				m.notification = " ✗ Nothing staged to commit"
				m.notifyStyle = NotifyErrorStyle
				return m, clearNotifyAfter(3 * time.Second)
			}
			m.commitInputMode = true
			m.commitMessage = ""
			return m, nil
		}
		c := m.selectedCommit()
		if c != nil {
			m.confirmCheckout = true
		}
		return m, nil

	case KeyStageFile:
		if m.wtSelected {
			return m, m.stageSelectedFile()
		}
		return m, nil

	case KeyUnstageFile:
		if m.wtSelected {
			return m, m.unstageSelectedFile()
		}
		return m, nil

	case KeyDiscard:
		if m.wtSelected {
			if ref, ok := m.selectedWorkingTreeFile(); ok && !ref.staged {
				m.confirmDiscard = true
			}
		}
		return m, nil

	case KeyFilter:
		m.searchMode = true
		m.searchQuery = ""
		return m, nil

	case KeyBranch:
		m.branchMode = true
		m.branchSubMode = ""
		return m, nil

	case KeyHelp:
		m.helpMode = true
		return m, nil

	case KeyEnter:
		if m.wtSelected {
			return m, m.openStagingDiff()
		}
		c := m.selectedCommit()
		if c != nil {
			m.showDiff = true
			m.diffContent = ""
			m.diffScroll = 0
			return m, loadDiffCmd(c.Hash)
		}
		return m, nil
	}

	return m, nil
}

// ─── Filter & Search Logic ──────────────────────────────────────────────────

func (m *Model) applyFilter() {
	if m.searchQuery == "" {
		m.filteredCommits = m.allCommits
	} else {
		m.filteredCommits = nil
		matches := filterMatcher(m.searchQuery)
		for _, c := range m.allCommits {
			match := matches(c.Hash) || matches(c.Subject) || matches(c.Author) || matches(c.Body)

			if !match {
				for _, r := range c.Refs {
					if matches(r) {
						match = true
						break
					}
				}
			}
			if match {
				m.filteredCommits = append(m.filteredCommits, c)
			}
		}
	}
	m.graphRows = graph.BuildGraph(m.filteredCommits)

	// Clamp cursor to valid filtered range.
	total := m.commitRowCount()
	if m.cursor >= total {
		m.cursor = total - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.adjustScroll()
}

// filterMatcher compiles the query as a case-insensitive regular expression
// and returns a matcher function. If the query isn't a valid regex (e.g. an
// unbalanced "(" typed mid-pattern), it falls back to a plain case-insensitive
// substring match so the filter never goes "dead" while the user is typing.
func filterMatcher(query string) func(string) bool {
	if re, err := regexp.Compile("(?i)" + query); err == nil {
		return re.MatchString
	}
	lowerQuery := strings.ToLower(query)
	return func(s string) bool {
		return strings.Contains(strings.ToLower(s), lowerQuery)
	}
}

// ─── Cursor & Scroll ────────────────────────────────────────────────────────

func (m Model) commitRowCount() int {
	count := 0
	for _, r := range m.graphRows {
		if r.Commit != nil {
			count++
		}
	}
	return count
}

// wtFileRef pairs a working-tree status entry with whether it came from the
// staged or unstaged list, so callers can tell them apart after they've
// been combined into one navigable sequence.
type wtFileRef struct {
	entry  git.StatusEntry
	staged bool
}

// wtFileEntries returns the working tree's staged files followed by its
// unstaged files, as one combined, indexable sequence — this ordering is
// what wtFileCursor indexes into.
func (m Model) wtFileEntries() []wtFileRef {
	entries := make([]wtFileRef, 0, len(m.wtStatus.Staged)+len(m.wtStatus.Unstaged))
	for _, e := range m.wtStatus.Staged {
		entries = append(entries, wtFileRef{entry: e, staged: true})
	}
	for _, e := range m.wtStatus.Unstaged {
		entries = append(entries, wtFileRef{entry: e, staged: false})
	}
	return entries
}

// selectedWorkingTreeFile returns the file at wtFileCursor, or false if the
// working tree is clean (or the cursor is out of range).
func (m Model) selectedWorkingTreeFile() (wtFileRef, bool) {
	entries := m.wtFileEntries()
	if m.wtFileCursor < 0 || m.wtFileCursor >= len(entries) {
		return wtFileRef{}, false
	}
	return entries[m.wtFileCursor], true
}

// stageSelectedFile stages the working-tree file under the cursor, or does
// nothing if it's already staged (or nothing is selected).
func (m Model) stageSelectedFile() tea.Cmd {
	ref, ok := m.selectedWorkingTreeFile()
	if !ok || ref.staged {
		return nil
	}
	return stageFileCmd(ref.entry.Path)
}

// unstageSelectedFile unstages the working-tree file under the cursor, or
// does nothing if it's already unstaged (or nothing is selected).
func (m Model) unstageSelectedFile() tea.Cmd {
	ref, ok := m.selectedWorkingTreeFile()
	if !ok || !ref.staged {
		return nil
	}
	return unstageFileCmd(ref.entry.Path)
}

// openStagingDiff opens the fullscreen hunk-staging view for whichever
// working-tree file is currently focused. Returns nil (no-op) if the
// working tree is clean.
func (m *Model) openStagingDiff() tea.Cmd {
	ref, ok := m.selectedWorkingTreeFile()
	if !ok {
		return nil
	}
	m.showDiff = true
	m.stagingDiffMode = true
	m.stagingFilePath = ref.entry.Path
	m.stagingFileStaged = ref.staged
	m.stagingHunkCursor = 0
	m.stagingScroll = 0
	return loadStagingDiffCmd(ref.entry.Path, ref.staged)
}

// stagingHunkLineRanges returns the [start, end) line range each hunk in
// m.stagingHunks occupies within the rendered hunk body (the content area
// of renderStagingDiffView, before its title/spacer lines).
func (m Model) stagingHunkLineRanges() []struct{ start, end int } {
	ranges := make([]struct{ start, end int }, len(m.stagingHunks))
	pos := 0
	for i, h := range m.stagingHunks {
		ranges[i].start = pos
		pos += len(h.Lines)
		ranges[i].end = pos
	}
	return ranges
}

// stagingViewportHeight returns the number of hunk-body lines visible at
// once in the fullscreen staging diff view (mirrors renderDiffView's
// visibleHeight: reserves rows for the title, spacer, and help bar).
func (m Model) stagingViewportHeight() int {
	h := m.height - 3
	if h < 1 {
		h = 1
	}
	return h
}

// adjustStagingScroll keeps the hunk under stagingHunkCursor within the
// visible scroll window, nudging stagingScroll only as far as needed.
func (m *Model) adjustStagingScroll() {
	ranges := m.stagingHunkLineRanges()
	if m.stagingHunkCursor < 0 || m.stagingHunkCursor >= len(ranges) {
		return
	}
	viewport := m.stagingViewportHeight()
	r := ranges[m.stagingHunkCursor]
	if r.start < m.stagingScroll {
		m.stagingScroll = r.start
	}
	if r.end > m.stagingScroll+viewport {
		m.stagingScroll = r.end - viewport
	}
	if m.stagingScroll < 0 {
		m.stagingScroll = 0
	}
}

// toggleSelectedHunk stages the hunk under the cursor if we're viewing the
// unstaged diff, or unstages it if we're viewing the staged diff.
func (m Model) toggleSelectedHunk() tea.Cmd {
	if m.stagingHunkCursor < 0 || m.stagingHunkCursor >= len(m.stagingHunks) {
		return nil
	}
	hunk := m.stagingHunks[m.stagingHunkCursor]
	header := m.stagingFileHeader
	path := m.stagingFilePath
	staged := m.stagingFileStaged
	return func() tea.Msg {
		var err error
		if staged {
			err = git.UnstageHunk(header, hunk)
		} else {
			err = git.StageHunk(header, hunk)
		}
		return hunkStagingDoneMsg{path: path, staged: staged, err: err}
	}
}

func (m *Model) moveCursor(delta int) {
	if m.wtSelected {
		total := len(m.wtStatus.Staged) + len(m.wtStatus.Unstaged)
		if total == 0 {
			if delta > 0 {
				m.wtSelected = false
				m.cursor = 0
				m.adjustScroll()
			}
			return
		}
		newCursor := m.wtFileCursor + delta
		if newCursor < 0 {
			return // already at the top file — nothing above the working tree row
		}
		if newCursor >= total {
			m.wtSelected = false
			m.cursor = 0
			m.adjustScroll()
			return
		}
		m.wtFileCursor = newCursor
		return
	}

	total := m.commitRowCount()
	if total == 0 {
		return
	}
	if m.cursor == 0 && delta == -1 {
		m.wtSelected = true
		m.wtFileCursor = 0
		m.adjustScroll()
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

func (m *Model) adjustScroll() {
	visible := m.visibleRows()
	if visible <= 0 {
		return
	}
	rowIdx := m.cursorRowIndex()
	if rowIdx < m.scrollOff {
		m.scrollOff = rowIdx
	}
	if rowIdx >= m.scrollOff+visible {
		m.scrollOff = rowIdx - visible + 1
	}
}

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

func (m Model) visibleRows() int {
	h := m.height - 4
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) selectedCommit() *git.Commit {
	if m.wtSelected {
		return nil
	}
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

func (m Model) renderLayout() string {
	leftWidth := m.width * 60 / 100
	rightWidth := m.width - leftWidth

	paneHeight := m.height - 3

	leftContent := m.renderGraphPane(leftWidth-4, paneHeight-2)
	rightContent := m.renderDetailPane(rightWidth-4, paneHeight-2)

	leftPane := lipgloss.NewStyle().
		Width(leftWidth-2).
		Height(paneHeight-2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PaneBorderColor).
		Padding(0, 1).
		Render(leftContent)

	rightPane := lipgloss.NewStyle().
		Width(rightWidth-2).
		Height(paneHeight-2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PaneBorderColor).
		Padding(0, 1).
		Render(rightContent)

	mainView := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	statusBar := renderStatusBar(m.width, m.repoBranch, m.repoAhead, m.repoBehind, m.dirtyStaged, m.dirtyUnstaged)
	helpBar := m.renderHelpBar()

	return lipgloss.JoinVertical(lipgloss.Left, statusBar, mainView, helpBar)
}

func (m Model) renderGraphPane(width, height int) string {
	wtRow := m.renderWorkingTreeRow(width)
	height--
	if height < 0 {
		height = 0
	}

	if len(m.graphRows) == 0 {
		return wtRow + "\n" + DimStyle.Render("No matching commits.")
	}

	maxGraphCols := graph.MaxColumns(m.graphRows)
	graphWidth := maxGraphCols*2 + 1
	metaWidth := width - graphWidth - 2
	if metaWidth < 10 {
		metaWidth = 10
	}

	var lines []string

	endRow := m.scrollOff + height
	if endRow > len(m.graphRows) {
		endRow = len(m.graphRows)
	}

	for rowIdx := m.scrollOff; rowIdx < endRow; rowIdx++ {
		row := m.graphRows[rowIdx]
		graphStr := m.renderGraphCells(row, maxGraphCols)

		if row.Commit == nil {
			lines = append(lines, graphStr)
			continue
		}

		actualCommitIdx := 0
		for ci := 0; ci < rowIdx; ci++ {
			if m.graphRows[ci].Commit != nil {
				actualCommitIdx++
			}
		}
		isSelected := !m.wtSelected && actualCommitIdx == m.cursor

		meta := m.renderCommitMeta(row.Commit, metaWidth)
		line := graphStr + "  " + meta

		if isSelected {
			line = SelectedRowStyle.Width(width).Render(line)
		}

		lines = append(lines, line)
	}

	return wtRow + "\n" + strings.Join(lines, "\n")
}

// renderWorkingTreeRow renders the single pinned line representing the
// working tree, above the scrollable commit graph.
func (m Model) renderWorkingTreeRow(width int) string {
	staged, unstaged := len(m.wtStatus.Staged), len(m.wtStatus.Unstaged)
	var label string
	if staged == 0 && unstaged == 0 {
		label = "● Working Tree (clean)"
	} else {
		label = fmt.Sprintf("● Working Tree (%d staged, %d unstaged)", staged, unstaged)
	}
	if m.wtSelected {
		return SelectedRowStyle.Width(width).Render(label)
	}
	return DimStyle.Render(label)
}

func (m Model) renderGraphCells(row graph.GraphRow, maxCols int) string {
	var sb strings.Builder
	for i := 0; i < maxCols; i++ {
		if i > 0 {
			connectsRight := false
			connectsLeft := false
			if i-1 < len(row.Cells) {
				c := row.Cells[i-1].Char
				connectsRight = (c == '╰' || c == '╭' || c == '─' || c == '├' || c == '┼')
			}
			if i < len(row.Cells) {
				c := row.Cells[i].Char
				connectsLeft = (c == '╯' || c == '╮' || c == '─' || c == '┤' || c == '┼')
			}
			if connectsRight && connectsLeft {
				colorIdx := 0
				if i-1 < len(row.Cells) {
					colorIdx = row.Cells[i-1].Color
				}
				style := lipgloss.NewStyle().Foreground(BranchColors[colorIdx%len(BranchColors)])
				sb.WriteString(style.Render("─"))
			} else {
				sb.WriteRune(' ')
			}
		}

		if i < len(row.Cells) {
			cell := row.Cells[i]
			charStr := string(cell.Char)
			colorIdx := cell.Color % len(BranchColors)
			style := lipgloss.NewStyle().Foreground(BranchColors[colorIdx])

			switch cell.Char {
			case '*':
				style = style.Bold(true)
				sb.WriteString(style.Render("●"))
			case '│', '─', '╭', '╮', '╯', '╰', '├', '┤', '┼':
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

func (m Model) renderCommitMeta(c *git.Commit, width int) string {
	var parts []string

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

	parts = append(parts, HashStyle.Render(c.ShortHash))

	subject := c.Subject
	usedWidth := 0
	for _, p := range parts {
		usedWidth += lipgloss.Width(p) + 1
	}
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
	parts = append(parts, DateStyle.Render(dateStr))

	return strings.Join(parts, " ")
}

func (m Model) renderDetailPane(width, height int) string {
	if m.wtSelected {
		return m.renderWorkingTreeDetail(width, height)
	}

	c := m.selectedCommit()
	if c == nil {
		return DimStyle.Render("No commit selected.")
	}

	var lines []string

	lines = append(lines, SectionHeaderStyle.Render("Commit Details"))
	lines = append(lines, "")

	lines = append(lines, DetailLabelStyle.Render("Commit  ")+HashStyle.Render(c.Hash))
	lines = append(lines, DetailLabelStyle.Render("Author  ")+AuthorStyle.Render(c.Author)+" "+DimStyle.Render("<"+c.Email+">"))
	lines = append(lines, DetailLabelStyle.Render("Date    ")+DateStyle.Render(c.RelDate))

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

	lines = append(lines, SectionHeaderStyle.Render("Message"))
	lines = append(lines, "")
	lines = append(lines, SubjectStyle.Render(c.Subject))
	if c.Body != "" {
		lines = append(lines, "")
		bodyLines := strings.Split(c.Body, "\n")
		for _, bl := range bodyLines {
			if len(bl) > width {
				bl = bl[:width-1] + "…"
			}
			lines = append(lines, BodyStyle.Render(bl))
		}
	}

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

// renderWorkingTreeDetail lists the working tree's staged and unstaged
// files, highlighting whichever one wtFileCursor currently points at.
func (m Model) renderWorkingTreeDetail(width, height int) string {
	var lines []string
	lines = append(lines, SectionHeaderStyle.Render("Working Tree"))
	lines = append(lines, "")

	entries := m.wtFileEntries()
	if len(entries) == 0 {
		lines = append(lines, DimStyle.Render("Nothing to commit, working tree clean."))
		return strings.Join(lines, "\n")
	}

	if len(m.wtStatus.Staged) > 0 {
		lines = append(lines, SectionHeaderStyle.Render(fmt.Sprintf("Staged (%d)", len(m.wtStatus.Staged))))
	}
	for i, ref := range entries {
		if i == len(m.wtStatus.Staged) && len(m.wtStatus.Unstaged) > 0 {
			lines = append(lines, "")
			lines = append(lines, SectionHeaderStyle.Render(fmt.Sprintf("Unstaged (%d)", len(m.wtStatus.Unstaged))))
		}
		row := renderWtFileRow(ref, width)
		if i == m.wtFileCursor {
			row = SelectedRowStyle.Width(width).Render(row)
		}
		lines = append(lines, row)
	}
	return strings.Join(lines, "\n")
}

func renderWtFileRow(ref wtFileRef, width int) string {
	var statusStyle lipgloss.Style
	switch ref.entry.Status {
	case "M":
		statusStyle = FileModifiedStyle
	case "A", "??":
		statusStyle = FileAddedStyle
	case "D":
		statusStyle = FileDeletedStyle
	default:
		statusStyle = DimStyle
	}
	path := ref.entry.Path
	if len(path) > width-6 && width > 7 {
		path = "…" + path[len(path)-(width-7):]
	}
	return "  " + statusStyle.Render(ref.entry.Status) + " " + DimStyle.Render(path)
}

// renderDiffView renders the fullscreen, syntax-colored unified diff.
func (m Model) renderDiffView() string {
	if m.diffContent == "" {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			DimStyle.Render("Loading diff…"),
		)
	}

	diffLines := strings.Split(m.diffContent, "\n")
	visibleHeight := m.height - 3 // reserve for title, header spacer, and help bar
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	var rendered []string
	c := m.selectedCommit()
	title := fmt.Sprintf(" Diff for commit %s (Esc/Enter to return)", c.ShortHash)
	rendered = append(rendered, SectionHeaderStyle.Render(title))
	rendered = append(rendered, "")

	endLine := m.diffScroll + visibleHeight
	if endLine > len(diffLines) {
		endLine = len(diffLines)
	}

	for i := m.diffScroll; i < endLine; i++ {
		line := diffLines[i]
		var styledLine string
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			styledLine = FileAddedStyle.Render(line)
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			styledLine = FileDeletedStyle.Render(line)
		} else if strings.HasPrefix(line, "@@") {
			styledLine = lipgloss.NewStyle().Foreground(lipgloss.Color("#4FC3F7")).Render(line)
		} else if strings.HasPrefix(line, "commit ") || strings.HasPrefix(line, "Author:") || strings.HasPrefix(line, "Date:") {
			styledLine = HashStyle.Render(line)
		} else {
			styledLine = BodyStyle.Render(line)
		}

		if len(styledLine) > m.width {
			styledLine = styledLine[:m.width]
		}
		rendered = append(rendered, styledLine)
	}

	// Pad remaining empty vertical space so layout remains stable
	for len(rendered) < m.height-1 {
		rendered = append(rendered, "")
	}

	rendered = append(rendered, m.renderHelpBar())
	return strings.Join(rendered, "\n")
}

// renderStagingDiffView renders every hunk of the currently-open
// working-tree file diff, highlighting the hunk under stagingHunkCursor.
func (m Model) renderStagingDiffView() string {
	if len(m.stagingHunks) == 0 {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			DimStyle.Render("No changes left in this file."),
		)
	}

	var body []string
	for i, h := range m.stagingHunks {
		for _, l := range h.Lines {
			var styled string
			switch {
			case strings.HasPrefix(l, "+"):
				styled = FileAddedStyle.Render(l)
			case strings.HasPrefix(l, "-"):
				styled = FileDeletedStyle.Render(l)
			case strings.HasPrefix(l, "@@"):
				styled = lipgloss.NewStyle().Foreground(lipgloss.Color("#4FC3F7")).Render(l)
			default:
				styled = BodyStyle.Render(l)
			}
			if i == m.stagingHunkCursor {
				styled = SelectedRowStyle.Width(m.width).Render(styled)
			}
			body = append(body, styled)
		}
	}

	viewport := m.stagingViewportHeight()
	scroll := m.stagingScroll
	if scroll > len(body) {
		scroll = len(body)
	}
	endLine := scroll + viewport
	if endLine > len(body) {
		endLine = len(body)
	}

	var rendered []string
	title := fmt.Sprintf(" %s (esc to return, space to stage/unstage hunk)", m.stagingFilePath)
	rendered = append(rendered, SectionHeaderStyle.Render(title))
	rendered = append(rendered, "")
	rendered = append(rendered, body[scroll:endLine]...)

	for len(rendered) < m.height-1 {
		rendered = append(rendered, "")
	}
	rendered = append(rendered, m.renderHelpBar())
	return strings.Join(rendered, "\n")
}

func (m Model) renderHelpBar() string {
	var text string

	if m.showDiff && m.stagingDiffMode {
		text = "  ↑/k ↓/j Hunk  space Stage/Unstage  esc Close"
	} else if m.showDiff {
		text = "  ↑/k Up  ↓/j Down  pgup/pgdn Scroll  enter/esc/q Close"
	} else if m.searchMode {
		text = "  Filter: " + m.searchQuery + "█ (esc Clear, enter Apply)"
	} else if m.branchMode {
		if m.branchSubMode == "" {
			text = "  Branch Manager: [c] Create Branch  [d] Delete Branch  [esc] Cancel"
		} else if m.branchSubMode == "create" {
			text = "  Create Branch: " + m.branchInput + "█ (enter Confirm, esc Cancel)"
		} else if m.branchSubMode == "delete" {
			text = "  Delete Branch: " + m.branchInput + "█ (enter Confirm, esc Cancel)"
		}
	} else if m.commitInputMode {
		text = "  Commit message: " + m.commitMessage + "█ (enter Commit, esc Cancel)"
	} else if m.confirmCheckout {
		c := m.selectedCommit()
		hash := "unknown"
		if c != nil {
			hash = c.ShortHash
		}
		text = NotifyErrorStyle.Render(ConfirmText(hash))
	} else if m.confirmDiscard {
		text = NotifyErrorStyle.Render("  Discard changes to this file? (y/n)")
	} else if m.notification != "" {
		text = m.notifyStyle.Render(m.notification)
	} else if m.wtSelected {
		text = fmt.Sprintf("  ↑/k ↓/j Files  %s Stage  %s Unstage  %s Discard  enter Hunk-stage  c Commit  q Quit", KeyStageFile, KeyUnstageFile, KeyDiscard)
	} else {
		text = "  ↑/k Up  ↓/j Down  g/G Top/Bottom  enter Diff  / Filter  b Branch  c Checkout  q Quit"
	}

	return HelpBarStyle.Width(m.width).Render(text)
}

// ─── Actions / Helpers ──────────────────────────────────────────────────────

func (m Model) executeBranchAction() tea.Cmd {
	c := m.selectedCommit()
	if c == nil {
		return nil
	}
	subMode := m.branchSubMode
	input := m.branchInput

	return func() tea.Msg {
		var err error
		var msg string
		if subMode == "create" {
			err = git.CreateBranch(input, c.Hash)
			msg = fmt.Sprintf(" ✓ Created branch %s", input)
		} else if subMode == "delete" {
			err = git.DeleteBranch(input, false)
			msg = fmt.Sprintf(" ✓ Deleted branch %s", input)
		}

		result := git.CheckoutResult{
			Hash:    c.Hash,
			Success: err == nil,
			Message: msg,
		}
		if err != nil {
			result.Message = err.Error()
		}
		return checkoutDoneMsg{result: result}
	}
}

// ─── Commands ───────────────────────────────────────────────────────────────

func parseCommitsCmd() tea.Cmd {
	return func() tea.Msg {
		commits, err := git.ParseLog()
		return commitsParsedMsg{commits: commits, err: err}
	}
}

func loadFilesCmd(hash string) tea.Cmd {
	return func() tea.Msg {
		files, err := git.GetChangedFiles(hash)
		return filesLoadedMsg{files: files, hash: hash, err: err}
	}
}

func loadStatusCmd() tea.Cmd {
	return func() tea.Msg {
		status, err := git.GetStatus()
		return statusLoadedMsg{status: status, err: err}
	}
}

func loadDiffCmd(hash string) tea.Cmd {
	return func() tea.Msg {
		diff, err := git.GetCommitDiff(hash)
		return diffLoadedMsg{hash: hash, diff: diff, err: err}
	}
}

func loadStagingDiffCmd(path string, staged bool) tea.Cmd {
	return func() tea.Msg {
		diff, err := git.GetWorkingTreeDiff(path, staged)
		if err != nil {
			return stagingDiffLoadedMsg{path: path, staged: staged, err: err}
		}
		header, hunks := git.ParseHunks(diff)
		return stagingDiffLoadedMsg{path: path, staged: staged, header: header, hunks: hunks}
	}
}

func checkoutCmd(hash string) tea.Cmd {
	return func() tea.Msg {
		result := git.Checkout(hash)
		return checkoutDoneMsg{result: result}
	}
}

func stageFileCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return stagingDoneMsg{action: "staged", path: path, err: git.StageFile(path)}
	}
}

func unstageFileCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return stagingDoneMsg{action: "unstaged", path: path, err: git.UnstageFile(path)}
	}
}

func discardFileCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return stagingDoneMsg{action: "discarded", path: path, err: git.DiscardFile(path)}
	}
}

func commitCmd(message string) tea.Cmd {
	return func() tea.Msg {
		return commitDoneMsg{result: git.CreateCommit(message)}
	}
}

func clearNotifyAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearNotifyMsg{}
	})
}

// filterDebounceCmd schedules a filter recompute after filterDebounceDelay,
// tagged with the generation active at the time of the keystroke. If more
// keystrokes arrive before it fires, the model's filterGen will have moved
// on and the stale tick is ignored (see the filterDebounceMsg case in Update).
func filterDebounceCmd(gen int) tea.Cmd {
	return tea.Tick(filterDebounceDelay, func(t time.Time) tea.Msg {
		return filterDebounceMsg{gen: gen}
	})
}
