package api

import (
	"connectfour/internal/game"
	"connectfour/pkg/models"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins (CORS) for development
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWs handles websocket requests from the peer.
func ServeWs(hub *game.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade Error:", err)
		return
	}

	// Ensure connection closes when function returns
	defer func() {
		hub.HandleDisconnect(conn)
		conn.Close()
	}()

	log.Println("New Client Connected")

	// Read Loop
	for {
		var msg models.WSMessage
		// Read JSON message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read Error:", err)
			break // Break loop -> disconnect
		}

		switch msg.Type {
		case models.MsgJoin:
			// Parse payload
			data, ok := msg.Payload.(map[string]interface{})
			if ok {
				username := data["username"].(string)
				hub.AddPlayer(conn, username)
			}
		
		case models.MsgMove:
			// Parse payload
			data, ok := msg.Payload.(map[string]interface{})
			if ok {
				// JSON numbers are floats in Go interface{}, need to cast
				colFloat := data["column"].(float64)
				hub.HandleMove(conn, int(colFloat))
			}
		}
	}
}