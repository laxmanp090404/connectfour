package game

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WaitingPlayer represents a user in the queue
type WaitingPlayer struct {
	Player   *Player
	JoinedAt time.Time
}

// Hub manages matchmaking and active games
type Hub struct {
	// Queue of waiting players
	waiting []*WaitingPlayer
	
	// Active games: map[GameID]*Game
	games map[string]*Game

	mutex sync.Mutex
}

// NewHub creates the manager and starts the background matchmaker
func NewHub() *Hub {
	h := &Hub{
		waiting: make([]*WaitingPlayer, 0),
		games:   make(map[string]*Game),
	}
	// Start the matchmaker "Ticker" in a background goroutine
	go h.matchmakerLoop()
	return h
}

// matchmakerLoop checks every second for pairs or timeouts
func (h *Hub) matchmakerLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mutex.Lock()
		
		// 1. MATCHMAKING: Pair up players if 2+ are waiting
		for len(h.waiting) >= 2 {
			p1 := h.waiting[0]
			p2 := h.waiting[1]
			h.waiting = h.waiting[2:] // Remove matched players from queue
			
			h.startGame(p1.Player, p2.Player)
		}

		// 2. BOT FALLBACK: Check for timeouts (> 10 seconds)
		// We rebuild the list, keeping only those who haven't timed out yet
		remaining := []*WaitingPlayer{}
		for _, wp := range h.waiting {
			if time.Since(wp.JoinedAt) > 10*time.Second {
				// Player waited too long -> Start game vs Bot
				bot := &Player{Username: "Bot", IsBot: true, Symbol: 2}
				h.startGame(wp.Player, bot)
			} else {
				remaining = append(remaining, wp)
			}
		}
		h.waiting = remaining
		
		h.mutex.Unlock()
	}
}

// AddPlayer adds a new connection to the matchmaking queue
func (h *Hub) AddPlayer(conn *websocket.Conn, username string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	wp := &WaitingPlayer{
		Player:   &Player{Conn: conn, Username: username},
		JoinedAt: time.Now(),
	}
	h.waiting = append(h.waiting, wp)
	fmt.Printf("Player %s joined queue. Total waiting: %d\n", username, len(h.waiting))
}

// startGame creates a game instance and launches it
func (h *Hub) startGame(p1, p2 *Player) {
	id := uuid.New().String()
	game := NewGame(id, p1, p2, h.handleGameOver)
	
	h.games[id] = game
	
	fmt.Printf("Starting Game %s: %s vs %s\n", id, p1.Username, p2.Username)
	
	// Run the game in its own Goroutine so it doesn't block the Hub
	go game.Start()
}

// handleGameOver is a callback passed to Game
func (h *Hub) handleGameOver(g *Game, winner, reason string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	// Remove game from memory
	delete(h.games, g.ID)
	
	fmt.Printf("Game %s ended. Winner: %s (%s). Removing from memory.\n", g.ID, winner, reason)
	
	// TODO: We will hook up Postgres/Kafka here in the next steps
}

// GetGame retrieves an active game by ID (useful for reconnection later)
func (h *Hub) GetGame(gameID string) *Game {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.games[gameID]
}