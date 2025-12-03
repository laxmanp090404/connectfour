package game

import (
	"fmt"
	"sync"
	"time"

	"connectfour/internal/db"
	"connectfour/internal/event"
	"connectfour/pkg/models"

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

	repo     *db.Repository
	producer *event.Producer
}

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
		
		// Matchmaking
		for len(h.waiting) >= 2 {
			p1 := h.waiting[0]
			p2 := h.waiting[1]
			h.waiting = h.waiting[2:]
			h.startGame(p1.Player, p2.Player)
		}

		// Bot Fallback
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

// AddPlayer handles New Joins AND Reconnections
func (h *Hub) AddPlayer(conn *websocket.Conn, username string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// 1. RECONNECTION LOGIC
	for _, game := range h.games {
		if game.Status == "playing" {
			// Player 1 Rejoin
			if game.Player1.Username == username {
				fmt.Printf("♻️ REJOIN: %s reconnected to game %s\n", username, game.ID)
				if game.P1Timer != nil { 
					game.P1Timer.Stop() // STOP THE FORFEIT TIMER
					game.P1Timer = nil
				}
				h.reconnectPlayer(game, game.Player1, conn, 1)
				return
			}
			// Player 2 Rejoin
			if game.Player2.Username == username {
				fmt.Printf("♻️ REJOIN: %s reconnected to game %s\n", username, game.ID)
				if game.P2Timer != nil { 
					game.P2Timer.Stop() // STOP THE FORFEIT TIMER
					game.P2Timer = nil
				}
				h.reconnectPlayer(game, game.Player2, conn, 2)
				return
			}
		}
	}

	// 2. NEW PLAYER LOGIC
	delete(h.playerGameMap, conn)
	wp := &WaitingPlayer{
		Player:   &Player{Conn: conn, Username: username},
		JoinedAt: time.Now(),
	}
	h.waiting = append(h.waiting, wp)
	fmt.Printf("Player %s joined queue.\n", username)
}

func (h *Hub) reconnectPlayer(g *Game, p *Player, conn *websocket.Conn, symbol int) {
	p.Conn = conn
	h.playerGameMap[conn] = g

	// Send State
	startPayload := models.GameStartPayload{
		GameID: g.ID, Opponent: g.Player2.Username, Symbol: symbol, IsTurn: (g.Turn == symbol),
	}
	if symbol == 2 { startPayload.Opponent = g.Player1.Username }
	conn.WriteJSON(models.WSMessage{Type: models.MsgGameStart, Payload: startPayload})

	updatePayload := models.GameUpdatePayload{
		Board:      *g.Board,
		Turn:       g.Turn,
		IsYourTurn: (g.Turn == symbol),
	}
	conn.WriteJSON(models.WSMessage{Type: models.MsgUpdate, Payload: updatePayload})
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

	if !exists || game == nil { return }

	symbol := 1
	if game.Player2.Conn == conn { symbol = 2 }
	game.MakeMove(symbol, col)
}

func (h *Hub) HandleDisconnect(conn *websocket.Conn) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	game, exists := h.playerGameMap[conn]
	
	// Remove from connection lookup immediately
	delete(h.playerGameMap, conn)

	// Remove from waiting queue if they were just waiting
	for i, wp := range h.waiting {
		if wp.Player.Conn == conn {
			h.waiting = append(h.waiting[:i], h.waiting[i+1:]...)
			return // Just a waiter, no game logic needed
		}
	}

	// GAME DISCONNECT LOGIC
	if exists && game != nil && game.Status == "playing" {
		fmt.Printf("⚠️ Player disconnected from Game %s. Starting 30s timer.\n", game.ID)
		
		// Define the forfeit callback
		forfeitFunc := func() {
			// We must lock inside the callback because it runs in a separate goroutine
			h.mutex.Lock()
			defer h.mutex.Unlock()
			
			// Verify game still exists and is still playing
			if currentG, ok := h.games[game.ID]; ok && currentG.Status == "playing" {
				fmt.Printf("⏰ Timeout! Forfeiting game %s\n", game.ID)
				
				winner := game.Player2.Username
				if game.Player2.Conn == conn { // If P2 disconnected, P1 wins
					winner = game.Player1.Username
				}
				
				// Manually trigger game over
				currentG.Status = "finished"
				
				// Notify the winner (if they are still there)
				// We call endGame on the game struct which sends WS messages
				currentG.Status = "finished" // Ensure status is updated
				
				// Note: accessing endGame requires us to be careful about locks, 
				// but handleGameOver deals with the cleanup.
				// Simplest way: Call handleGameOver directly to clean DB/Map
				h.handleGameOver(currentG, winner, "forfeit")

				// Send WS message to the remaining player
				msg := models.WSMessage{
					Type: models.MsgGameOver,
					Payload: models.GameOverPayload{
						Winner: winner,
						Reason: "forfeit",
					},
				}
				if winner == game.Player1.Username {
					game.Player1.Conn.WriteJSON(msg)
				} else if !game.Player2.IsBot {
					game.Player2.Conn.WriteJSON(msg)
				}
			}
		}

		// Start Timer based on who disconnected
		if game.Player1.Conn == conn {
			game.P1Timer = time.AfterFunc(30*time.Second, forfeitFunc)
		} else if game.Player2.Conn == conn {
			game.P2Timer = time.AfterFunc(30*time.Second, forfeitFunc)
		}
	}
}

func (h *Hub) handleGameOver(g *Game, winner, reason string) {
	// Note: mutex is usually held by caller, but if called from Game struct, it might not be.
	// However, in our flow, Hub controls the lifecycle.
	
	delete(h.games, g.ID)
	// Clean up connection maps if they exist
	if !g.Player1.IsBot && g.Player1.Conn != nil { delete(h.playerGameMap, g.Player1.Conn) }
	if !g.Player2.IsBot && g.Player2.Conn != nil { delete(h.playerGameMap, g.Player2.Conn) }
	
	fmt.Printf("Game Over: %s won (%s). Saving to DB/Kafka.\n", winner, reason)

	if h.repo != nil {
		h.repo.SaveGame(g.ID, g.Player1.Username, g.Player2.Username, winner, reason)
	}
	if h.producer != nil {
		h.producer.EmitGameOver(g.ID, winner)
	}
}