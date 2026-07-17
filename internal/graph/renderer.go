package graph

import (
	"strings"

	"github.com/Wian47/GitSketch/internal/git"
)

// Cell represents a single character cell in the ASCII/Unicode graph.
type Cell struct {
	Char  rune // '*', '│', '─', '╭', '╮', '╯', '╰', '├', '┤', '┼', ' '
	Color int  // color index 0-7 for cycling branch colors
}

// GraphRow represents one rendered row of the graph.
type GraphRow struct {
	Cells  []Cell      // individual cells making up the graph prefix
	Commit *git.Commit // nil for connector/merge rows
	Column int         // which column the commit node sits in (-1 for connector rows)
}

// BuildGraph converts a topologically-sorted commit list into a beautiful
// curved Unicode branch graph.
func BuildGraph(commits []git.Commit) []GraphRow {
	if len(commits) == 0 {
		return nil
	}

	// columns tracks which commit hash is expected in each lane.
	var columns []string
	colorMap := make(map[int]int)
	nextColor := 0

	var rows []GraphRow

	for i := range commits {
		c := &commits[i]

		// ── Step 1: Find commit column and merge sources ──
		commitCol := -1
		var mergeCols []int

		for col, hash := range columns {
			if hash == c.Hash {
				if commitCol == -1 {
					commitCol = col
				} else {
					mergeCols = append(mergeCols, col)
				}
			}
		}

		if commitCol == -1 {
			commitCol = findFreeColumn(columns)
			if commitCol == len(columns) {
				columns = append(columns, c.Hash)
			} else {
				columns[commitCol] = c.Hash
			}
		}

		if _, ok := colorMap[commitCol]; !ok {
			colorMap[commitCol] = nextColor % 8
			nextColor++
		}

		// ── Step 2: Render commit row with merge curves ──
		numCols := len(columns)
		cells := make([]Cell, numCols)

		// Initialize default vertical lines or empty spaces
		for col := 0; col < numCols; col++ {
			if columns[col] != "" {
				cells[col] = Cell{Char: '│', Color: getColor(colorMap, col)}
			} else {
				cells[col] = Cell{Char: ' ', Color: 0}
			}
		}

		// Place the commit node
		cells[commitCol] = Cell{Char: '*', Color: colorMap[commitCol]}

		// Overlay merge curves leading to the commit node
		for _, mc := range mergeCols {
			color := getColor(colorMap, mc)
			if mc < commitCol {
				// Merging from left to right: starts at mc, goes right to commitCol
				cells[mc] = Cell{Char: '╰', Color: color}
				for col := mc + 1; col < commitCol; col++ {
					cells[col] = Cell{Char: '─', Color: color}
				}
			} else {
				// Merging from right to left: starts at mc, goes left to commitCol
				cells[mc] = Cell{Char: '╯', Color: color}
				for col := commitCol + 1; col < mc; col++ {
					cells[col] = Cell{Char: '─', Color: color}
				}
			}
		}

		rows = append(rows, GraphRow{
			Cells:  trimTrailingEmpty(cells),
			Commit: c,
			Column: commitCol,
		})

		// ── Step 3: Free merge columns ──
		for _, mc := range mergeCols {
			columns[mc] = ""
			delete(colorMap, mc)
		}

		// ── Step 4: Handle branch forks (child to parent propagation) ──
		var branchForks []int // columns of new branches being created in this step
		oldCommitCol := commitCol

		if len(c.Parents) == 0 {
			columns[commitCol] = ""
			delete(colorMap, commitCol)
		} else {
			// First parent inherits current lane
			columns[commitCol] = c.Parents[0]

			// Additional parents get new/free lanes (branch out)
			for _, parentHash := range c.Parents[1:] {
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
					colorMap[freeCol] = nextColor % 8
					nextColor++
					branchForks = append(branchForks, freeCol)
				}
			}
		}

		// ── Step 5: Render connector row for branch forks ──
		if len(branchForks) > 0 {
			connCells := make([]Cell, len(columns))
			for col := 0; col < len(columns); col++ {
				if col < len(cells) && cells[col].Char == '│' && columns[col] != "" {
					connCells[col] = Cell{Char: '│', Color: getColor(colorMap, col)}
				} else {
					connCells[col] = Cell{Char: ' ', Color: 0}
				}
			}

			// Main parent line continues down
			connCells[oldCommitCol] = Cell{Char: '│', Color: getColor(colorMap, oldCommitCol)}

			// Render branching lines from oldCommitCol to each fork destination
			for _, fc := range branchForks {
				color := getColor(colorMap, fc)
				if fc > oldCommitCol {
					// Branching to the right: connects main line to the right
					connCells[oldCommitCol] = Cell{Char: '├', Color: getColor(colorMap, oldCommitCol)}
					for col := oldCommitCol + 1; col < fc; col++ {
						connCells[col] = Cell{Char: '─', Color: color}
					}
					connCells[fc] = Cell{Char: '╮', Color: color}
				} else {
					// Branching to the left: connects main line to the left
					connCells[oldCommitCol] = Cell{Char: '┤', Color: getColor(colorMap, oldCommitCol)}
					for col := fc + 1; col < oldCommitCol; col++ {
						connCells[col] = Cell{Char: '─', Color: color}
					}
					connCells[fc] = Cell{Char: '╭', Color: color}
				}
			}

			rows = append(rows, GraphRow{
				Cells:  trimTrailingEmpty(connCells),
				Commit: nil,
				Column: -1,
			})
		}

		columns = trimTrailingEmptyStr(columns)
	}

	return rows
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

// RenderRow converts a GraphRow's cells into a plain string.
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

// findFreeColumn returns the first empty slot.
func findFreeColumn(columns []string) int {
	for i, h := range columns {
		if h == "" {
			return i
		}
	}
	return len(columns)
}

func getColor(colorMap map[int]int, col int) int {
	if c, ok := colorMap[col]; ok {
		return c
	}
	return 0
}

func trimTrailingEmpty(cells []Cell) []Cell {
	end := len(cells)
	for end > 0 && cells[end-1].Char == ' ' {
		end--
	}
	if end == 0 {
		return cells[:1]
	}
	return cells[:end]
}

func trimTrailingEmptyStr(s []string) []string {
	end := len(s)
	for end > 0 && s[end-1] == "" {
		end--
	}
	return s[:end]
}
