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

	// ==========================================
	// 1. Database Connection (Supabase vs Local)
	// ==========================================
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Fallback to local Docker logic if no cloud URL is found
		dsn = fmt.Sprintf("postgres://%s:%s@localhost:5432/%s?sslmode=disable",
			getEnv("DB_USER", "user"),
			getEnv("DB_PASSWORD", "password"),
			getEnv("DB_NAME", "connect4"),
		)
		log.Println("Using Local DB Configuration")
	} else {
		log.Println("Using Cloud DB Configuration (DATABASE_URL found)")
	}

	repository, err := db.NewRepository(dsn)
	if err != nil {
		log.Printf("WARNING: Could not connect to Database: %v", err)
	} else {
		log.Println("Connected to Postgres successfully.")
	}

	// ==========================================
	// 2. Kafka Connection (Upstash vs Local)
	// ==========================================
	kafkaURL := os.Getenv("KAFKA_URL")
	brokers := []string{"localhost:9092"} // Default to local
	
	if kafkaURL != "" {
		brokers = []string{kafkaURL}
		log.Println("Using Cloud Kafka Configuration")
	} else {
		log.Println("Using Local Kafka Configuration")
	}

	// Initialize Producer
	producer, err := event.NewProducer(brokers)
	if err != nil {
		log.Printf("WARNING: Producer error: %v", err)
	} else {
		log.Println("Kafka Producer Connected")
		defer producer.Close()
	}

	// Initialize Consumer
	consumer, err := event.NewConsumer(brokers)
	if err == nil {
		consumer.Start()
		log.Println("Kafka Consumer Started")
	} else {
		log.Printf("WARNING: Consumer error: %v", err)
	}

	// ==========================================
	// 3. Game Hub Initialization
	// ==========================================
	hub := game.NewHub(repository, producer)

	// ==========================================
	// 4. Routes
	// ==========================================
	
	// WebSocket (Upgrader handles its own CORS, but we wrap it just in case)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		api.ServeWs(hub, w, r)
	})
	
	// Leaderboard
	http.HandleFunc("/leaderboard", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		if repository == nil {
			http.Error(w, "Database not available", http.StatusServiceUnavailable)
			return
		}
		api.HandleLeaderboard(repository, w, r)
	}))

	// Health Check (Used by Render to verify app is alive)
	http.HandleFunc("/health", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))

	// ==========================================
	// 5. Start Server (Dynamic Port for Cloud)
	// ==========================================
	port := getEnv("PORT", "8080") // Render provides PORT, Local defaults to 8080
	
	log.Printf("Server running on http://0.0.0.0:%s", port)
	
	// Listen on 0.0.0.0 instead of just localhost to allow external access in containers
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