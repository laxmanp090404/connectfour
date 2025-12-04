package api

import (
	"connectfour/internal/db"
	"encoding/json"
	"net/http"
)

func HandleLeaderboard(repo *db.Repository, w http.ResponseWriter, r *http.Request) {
	// 1. Set CORS Headers 
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 2. Handle Preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// 3. Actual Logic
	entries, err := repo.GetLeaderboard()
	if err != nil {
		http.Error(w, "Failed to fetch leaderboard", 500)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}
