// Package stats provides statistics data structures and utilities for server-client communication
package stats

import (
	"encoding/json"
	"fmt"
	"time"
)

// CountryStats represents statistics for a single country
type CountryStats struct {
	Name     string    `json:"name"`
	HitCount int       `json:"hit_count"`
	Boring   bool      `json:"boring"`
	LastHit  time.Time `json:"last_hit"`
}

// Achievement represents an achievement with progress information
type Achievement struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Unlocked    bool     `json:"unlocked"`
	Progress    int      `json:"progress"`
	Target      int      `json:"target"`
	Countries   []string `json:"countries,omitempty"`
}

// GameStatistics represents the complete game statistics for client consumption
type GameStatistics struct {
	// Basic stats
	TotalCountries  int     `json:"total_countries"`
	TotalVisits     int     `json:"total_visits"`
	BoringCountries int     `json:"boring_countries"`
	OvervisitedRate float64 `json:"overvisited_rate"`

	// Current target
	TargetCountry       string        `json:"target_country"`
	TargetSetAt         time.Time     `json:"target_set_at"`
	TargetTimeRemaining time.Duration `json:"target_time_remaining"`

	// Detailed country data
	Countries []CountryStats `json:"countries"`

	// Achievement data
	Achievements         []Achievement `json:"achievements"`
	UnlockedAchievements int           `json:"unlocked_achievements"`
	TotalAchievements    int           `json:"total_achievements"`

	// Server info
	ServerVersion string    `json:"server_version"`
	Timestamp     time.Time `json:"timestamp"`
}

// ToJSON converts GameStatistics to JSON bytes
func (gs *GameStatistics) ToJSON() ([]byte, error) {
	return json.MarshalIndent(gs, "", "  ")
}

// FromJSON creates GameStatistics from JSON bytes
func FromJSON(data []byte) (*GameStatistics, error) {
	var gs GameStatistics
	err := json.Unmarshal(data, &gs)
	return &gs, err
}

// Summary provides a brief text summary of the statistics
func (gs *GameStatistics) Summary() string {
	if gs.TotalCountries == 0 {
		return "No countries visited yet - start browsing to begin your virtual travels!"
	}

	summary := ""
	summary += fmt.Sprintf("🌍 Countries Visited: %d\n", gs.TotalCountries)
	summary += fmt.Sprintf("📊 Total Visits: %d\n", gs.TotalVisits)
	summary += fmt.Sprintf("😴 Boring Countries: %d (%.1f%%)\n", gs.BoringCountries, gs.OvervisitedRate)
	summary += fmt.Sprintf("🏆 Achievements Unlocked: %d/%d\n", gs.UnlockedAchievements, gs.TotalAchievements)

	if gs.TargetCountry != "" {
		if gs.TargetTimeRemaining > 0 {
			summary += fmt.Sprintf("🎯 Current Target: %s (%.1f minutes remaining)\n",
				gs.TargetCountry, gs.TargetTimeRemaining.Minutes())
		} else {
			summary += fmt.Sprintf("⚠️  Target Overdue: %s\n", gs.TargetCountry)
		}
	} else {
		summary += "🎯 No active target - all countries hit!\n"
	}

	return summary
}
