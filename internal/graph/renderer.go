package graph

import (
	"strings"

	"gitsketch/internal/git"
)

// Cell represents a single character cell in the ASCII graph.
type Cell struct {
	Char  rune // '*', '│', '/', '\\', '─', '·', ' '
	Color int  // color index 0-7 for cycling branch colors
}

// GraphRow represents one rendered row of the ASCII graph,
// containing the visual cells and a reference to the commit.
type GraphRow struct {
	Cells  []Cell      // individual cells making up the graph prefix
	Commit *git.Commit // nil for connector/merge rows
	Column int         // which column the commit node sits in (-1 for connector rows)
}

// BuildGraph converts a topologically-sorted commit list into an ASCII
// branch visualization. It uses a column-lane assignment algorithm that
// dynamically manages active lanes, handling branches, merges, and
// octopus merges.
func BuildGraph(commits []git.Commit) []GraphRow {
	if len(commits) == 0 {
		return nil
	}

	// columns tracks which commit hash is "expected" in each lane.
	// An empty string means the lane is free.
	var columns []string

	// colorMap assigns a stable color to each lane.
	colorMap := make(map[int]int)
	nextColor := 0

	var rows []GraphRow

	for i := range commits {
		c := &commits[i]

		// ── Step 1: Find this commit's column ──
		commitCol := -1
		var mergeCols []int // columns where this commit also appears (merge convergence)

		for col, hash := range columns {
			if hash == c.Hash {
				if commitCol == -1 {
					commitCol = col
				} else {
					mergeCols = append(mergeCols, col)
				}
			}
		}

		// If not found in any column, append a new one.
		if commitCol == -1 {
			commitCol = findFreeColumn(columns)
			if commitCol == len(columns) {
				columns = append(columns, c.Hash)
			} else {
				columns[commitCol] = c.Hash
			}
		}

		// Assign color to this column if not already assigned.
		if _, ok := colorMap[commitCol]; !ok {
			colorMap[commitCol] = nextColor % 8
			nextColor++
		}

		// ── Step 2: Build the commit row ──
		numCols := len(columns)
		cells := make([]Cell, numCols)

		for col := 0; col < numCols; col++ {
			if col == commitCol {
				cells[col] = Cell{Char: '*', Color: colorMap[commitCol]}
			} else if containsInt(mergeCols, col) {
				// This column is converging into the commit (merge)
				if col < commitCol {
					cells[col] = Cell{Char: '/', Color: getColor(colorMap, col)}
				} else {
					cells[col] = Cell{Char: '\\', Color: getColor(colorMap, col)}
				}
			} else if columns[col] != "" {
				cells[col] = Cell{Char: '│', Color: getColor(colorMap, col)}
			} else {
				cells[col] = Cell{Char: ' ', Color: 0}
			}
		}

		rows = append(rows, GraphRow{
			Cells:  trimTrailingEmpty(cells),
			Commit: c,
			Column: commitCol,
		})

		// ── Step 3: Free merge convergence columns ──
		for _, mc := range mergeCols {
			columns[mc] = ""
			delete(colorMap, mc)
		}

		// ── Step 4: Assign parents to columns ──
		if len(c.Parents) == 0 {
			// Root commit: free this column.
			columns[commitCol] = ""
			delete(colorMap, commitCol)
		} else {
			// First parent inherits this commit's column.
			columns[commitCol] = c.Parents[0]

			// Additional parents (merge parents) get new/free columns.
			for _, parentHash := range c.Parents[1:] {
				// Check if this parent is already expected somewhere.
				alreadyAssigned := false
				for _, h := range columns {
					if h == parentHash {
						alreadyAssigned = true
						break
					}
				}
				if !alreadyAssigned {
					freeCol := findFreeColumn(columns)
					if freeCol == len(columns) {
						columns = append(columns, parentHash)
					} else {
						columns[freeCol] = parentHash
					}
					// Assign color to the new branch lane.
					if _, ok := colorMap[freeCol]; !ok {
						colorMap[freeCol] = nextColor % 8
						nextColor++
					}
				}
			}
		}

		// ── Step 5: Generate connector rows for visual clarity ──
		// If merge columns were freed, we may need a connector row
		// showing the diagonal lines converging.
		if len(mergeCols) > 0 {
			connCells := buildConnectorRow(columns, commitCol, mergeCols, colorMap)
			if connCells != nil {
				rows = append(rows, GraphRow{
					Cells:  trimTrailingEmpty(connCells),
					Commit: nil,
					Column: -1,
				})
			}
		}

		// Compact trailing empty columns.
		columns = trimTrailingEmptyStr(columns)
	}

	return rows
}

// buildConnectorRow generates an optional connector row showing
// the visual lines after a merge commit.
func buildConnectorRow(columns []string, commitCol int, mergeCols []int, colorMap map[int]int) []Cell {
	numCols := len(columns)
	if numCols == 0 {
		return nil
	}

	cells := make([]Cell, numCols)
	hasContent := false

	for col := 0; col < numCols; col++ {
		if columns[col] != "" {
			cells[col] = Cell{Char: '│', Color: getColor(colorMap, col)}
			hasContent = true
		} else {
			cells[col] = Cell{Char: ' ', Color: 0}
		}
	}

	if !hasContent {
		return nil
	}
	return cells
}

// MaxColumns returns the maximum number of cells across all rows.
func MaxColumns(rows []GraphRow) int {
	max := 0
	for _, r := range rows {
		if len(r.Cells) > max {
			max = len(r.Cells)
		}
	}
	return max
}

// RenderRow converts a GraphRow's cells into a plain string for debugging.
func RenderRow(row GraphRow) string {
	var sb strings.Builder
	for i, cell := range row.Cells {
		if i > 0 {
			sb.WriteRune(' ')
		}
		sb.WriteRune(cell.Char)
	}
	return sb.String()
}

// ─── Helpers ────────────────────────────────────────────────────────────────

// findFreeColumn returns the index of the first empty column,
// or len(columns) if none are free (meaning a new column should be appended).
func findFreeColumn(columns []string) int {
	for i, h := range columns {
		if h == "" {
			return i
		}
	}
	return len(columns)
}

// containsInt checks if a slice contains a given integer.
func containsInt(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// getColor retrieves the color for a column, defaulting to 0.
func getColor(colorMap map[int]int, col int) int {
	if c, ok := colorMap[col]; ok {
		return c
	}
	return 0
}

// trimTrailingEmpty removes trailing space cells from a cell slice.
func trimTrailingEmpty(cells []Cell) []Cell {
	end := len(cells)
	for end > 0 && cells[end-1].Char == ' ' {
		end--
	}
	if end == 0 {
		return cells[:1] // keep at least one cell
	}
	return cells[:end]
}

// trimTrailingEmptyStr removes trailing empty strings from a string slice.
func trimTrailingEmptyStr(s []string) []string {
	end := len(s)
	for end > 0 && s[end-1] == "" {
		end--
	}
	return s[:end]
}
