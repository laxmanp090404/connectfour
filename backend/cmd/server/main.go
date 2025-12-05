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

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func main() {
	log.Println("Starting Connect 4 Server...")

	// 1. Database Connection (Required)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@localhost:5432/%s?sslmode=disable",
			getEnv("DB_USER", "user"),
			getEnv("DB_PASSWORD", "password"),
			getEnv("DB_NAME", "connect4"),
		)
	}

	repository, err := db.NewRepository(dsn)
	if err != nil {
		// It is okay to crash if DB fails, as game state depends on it
		log.Printf("WARNING: Could not connect to Database: %v", err)
	} else {
		log.Println("Connected to Postgres successfully.")
	}

	// 2. Kafka Connection (Optional - Graceful Degradation)
	var producer *event.Producer
	kafkaURL := os.Getenv("KAFKA_URL")
	
	if kafkaURL != "" {
		// Cloud / Configured Mode
		log.Println("Attempting to connect to Cloud Kafka...")
		brokers := []string{kafkaURL}
		p, err := event.NewProducer(brokers)
		if err != nil {
			log.Printf("⚠️ WARNING: Kafka Connection Failed (%v). Analytics will be disabled.", err)
		} else {
			producer = p
			log.Println("✅ Kafka Producer Connected")
			defer producer.Close()

			// Only start Consumer if Producer worked
			consumer, err := event.NewConsumer(brokers)
			if err == nil {
				consumer.Start()
				log.Println("✅ Kafka Consumer Started")
			}
		}
	} else if os.Getenv("KAFKA_ENABLE_LOCAL") == "true" {
		// Local Docker Mode
		log.Println("Connecting to Local Docker Kafka...")
		p, err := event.NewProducer([]string{"localhost:9092"})
		if err == nil {
			producer = p
			log.Println("✅ Local Kafka Connected")
			defer producer.Close()
			
			c, err := event.NewConsumer([]string{"localhost:9092"})
			if err == nil { c.Start() }
		}
	} else {
		log.Println("ℹ️ Kafka URL not found. Running in NO-ANALYTICS mode.")
	}

	// 3. Hub (Handles nil producer gracefully)
	hub := game.NewHub(repository, producer)

	// 4. Routes
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		api.ServeWs(hub, w, r)
	})
	
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
	port := getEnv("PORT", "8080")
	log.Printf("Server running on http://0.0.0.0:%s", port)
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