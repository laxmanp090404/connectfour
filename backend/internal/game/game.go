package game

import (
	"connectfour/pkg/models"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	Conn     *websocket.Conn
	Username string
	Symbol   int // 1 or 2
	IsBot    bool
}

type Game struct {
	ID        string
	Board     *Board
	Player1   *Player
	Player2   *Player
	Turn      int // 1 or 2
	Status    string // "playing", "finished"
	CreatedAt time.Time
	
	broadcast chan models.WSMessage
	mutex     sync.Mutex
	
	// callback to save game result
	OnGameOver func(game *Game, winner string, reason string)
}

func NewGame(id string, p1, p2 *Player, onGameOver func(*Game, string, string)) *Game {
	g := &Game{
		ID:         id,
		Board:      NewBoard(),
		Player1:    p1,
		Player2:    p2,
		Turn:       1, // Player 1 starts
		Status:     "playing",
		CreatedAt:  time.Now(),
		broadcast:  make(chan models.WSMessage),
		OnGameOver: onGameOver,
	}
	p1.Symbol = 1
	p2.Symbol = 2
	return g
}

// Start begins the game loop
func (g *Game) Start() {
	// Notify players game started
	g.sendTo(g.Player1, models.MsgGameStart, models.GameStartPayload{
		GameID: g.ID, Opponent: g.Player2.Username, Symbol: 1, IsTurn: true,
	})
	
	if !g.Player2.IsBot {
		g.sendTo(g.Player2, models.MsgGameStart, models.GameStartPayload{
			GameID: g.ID, Opponent: g.Player1.Username, Symbol: 2, IsTurn: false,
		})
	} else {
		// If P2 is bot, we don't send WS message, but we might need to trigger move if it was P2 turn (not the case here)
	}
}

// MakeMove handles a player dropping a disc
func (g *Game) MakeMove(playerSymbol, col int) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.Status != "playing" || g.Turn != playerSymbol {
		return
	}

	row := g.Board.DropDisc(col, playerSymbol)
	if row == -1 {
		return // Column full
	}

	// Check Win
	if g.Board.CheckWin(row, col, playerSymbol) {
		g.Status = "finished"
		winnerName := g.Player1.Username
		if playerSymbol == 2 {
			winnerName = g.Player2.Username
		}
		g.broadcastUpdate()
		g.endGame(winnerName, "connect4")
		return
	}

	// Check Draw
	if g.Board.IsFull() {
		g.Status = "finished"
		g.broadcastUpdate()
		g.endGame("Draw", "draw")
		return
	}

	// Switch Turn
	g.Turn = 3 - g.Turn // 1 becomes 2, 2 becomes 1
	g.broadcastUpdate()

	// If next player is Bot, trigger bot move
	if g.Turn == 2 && g.Player2.IsBot {
		go func() {
			time.Sleep(500 * time.Millisecond) // Simulate thinking
			bot := NewBot(2)
			botMove := bot.GetMove(g.Board)
			g.MakeMove(2, botMove)
		}()
	}
}

func (g *Game) broadcastUpdate() {
	payload := models.GameUpdatePayload{
		Board: *g.Board,
		Turn:  g.Turn,
	}
	
	payload.IsYourTurn = (g.Turn == 1)
	g.sendTo(g.Player1, models.MsgUpdate, payload)

	if !g.Player2.IsBot {
		payload.IsYourTurn = (g.Turn == 2)
		g.sendTo(g.Player2, models.MsgUpdate, payload)
	}
}

func (g *Game) endGame(winner, reason string) {
	msg := models.WSMessage{
		Type: models.MsgGameOver,
		Payload: models.GameOverPayload{
			Winner: winner,
			Reason: reason,
		},
	}
	
	g.safeWrite(g.Player1.Conn, msg)
	if !g.Player2.IsBot {
		g.safeWrite(g.Player2.Conn, msg)
	}
	
	if g.OnGameOver != nil {
		g.OnGameOver(g, winner, reason)
	}
}

func (g *Game) sendTo(p *Player, msgType models.MessageType, data interface{}) {
	if p.IsBot {
		return
	}
	msg := models.WSMessage{Type: msgType, Payload: data}
	g.safeWrite(p.Conn, msg)
}

func (g *Game) safeWrite(conn *websocket.Conn, msg interface{}) {
	if conn == nil {
		return
	}
	// In a real production app, we would use a writePump channel to avoid concurrent writes
	// For this interview assignment, strict locking on the connection is acceptable or simple sync write
	conn.WriteJSON(msg)
}