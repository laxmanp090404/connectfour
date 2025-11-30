package models

// MessageType defines the type of websocket message
type MessageType string

const (
	MsgJoin      MessageType = "JOIN"
	MsgMove      MessageType = "MOVE"
	MsgGameStart MessageType = "START"
	MsgUpdate    MessageType = "UPDATE"
	MsgGameOver  MessageType = "GAME_OVER"
	MsgError     MessageType = "ERROR"
	MsgPing      MessageType = "PING"
)

// WSMessage is the envelope for all websocket communications
type WSMessage struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

// JoinPayload is sent by client to join queue
type JoinPayload struct {
	Username string `json:"username"`
}

// MovePayload is sent by client to make a move
type MovePayload struct {
	Column int `json:"column"`
}

// GameStartPayload is sent to client when game begins
type GameStartPayload struct {
	GameID    string `json:"gameId"`
	Opponent  string `json:"opponent"`
	Symbol    int    `json:"symbol"` // 1 or 2
	IsTurn    bool   `json:"isTurn"`
}

// GameUpdatePayload sends the new board state
type GameUpdatePayload struct {
	Board      [6][7]int `json:"board"`
	Turn       int       `json:"turn"` // 1 or 2
	IsYourTurn bool      `json:"isYourTurn"`
}

// GameOverPayload sends the result
type GameOverPayload struct {
	Winner   string `json:"winner"` // Username or "Draw"
	Reason   string `json:"reason"` // "connect4", "forfeit", "draw"
	WinLines [][]int `json:"winLines,omitempty"` // Coordinates of winning discs
}