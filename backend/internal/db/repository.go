package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq" // Postgres Driver
)

type Repository struct {
	db *sql.DB
}

type LeaderboardEntry struct {
	Username string `json:"username"`
	Wins     int    `json:"wins"`
}

func NewRepository(dsn string) (*Repository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Auto-Migration: Create table if not exists
	query := `
	CREATE TABLE IF NOT EXISTS games (
		id UUID PRIMARY KEY,
		player1 TEXT NOT NULL,
		player2 TEXT NOT NULL,
		winner TEXT,
		reason TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	
	_, err = db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate db: %v", err)
	}

	return &Repository{db: db}, nil
}

func (r *Repository) SaveGame(gameID, p1, p2, winner, reason string) {
	query := `INSERT INTO games (id, player1, player2, winner, reason) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.Exec(query, gameID, p1, p2, winner, reason)
	if err != nil {
		log.Printf("ERROR: Failed to save game to DB: %v", err)
	} else {
		log.Printf("DB: Game %s saved successfully.", gameID)
	}
}

func (r *Repository) GetLeaderboard() ([]LeaderboardEntry, error) {
	// Simple query: Count wins per user (excluding 'Draw')
	query := `
		SELECT winner, COUNT(*) as wins 
		FROM games 
		WHERE winner != 'Draw' 
		GROUP BY winner 
		ORDER BY wins DESC 
		LIMIT 10;
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.Username, &e.Wins); err != nil {
			continue
		}
		leaderboard = append(leaderboard, e)
	}
	return leaderboard, nil
}