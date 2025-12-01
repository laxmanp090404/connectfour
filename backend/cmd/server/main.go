package main

import (
	"connectfour/internal/api"
	"connectfour/internal/game"
	"log"
	"net/http"
)

func main() {
	log.Println("Starting Connect 4 Server...")

	// 1. Initialize Game Hub
	hub := game.NewHub()

	// 2. Setup Routes
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		api.ServeWs(hub, w, r)
	})
	
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// 3. Start Server
	port := "8080"
	log.Printf("Server running on http://localhost:%s", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}