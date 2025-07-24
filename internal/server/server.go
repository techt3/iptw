// Package server provides HTTP server functionality for serving game statistics
package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"iptw/internal/config"
	"iptw/internal/gui"
	"iptw/internal/stats"
)

// Server represents the HTTP server for serving game statistics
type Server struct {
	app    *gui.App
	config *config.Config
	port   string
}

// NewServer creates a new server instance
func NewServer(app *gui.App, cfg *config.Config, port string) *Server {
	if port == "" {
		port = "32782" // Default port
	}
	return &Server{
		app:    app,
		config: cfg,
		port:   port,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	http.HandleFunc("/", s.handleRoot)
	http.HandleFunc("/stats", s.handleStats)
	http.HandleFunc("/stats/json", s.handleStatsJSON)
	http.HandleFunc("/achievements", s.handleAchievements)
	http.HandleFunc("/countries", s.handleCountries)
	http.HandleFunc("/countries/boring", s.handleMarkBoring)
	http.HandleFunc("/wallpaper/restore", s.handleRestoreWallpaper)
	http.HandleFunc("/health", s.handleHealth)

	addr := ":" + s.port
	slog.Info("Starting statistics server", "addr", addr)
	slog.Info("Available endpoints:")
	slog.Info("  GET /           - Server information")
	slog.Info("  GET /stats      - Game statistics (text)")
	slog.Info("  GET /stats/json - Game statistics (JSON)")
	slog.Info("  GET /achievements - Achievement details")
	slog.Info("  GET /countries  - Country visit details")
	slog.Info("  POST /countries/boring - Mark a country as boring")
	slog.Info("  POST /wallpaper/restore - Restore original wallpaper")
	slog.Info("  GET /health     - Health check")

	return http.ListenAndServe(addr, nil)
}

// handleRoot provides basic server information
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "IPTW Statistics Server\n")
	fmt.Fprintf(w, "======================\n\n")
	fmt.Fprintf(w, "Available endpoints:\n")
	fmt.Fprintf(w, "  GET /stats      - Game statistics (text)\n")
	fmt.Fprintf(w, "  GET /stats/json - Game statistics (JSON)\n")
	fmt.Fprintf(w, "  GET /achievements - Achievement details\n")
	fmt.Fprintf(w, "  GET /countries  - Country visit details\n")
	fmt.Fprintf(w, "  POST /countries/boring - Mark a country as boring\n")
	fmt.Fprintf(w, "  GET /health     - Health check\n")
	fmt.Fprintf(w, "\nServer time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}

// handleStats returns game statistics in text format
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	gameStats := s.collectGameStatistics()
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, gameStats.Summary())
}

// handleStatsJSON returns game statistics in JSON format
func (s *Server) handleStatsJSON(w http.ResponseWriter, r *http.Request) {
	gameStats := s.collectGameStatistics()
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := gameStats.ToJSON()
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}

// handleAchievements returns achievement details
func (s *Server) handleAchievements(w http.ResponseWriter, r *http.Request) {
	gameStats := s.collectGameStatistics()
	w.Header().Set("Content-Type", "text/plain")

	fmt.Fprintf(w, "IPTW Achievements (%d/%d unlocked)\n",
		gameStats.UnlockedAchievements, gameStats.TotalAchievements)
	fmt.Fprintf(w, "=====================================\n\n")

	for _, achievement := range gameStats.Achievements {
		status := "🔒"
		if achievement.Unlocked {
			status = "🏆"
		}

		fmt.Fprintf(w, "%s %s\n", status, achievement.Name)
		fmt.Fprintf(w, "   %s\n", achievement.Description)
		fmt.Fprintf(w, "   Progress: %d/%d\n\n", achievement.Progress, achievement.Target)
	}
}

// handleCountries returns country visit details
func (s *Server) handleCountries(w http.ResponseWriter, r *http.Request) {
	gameStats := s.collectGameStatistics()
	w.Header().Set("Content-Type", "text/plain")

	fmt.Fprintf(w, "Country Visit Statistics (%d countries)\n", len(gameStats.Countries))
	fmt.Fprintf(w, "========================================\n\n")

	if len(gameStats.Countries) == 0 {
		fmt.Fprintf(w, "No countries visited yet - start browsing to begin your virtual travels!\n")
		return
	}

	for _, country := range gameStats.Countries {
		status := "🌍"
		if country.Boring {
			status = "😴"
		} else if country.HitCount >= 7 {
			status = "⚠️"
		} else if country.HitCount >= 4 {
			status = "🔥"
		}

		fmt.Fprintf(w, "%s %s: %d visits", status, country.Name, country.HitCount)
		if !country.LastHit.IsZero() {
			fmt.Fprintf(w, " (last: %s)", country.LastHit.Format("2006-01-02 15:04:05"))
		}
		fmt.Fprintf(w, "\n")
	}
}

