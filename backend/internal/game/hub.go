package game

import (
	"fmt"
	"sync"
	"time"

	"connectfour/internal/db"    // Import DB
	"connectfour/internal/event" // Import Kafka

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type WaitingPlayer struct {
	Player   *Player
	JoinedAt time.Time
}

type Hub struct {
	waiting       []*WaitingPlayer
	games         map[string]*Game
	playerGameMap map[*websocket.Conn]*Game
	mutex         sync.Mutex

	// Dependencies
	repo     *db.Repository
	producer *event.Producer
}

// Update Constructor to accept dependencies
func NewHub(repo *db.Repository, producer *event.Producer) *Hub {
	h := &Hub{
		waiting:       make([]*WaitingPlayer, 0),
		games:         make(map[string]*Game),
		playerGameMap: make(map[*websocket.Conn]*Game),
		repo:          repo,
		producer:      producer,
	}
	go h.matchmakerLoop()
	return h
}

func (h *Hub) matchmakerLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mutex.Lock()
		for len(h.waiting) >= 2 {
			p1 := h.waiting[0]
			p2 := h.waiting[1]
			h.waiting = h.waiting[2:]
			h.startGame(p1.Player, p2.Player)
		}

		remaining := []*WaitingPlayer{}
		for _, wp := range h.waiting {
			if time.Since(wp.JoinedAt) > 10*time.Second {
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

func (h *Hub) AddPlayer(conn *websocket.Conn, username string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	delete(h.playerGameMap, conn)
	wp := &WaitingPlayer{
		Player:   &Player{Conn: conn, Username: username},
		JoinedAt: time.Now(),
	}
	h.waiting = append(h.waiting, wp)
	fmt.Printf("Player %s joined queue.\n", username)
}

func (h *Hub) startGame(p1, p2 *Player) {
	id := uuid.New().String()
	game := NewGame(id, p1, p2, h.handleGameOver)
	h.games[id] = game
	
	if !p1.IsBot { h.playerGameMap[p1.Conn] = game }
	if !p2.IsBot { h.playerGameMap[p2.Conn] = game }

	fmt.Printf("Starting Game %s\n", id)
	go game.Start()
}

func (h *Hub) HandleMove(conn *websocket.Conn, col int) {
	h.mutex.Lock()
	game, exists := h.playerGameMap[conn]
	h.mutex.Unlock()

	if !exists || game == nil {
		return
	}
	symbol := 1
	if game.Player2.Conn == conn {
		symbol = 2
	}
	game.MakeMove(symbol, col)
}

func (h *Hub) HandleDisconnect(conn *websocket.Conn) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.playerGameMap, conn)
	for i, wp := range h.waiting {
		if wp.Player.Conn == conn {
			h.waiting = append(h.waiting[:i], h.waiting[i+1:]...)
			break
		}
	}
}

func (h *Hub) handleGameOver(g *Game, winner, reason string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	delete(h.games, g.ID)
	if !g.Player1.IsBot { delete(h.playerGameMap, g.Player1.Conn) }
	if !g.Player2.IsBot { delete(h.playerGameMap, g.Player2.Conn) }
	
	fmt.Printf("Game Over: %s won. Saving to DB/Kafka.\n", winner)

	// 1. Save to Postgres
	if h.repo != nil {
		h.repo.SaveGame(g.ID, g.Player1.Username, g.Player2.Username, winner, reason)
	}

	// 2. Emit to Kafka
	if h.producer != nil {
		h.producer.EmitGameOver(g.ID, winner)
	}
}