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

func main() {
	log.Println("Starting Connect 4 Server...")

	// 1. Connect to Postgres
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

	// 2. Connect to Kafka (Producer & Consumer)
	kafkaBroker := "localhost:9092"
	
	// A. Producer
	producer, err := event.NewProducer([]string{kafkaBroker})
	if err != nil {
		log.Printf("WARNING: Could not connect Kafka Producer: %v", err)
	} else {
		defer producer.Close()
	}

	// B. Consumer (Analytics Service)
	consumer, err := event.NewConsumer([]string{kafkaBroker})
	if err == nil {
		consumer.Start() // Runs in background
	} else {
		log.Printf("WARNING: Could not connect Kafka Consumer: %v", err)
	}

	// 3. Initialize Hub
	hub := game.NewHub(repository, producer)

	// 4. Routes
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		api.ServeWs(hub, w, r)
	})
	
	http.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		if repository == nil {
			http.Error(w, "Database not available", 503)
			return
		}
		api.HandleLeaderboard(repository, w, r)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

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