package game

const (
	Rows = 6
	Cols = 7
)

type Board [Rows][Cols]int

// NewBoard creates an empty grid
func NewBoard() *Board {
	return &Board{}
}

// DropDisc attempts to place a disc in a column. Returns row index or -1 if full.
func (b *Board) DropDisc(col int, player int) int {
	if col < 0 || col >= Cols {
		return -1
	}
	// Start from bottom (Row 5) and go up
	for r := Rows - 1; r >= 0; r-- {
		if b[r][col] == 0 {
			b[r][col] = player
			return r
		}
	}
	return -1
}

// IsFull checks if the board is a draw
func (b *Board) IsFull() bool {
	for c := 0; c < Cols; c++ {
		if b[0][c] == 0 {
			return false
		}
	}
	return true
}

// CheckWin returns true if the last move at (row, col) created a win
func (b *Board) CheckWin(row, col, player int) bool {
	// Directions: Horizontal, Vertical, Diagonal /, Diagonal \
	directions := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}

	for _, d := range directions {
		count := 1 // Count the piece we just placed
		dr, dc := d[0], d[1]

		// Check forward
		for i := 1; i < 4; i++ {
			r, c := row+dr*i, col+dc*i
			if r < 0 || r >= Rows || c < 0 || c >= Cols || b[r][c] != player {
				break
			}
			count++
		}

		// Check backward
		for i := 1; i < 4; i++ {
			r, c := row-dr*i, col-dc*i
			if r < 0 || r >= Rows || c < 0 || c >= Cols || b[r][c] != player {
				break
			}
			count++
		}

		if count >= 4 {
			return true
		}
	}
	return false
}