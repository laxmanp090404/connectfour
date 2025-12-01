package api

import (
	"connectfour/internal/db"
	"encoding/json"
	"net/http"
)

func HandleLeaderboard(repo *db.Repository, w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	entries, err := repo.GetLeaderboard()
	if err != nil {
		http.Error(w, "Failed to fetch leaderboard", 500)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}