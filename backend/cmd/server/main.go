package main

import (
	"connectfour/internal/api"
	"connectfour/internal/db"
	"connectfour/internal/event"
	"connectfour/internal/game"
	"fmt"
	"log"
	"net/http"
	"os"
)

// Middleware to force CORS on any handler
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Set Headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// 2. Handle Preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 3. Call the actual handler
		next(w, r)
	}
}

func main() {
	log.Println("Starting Connect 4 Server...")

	// 1. Database
	dsn := fmt.Sprintf("postgres://%s:%s@localhost:5432/%s?sslmode=disable",
		getEnv("DB_USER", "user"),
		getEnv("DB_PASSWORD", "password"),
		getEnv("DB_NAME", "connect4"),
	)
	
	repository, err := db.NewRepository(dsn)
	if err != nil {
		log.Printf("WARNING: Could not connect to Database: %v", err)
	} else {
		log.Println("Connected to Postgres successfully.")
	}

	// 2. Kafka
	kafkaBroker := "localhost:9092"
	producer, err := event.NewProducer([]string{kafkaBroker})
	if err != nil {
		log.Printf("WARNING: Producer error: %v", err)
	} else {
		defer producer.Close()
	}

	consumer, err := event.NewConsumer([]string{kafkaBroker})
	if err == nil {
		consumer.Start()
	}

	// 3. Hub
	hub := game.NewHub(repository, producer)

	// 4. Routes (Wrapped with CORS)
	
	// WebSocket (Has its own CORS check inside Upgrader, but wrapping doesn't hurt)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		api.ServeWs(hub, w, r)
	})
	
	// Leaderboard (Wrapped!)
	http.HandleFunc("/leaderboard", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		if repository == nil {
			http.Error(w, "Database not available", http.StatusServiceUnavailable)
			return
		}
		api.HandleLeaderboard(repository, w, r)
	}))

	http.HandleFunc("/health", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))

	// 5. Start
	port := "8080"
	log.Printf("Server running on http://localhost:%s", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}