// handleHealth returns health check information
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"server":    "iptw-stats",
	}
	json.NewEncoder(w).Encode(health)
}

// handleMarkBoring handles POST requests to mark a country as boring
func (s *Server) handleMarkBoring(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.app == nil {
		http.Error(w, "App not available", http.StatusServiceUnavailable)
		return
	}

	// Parse JSON request body
	var request struct {
		Country string `json:"country"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Country == "" {
		http.Error(w, "Country name is required", http.StatusBadRequest)
		return
	}

	// Handle the fastest traveler achievement logic
	s.app.HandleFastestTravelerAchievement(request.Country)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Country '%s' marked as boring", request.Country),
		"country": request.Country,
	}
	json.NewEncoder(w).Encode(response)

	slog.Info("Country marked as boring via API",
		"country", request.Country,
		"client", r.RemoteAddr,
	)
}

// collectGameStatistics gathers current game statistics from the app
func (s *Server) collectGameStatistics() *stats.GameStatistics {
	if s.app == nil {
		return &stats.GameStatistics{
			ServerVersion: "dev",
			Timestamp:     time.Now(),
		}
	}

	// Get game state from the app
	gameState := s.app.GetGameState()
	achievementManager := s.app.GetAchievementManager()

	// Calculate basic statistics
	totalCountries := 0
	totalVisits := 0
	boringCountries := 0
	countries := make([]stats.CountryStats, 0)

	countriesMap := gameState.GetCountries()
	for countryName, countryState := range countriesMap {
		totalCountries++
		totalVisits += countryState.HitCount

		if countryState.Boring {
			boringCountries++
		}

		countries = append(countries, stats.CountryStats{
			Name:     countryName,
			HitCount: countryState.HitCount,
			Boring:   countryState.Boring,
			LastHit:  countryState.LastHit,
		})
	}

	targetCountry, targetSetAt := gameState.GetTargetCountry()

	// Calculate overvisited rate
	overvisitedRate := 0.0
	if totalCountries > 0 {
		overvisitedRate = float64(boringCountries) / float64(totalCountries) * 100
	}

	// Calculate target time remaining
	targetTimeRemaining := time.Duration(0)
	if !targetSetAt.IsZero() && s.config != nil {
		targetTimeRemaining = time.Duration(s.config.TargetInterval)*time.Minute - time.Since(targetSetAt)
		if targetTimeRemaining < 0 {
			targetTimeRemaining = 0
		}
	}

	// Get achievement data
	achievements := make([]stats.Achievement, 0)
	unlockedCount := 0
	totalAchievements := 0

	if achievementManager != nil {
		allAchievements := achievementManager.GetAllAchievements()
		for _, achievement := range allAchievements {
			totalAchievements++
			if achievement.Unlocked {
				unlockedCount++
			}

			achievements = append(achievements, stats.Achievement{
				ID:          achievement.ID,
				Name:        achievement.Name,
				Description: achievement.Description,
				Unlocked:    achievement.Unlocked,
				Progress:    achievement.Progress,
				Target:      achievement.Target,
				Countries:   achievement.Countries,
			})
		}
	}

	return &stats.GameStatistics{
		TotalCountries:       totalCountries,
		TotalVisits:          totalVisits,
		BoringCountries:      boringCountries,
		OvervisitedRate:      overvisitedRate,
		TargetCountry:        targetCountry,
		TargetSetAt:          targetSetAt,
		TargetTimeRemaining:  targetTimeRemaining,
		Countries:            countries,
		Achievements:         achievements,
		UnlockedAchievements: unlockedCount,
		TotalAchievements:    totalAchievements,
		ServerVersion:        "dev",
		Timestamp:            time.Now(),
	}
}

// handleRestoreWallpaper handles POST requests to restore the original wallpaper
func (s *Server) handleRestoreWallpaper(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if app has a backup wallpaper
	if !s.app.HasWallpaperBackup() {
		response := map[string]interface{}{
			"success": false,
			"error":   "No wallpaper backup available",
			"message": "Original wallpaper was not backed up or backup failed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Attempt to restore wallpaper
	if err := s.app.RestoreOriginalWallpaper(); err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"message": "Failed to restore original wallpaper",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Original wallpaper restored successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
