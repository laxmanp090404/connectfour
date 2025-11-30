package game

import (
	"math/rand"
	"time"
)

// Bot logic for single player mode
type Bot struct {
	Symbol int
	Name   string
}

func NewBot(symbol int) *Bot {
	return &Bot{
		Symbol: symbol,
		Name:   "AI_Bot_v1",
	}
}

// GetMove decides the best column to drop a disc
func (bot *Bot) GetMove(b *Board) int {
	opponent := 1
	if bot.Symbol == 1 {
		opponent = 2
	}

	// 1. Can I win immediately?
	if col := findWinningMove(b, bot.Symbol); col != -1 {
		return col
	}

	// 2. Do I need to block opponent from winning?
	if col := findWinningMove(b, opponent); col != -1 {
		return col
	}

	// 3. Prefer Center columns (Strategy)
	// Column priorities: 3 (center), then 2,4, then 1,5, then 0,6
	priorities := []int{3, 2, 4, 1, 5, 0, 6}
	for _, col := range priorities {
		if canPlay(b, col) {
			// Add a little randomness so it's not identical every time
			if rand.Intn(10) > 1 { 
				return col
			}
		}
	}

	// 4. Random fallback
	rand.Seed(time.Now().UnixNano())
	validCols := []int{}
	for c := 0; c < Cols; c++ {
		if canPlay(b, c) {
			validCols = append(validCols, c)
		}
	}
	if len(validCols) > 0 {
		return validCols[rand.Intn(len(validCols))]
	}
	return -1
}

func canPlay(b *Board, col int) bool {
	return b[0][col] == 0
}

func findWinningMove(b *Board, player int) int {
	for c := 0; c < Cols; c++ {
		if !canPlay(b, c) {
			continue
		}
		// Simulate move
		tempBoard := *b // Copy board
		row := tempBoard.DropDisc(c, player)
		if row != -1 && tempBoard.CheckWin(row, c, player) {
			return c
		}
	}
	return -1
}