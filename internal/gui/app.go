// Package gui provides the graphical user interface for IP Travel Wallpaper.
//
// Travel Mechanics:
// - Each network connection to a foreign IP address represents a "visit" to that country
// - Countries are colored based on the number of visits:
//   - 1-9 visits: Display national flag background (fresh destinations worth exploring)
//   - 10+ visits: Country becomes "boring" and displays sand/rocks gradient pattern
//
// - Countries visited too many times become boring and stop counting additional visits
// - The goal is to "travel" to different countries by generating network traffic to IPs in those countries
// - Target countries are highlighted with red borders to encourage visiting new places
//
// Achievements:
// - Visit all countries in a geographic region (Europe, Asia, Africa, etc.)
// - Visit all countries on a continent
// - Discover rare or remote countries
//
// Visual Elements:
// - World map background (Natural Earth vector data)
// - Country regions filled with colors/patterns based on visit counts
// - White dots show active connection points
// - National flags display for visited countries (1-9 hits)
// - Sand/rocks gradient patterns show for boring countries (10+ hits)
// - Red borders highlight target countries for exploration
//
// Resources:
// - Natural Earth GeoJSON data is embedded in the binary using Go's embed package
// - Vector graphics provide crisp rendering at any resolution
// - Theme support: light and dark backgrounds
// - No external resource files required - completely self-contained application
package gui

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log/slog"
	"math"
	mathrand "math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"

	"iptw/internal/achievements"
	"iptw/internal/background"
	"iptw/internal/config"
	"iptw/internal/geoip"
	"iptw/internal/logging"
	"iptw/internal/network"
	"iptw/internal/resources"
	"iptw/internal/screen"
	"iptw/internal/service"
)

//go:embed map.html
var mapHTMLContent []byte

// CountryGameState represents the game state for a country
type CountryGameState struct {
	HitCount int
	Boring   bool
	LastHit  time.Time
}

// RecentHit represents a recent network connection for the UI
type RecentHit struct {
	Country  string    `json:"country"`
	City     string    `json:"city"`
	Protocol string    `json:"protocol"`
	Domain   string    `json:"domain"`
	Time     time.Time `json:"time"`
}

// GameState manages the overall game state
type GameState struct {
	countries     map[string]*CountryGameState
	targetCountry string    // Currently targeted country
	targetSetAt   time.Time // When the target was set
	mutex         sync.RWMutex
}

// AddCountryHit adds a hit to a country
func (gs *GameState) AddCountryHit(country string) {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if gs.countries[country] == nil {
		gs.countries[country] = &CountryGameState{}
	}

	countryState := gs.countries[country]
	if !countryState.Boring {
		countryState.HitCount++
		countryState.LastHit = time.Now()

		// Mark as boring if hits >= 10
		if countryState.HitCount >= 10 {
			countryState.Boring = true
		}
	}
}

// AddCountryHitWithTargetCheck adds a hit to a country and returns if it became boring and was the target
func (gs *GameState) AddCountryHitWithTargetCheck(country string) (becameBoring bool, wasTarget bool) {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if gs.countries[country] == nil {
		gs.countries[country] = &CountryGameState{}
	}

	countryState := gs.countries[country]
	countryState.HitCount++
	countryState.LastHit = time.Now()

	if !countryState.Boring {
		// Mark as boring if hits >= 10
		if countryState.HitCount >= 10 {
			countryState.Boring = true
			becameBoring = true
			wasTarget = gs.targetCountry == country

			if wasTarget {
				// Clear the target since it's now boring
				gs.targetCountry = ""
				gs.targetSetAt = time.Time{}
			}
		}
	}

	return becameBoring, wasTarget
}

// MarkCountryAsBoring marks a country as boring and returns whether it was the target country
func (gs *GameState) MarkCountryAsBoring(country string) (wasTarget bool, targetCountry string) {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if gs.countries[country] == nil {
		gs.countries[country] = &CountryGameState{}
	}

	countryState := gs.countries[country]
	if !countryState.Boring {
		countryState.Boring = true
		countryState.LastHit = time.Now()

		// Check if this was the target country
		wasTarget = gs.targetCountry == country
		targetCountry = gs.targetCountry

		if wasTarget {
			// Clear the target since it's now boring
			gs.targetCountry = ""
			gs.targetSetAt = time.Time{}
		}
	}

	return wasTarget, targetCountry
}

// GetCountryState returns the state of a country
func (gs *GameState) GetCountryState(country string) *CountryGameState {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	if state, exists := gs.countries[country]; exists {
		// Return a copy to avoid race conditions
		return &CountryGameState{
			HitCount: state.HitCount,
			Boring:   state.Boring,
			LastHit:  state.LastHit,
		}
	}
	return nil
}

// HasCountry checks if a country has been visited (exists in the countries map)
func (gs *GameState) HasCountry(country string) bool {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	_, exists := gs.countries[country]
	return exists
}

// GetCountries returns a copy of the countries map for server access
func (gs *GameState) GetCountries() map[string]*CountryGameState {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	countries := make(map[string]*CountryGameState)
	for name, state := range gs.countries {
		countries[name] = &CountryGameState{
			HitCount: state.HitCount,
			Boring:   state.Boring,
			LastHit:  state.LastHit,
		}
	}
	return countries
}

// RLock provides read access to the mutex for server operations
func (gs *GameState) RLock() {
	gs.mutex.RLock()
}

// RUnlock provides read unlock for the mutex
func (gs *GameState) RUnlock() {
	gs.mutex.RUnlock()
}

// GetCountryColor returns the color for a country based on its hit count
func (gs *GameState) GetCountryColor(country string) color.RGBA {
	state := gs.GetCountryState(country)
	if state == nil || state.HitCount == 0 {
		return color.RGBA{0, 0, 0, 0} // Transparent for no hits
	}

	if state.Boring {
		// Bright red for boring countries
		return color.RGBA{255, 50, 50, 200}
	}

	// Progressive color intensity based on hit count (1-9)
	// Colors progress from light yellow (1 hit) to bright orange (9 hits)
	intensity := float64(state.HitCount) / 9.0 // Normalize to 0-1 for hits 1-9

	// Color progression: Light Yellow -> Orange -> Dark Orange
	red := uint8(255)
	green := uint8(255 - intensity*150) // Fade from 255 to 105
	blue := uint8(50 * (1 - intensity)) // Fade from 50 to 0
	alpha := uint8(80 + intensity*100)  // Alpha from 80 to 180

	return color.RGBA{red, green, blue, alpha}
}

// App represents the main application
type App struct {
	config                 *config.Config
	configMu               sync.RWMutex // protects concurrent access to config fields
	geoip                  *geoip.Database
	monitor                *network.Monitor
	worldMap               image.Image
	running                bool
	outputDir              string
	gameState              *GameState
	naturalEarth           *resources.NaturalEarthData
	achievements           *achievements.AchievementManager
	fontManager            *resources.FontManager
	flagManager            *resources.FlagManager
	originalWallpaper      string // Path to the backed up original wallpaper
	wallpaperBackedUp      bool   // Flag to track if we've backed up the wallpaper
	wallpaperBackedUpError error
	lastMapPNG             []byte       // Cached PNG bytes of the last generated map image
	mapPNGMu               sync.RWMutex // protects lastMapPNG
	sessionToken           string       // Per-session token for POST endpoint authorization
	serverURL              string       // URL of the local HTTP server
	serverListener         net.Listener // Local HTTP listener; closed on shutdown
	lastAutoWidth          int          // Memoized screen detection width
	lastAutoHeight         int          // Memoized screen detection height
	recentHits             []RecentHit  // Store the last few hits for the UI
	recentHitsMu           sync.RWMutex // protects recentHits
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config, geoipDB *geoip.Database, monitor *network.Monitor) (*App, error) {
	homeDir, _ := os.UserHomeDir()
	outputDir := filepath.Join(homeDir, ".config", "iptw", "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	gameState := &GameState{
		countries: make(map[string]*CountryGameState),
	}

	// Load Natural Earth data (required)
	naturalEarth, err := resources.LoadNaturalEarthData()
	if err != nil {
		return nil, fmt.Errorf("failed to load Natural Earth data: %w", err)
	}
	logging.LogNaturalEarth(len(naturalEarth.Countries))

	// Load embedded fonts
	fontManager, err := resources.LoadFonts()
	if err != nil {
		return nil, fmt.Errorf("failed to load fonts: %w", err)
	}

	// Load flag bitmaps (optional - if failed, flags won't be used for boring countries)
	flagManager, err := resources.LoadFlags()
	if err != nil {
		slog.Warn("Failed to load flag bitmaps - flag backgrounds will not be available", "error", err)
		flagManager = nil // Continue without flags
	} else {
		slog.Info("Flag bitmaps loaded successfully", "count", len(flagManager.ListFlags()))
	}

	// Generate a per-session token for POST endpoint authorization
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}
	generatedToken := hex.EncodeToString(tokenBytes)

	// Check for existing wallpaper backups in the output directory
	var firstBackup string
	if files, err := filepath.Glob(filepath.Join(outputDir, "original_wallpaper_*")); err == nil && len(files) > 0 {
		// Sort files to pick the earliest backup
		sort.Strings(files)
		firstBackup = files[0]
		slog.Info("🔍 Found existing wallpaper backup", "path", firstBackup)
	}

	return &App{
		config:            cfg,
		geoip:             geoipDB,
		monitor:           monitor,
		running:           true,
		outputDir:         outputDir,
		gameState:         gameState,
		naturalEarth:      naturalEarth,
		achievements:      achievements.NewAchievementManager(),
		fontManager:       fontManager,
		flagManager:       flagManager,
		originalWallpaper: firstBackup,
		wallpaperBackedUp: firstBackup != "",
		sessionToken:      generatedToken,
	}, nil
}

// Run starts the application
func (a *App) Run() error {
	// Start systray lifecycle. This blocks until systray quits.
	systray.Run(a.onReady, a.onExit)
	return nil
}

func (a *App) onReady() {
	// Log startup information
	slog.Info("IP Travel Wallpaper started",
		"screen_auto_detection", a.config.AutoDetectScreen,
		"log_level", a.config.LogLevel,
		"target_interval_minutes", a.config.TargetInterval,
		"update_wallpaper", a.config.UpdateWallpaper,
		"start_on_login", a.config.StartOnLogin,
	)
	if !a.config.AutoDetectScreen {
		slog.Debug("Manual map dimensions configured",
			"width", a.config.MapWidth,
			"height", a.config.MapWidth/2,
		)
	}

	// Setup Systray
	setTrayTitleAndTooltip("IPTW", "IP Travel Map")
	// TODO: set an actual icon bytes array, for now using a placeholder text if no icon

	mShowMap := systray.AddMenuItem("Show Map", "Open the interactive travel map")
	mToggleWallpaper := systray.AddMenuItemCheckbox("Update OS Wallpaper", "Automatically update desktop wallpaper", a.config.UpdateWallpaper)
	mStartOnLogin := systray.AddMenuItemCheckbox("Start on Login", "Automatically start IP Travel Map on system login", a.config.StartOnLogin)
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	// Load world map
	if err := a.loadWorldMap(); err != nil {
		slog.Error("failed to load world map", "error", err)
		systray.Quit()
		return
	}

	slog.Info("World map loaded successfully")

	// Start connection monitoring
	go a.connectionMonitorLoop()

	// Start target country selection loop
	go a.targetSelectionLoop()

	// Start image generation and display loop
	go a.displayLoop()

	// Start local HTTP server to host the UI
	go a.startLocalServer()

	// Handle Menu events
	go func() {
		for {
			select {
			case <-mShowMap.ClickedCh:
				a.showMapWindow()
			case <-mToggleWallpaper.ClickedCh:
				a.configMu.Lock()
				a.config.UpdateWallpaper = !a.config.UpdateWallpaper
				newVal := a.config.UpdateWallpaper
				a.configMu.Unlock()
				if newVal {
					mToggleWallpaper.Check()
				} else {
					mToggleWallpaper.Uncheck()
					// Restore original wallpaper if we backed it up
					if a.HasWallpaperBackup() {
						if err := a.RestoreOriginalWallpaper(); err != nil {
							slog.Error("Failed to restore original wallpaper", "error", err)
						}
					}
				}
				if err := a.saveConfig(); err != nil {
					slog.Error("Failed to save config after toggling wallpaper", "error", err)
				}
			case <-mStartOnLogin.ClickedCh:
				a.configMu.Lock()
				a.config.StartOnLogin = !a.config.StartOnLogin
				newVal := a.config.StartOnLogin
				a.configMu.Unlock()
				if newVal {
					mStartOnLogin.Check()
					if sm, err := service.NewServiceManager(); err == nil {
						if err := sm.Install(); err != nil {
							slog.Error("Failed to install auto-start service", "error", err)
						}
					}
				} else {
					mStartOnLogin.Uncheck()
					if sm, err := service.NewServiceManager(); err == nil {
						if err := sm.Uninstall(); err != nil {
							slog.Error("Failed to uninstall auto-start service", "error", err)
						}
					}
				}
				if err := a.saveConfig(); err != nil {
					slog.Error("Failed to save config after toggling start-on-login", "error", err)
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (a *App) onExit() {
	slog.Info("Received shutdown signal from systray, cleaning up...")
	a.Shutdown()
}

// saveConfig saves the current configuration to disk. It resolves the config
// path consistently with LoadConfig.
func (a *App) saveConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configPath := filepath.Join(homeDir, ".config", "iptw", "iptwrc")
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	return a.config.Save(configPath)
}

// requireSessionToken is middleware for POST endpoints that validates the
// per-session token to prevent cross-site request forgery from malicious pages.
func (a *App) requireSessionToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Session-Token")
		if token == "" {
			token = r.URL.Query().Get("_t")
		}
		if token != a.sessionToken {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (a *App) startLocalServer() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		slog.Error("Failed to bind local HTTP server", "error", err)
		return
	}

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		slog.Error("Unexpected listener address type for local HTTP server")
		if err := ln.Close(); err != nil {
			slog.Warn("Failed to close temporary listener", "error", err)
		}
		return
	}

	a.serverListener = ln
	a.serverURL = fmt.Sprintf("http://127.0.0.1:%d", tcpAddr.Port)
	slog.Info("Starting local HTTP server for UI", "url", a.serverURL)

	mux := http.NewServeMux()

	// Serve the embedded map HTML, injecting the session token so JS can use it
	mux.HandleFunc("/map.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Inject the session token as a JS variable before </head>
		injected := bytes.Replace(
			mapHTMLContent,
			[]byte("</head>"),
			[]byte(fmt.Sprintf(`<script>window._sessionToken = %q;</script></head>`, a.sessionToken)),
			1,
		)
		if _, err := w.Write(injected); err != nil {
			slog.Error("Failed to write map HTML", "error", err)
		}
	})

	// Serve the latest map image as PNG bytes directly (no double-encoding)
	mux.HandleFunc("/api/map.png", func(w http.ResponseWriter, r *http.Request) {
		a.mapPNGMu.RLock()
		pngBytes := a.lastMapPNG
		a.mapPNGMu.RUnlock()
		if len(pngBytes) == 0 {
			http.Error(w, "Map not yet generated", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-cache")
		if _, err := w.Write(pngBytes); err != nil {
			slog.Error("Failed to write map PNG", "error", err)
		}
	})

	// Mark a country as boring manually
	mux.HandleFunc("/countries/boring", a.requireSessionToken(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var data struct {
			Country string `json:"country"`
		}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if data.Country == "" {
			http.Error(w, "Country name is required", http.StatusBadRequest)
			return
		}

		wasTarget, _ := a.gameState.MarkCountryAsBoring(data.Country)
		if wasTarget {
			a.achievements.UnlockFastestTravelerAchievement(data.Country)
		}

		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Country %s marked as boring", data.Country),
		}); err != nil {
			slog.Error("Failed to encode boring country response", "error", err)
		}
	}))

	// Restore original wallpaper
	mux.HandleFunc("/wallpaper/restore", a.requireSessionToken(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if a.originalWallpaper == "" {
			http.Error(w, "No original wallpaper backup found", http.StatusNotFound)
			return
		}

		if err := a.RestoreOriginalWallpaper(); err != nil {
			slog.Error("Failed to restore wallpaper via API", "error", err)
			http.Error(w, "Failed to restore wallpaper", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Original wallpaper restored successfully",
		}); err != nil {
			slog.Error("Failed to encode wallpaper restore response", "error", err)
		}
	}))

	// Serve game statistics and recent hits
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		// Get game statistics
		a.gameState.mutex.RLock()
		visitedCount := len(a.gameState.countries)
		boringCount := 0

		// Identify inMatrixPrison state
		inMatrixPrison := false
		connections := a.monitor.GetConnections()
		recentCountries := make(map[string]bool)
		for _, conn := range connections {
			loc, err := a.geoip.Lookup(conn.RemoteIP)
			if err == nil {
				country := a.naturalEarth.FindCountryAtPoint(loc.Latitude, loc.Longitude)
				if country != "" {
					recentCountries[country] = true
				}
			}
		}

		type summaryItem struct {
			Country string `json:"country"`
			Hits    int    `json:"hits"`
		}
		var topCountries []summaryItem

		for country, state := range a.gameState.countries {
			if state.Boring {
				boringCount++
				if recentCountries[country] {
					inMatrixPrison = true
				}
			}
			topCountries = append(topCountries, summaryItem{Country: country, Hits: state.HitCount})
		}
		targetCountry, _ := a.gameState.GetTargetCountry()
		a.gameState.mutex.RUnlock()

		// Sort top countries by hits descending
		sort.Slice(topCountries, func(i, j int) bool {
			return topCountries[i].Hits > topCountries[j].Hits
		})
		if len(topCountries) > 10 {
			topCountries = topCountries[:10]
		}

		// Get achievement count
		achievementCount := len(a.achievements.GetUnlockedAchievements())

		// Get recent hits
		a.recentHitsMu.RLock()
		recentHits := make([]RecentHit, len(a.recentHits))
		copy(recentHits, a.recentHits)
		a.recentHitsMu.RUnlock()

		data := map[string]interface{}{
			"visited_count":     visitedCount,
			"boring_count":      boringCount,
			"achievement_count": achievementCount,
			"target_country":    targetCountry,
			"in_matrix_prison":  inMatrixPrison,
			"recent_hits":       recentHits,
			"top_countries":     topCountries,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("Failed to encode game stats response", "error", err)
		}
	})

	// Redirect root to map.html
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/map.html", http.StatusSeeOther)
	})

	if err := http.Serve(ln, mux); err != nil {
		slog.Error("Local HTTP server stopped", "error", err)
	}
}

func (a *App) showMapWindow() {
	if a.serverURL == "" {
		slog.Warn("Local HTTP server not yet ready, cannot open map")
		return
	}
	if err := open.Run(a.serverURL); err != nil {
		slog.Error("Failed to open browser", "error", err)
	}
}

// loadWorldMap loads the world map image from Natural Earth data
func (a *App) loadWorldMap() error {
	var width, height int

	// Use screen auto-detection if enabled
	if a.config.AutoDetectScreen {
		var err error
		width, height, err = screen.AutoDetectMapSize()
		if err != nil {
			slog.Warn("Screen auto-detection failed, falling back to configured size", "error", err)
			slog.Debug("Using fallback map dimensions")
			width = a.config.MapWidth
			if width <= 0 {
				width = 1000
			}
			height = width / 2
		}
	} else {
		// Use configured map width
		width = a.config.MapWidth
		if width <= 0 {
			width = 1000
		}
		height = width / 2
	}

	// Natural Earth data is required
	if a.naturalEarth == nil {
		return fmt.Errorf("natural Earth data not available")
	}

	// Create initial empty hit map for rendering
	hitCountries := make(map[string]int)
	boringCountries := a.getBoringCountries()

	img, err := resources.RenderNaturalEarthMap(a.naturalEarth, width, height, a.config.Black, hitCountries, "", a.flagManager, a.fontManager, boringCountries, nil)
	if err != nil {
		return fmt.Errorf("failed to render Natural Earth map: %w", err)
	}

	a.worldMap = img
	logging.LogMapRender(width, height, "Natural Earth")
	return nil
}

// connectionMonitorLoop periodically updates network connections
func (a *App) connectionMonitorLoop() {
	ticker := time.NewTicker(time.Duration(a.config.UpdateInterval) * time.Second)
	defer ticker.Stop()

	for a.running {
		<-ticker.C
		if err := a.monitor.RefreshConnections(); err != nil {
			logging.LogError("refresh connections", err)
		}
	}
}

// targetSelectionLoop periodically selects new target countries
func (a *App) targetSelectionLoop() {
	// Set initial target
	a.SelectRandomTargetCountry()

	ticker := time.NewTicker(time.Duration(a.config.TargetInterval) * time.Minute)
	defer ticker.Stop()

	for a.running {
		<-ticker.C
		a.SelectRandomTargetCountry()
		slog.Debug("New target country selected", "country", a.gameState.targetCountry)
	}
}

// displayLoop generates and displays the map
func (a *App) displayLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	statsTicker := time.NewTicker(10 * time.Second)
	defer statsTicker.Stop()

	for a.running {
		select {
		case <-ticker.C:
			if err := a.generateAndDisplayMap(); err != nil {
				logging.LogError("generate map", err)
			}
		case <-statsTicker.C:
			a.logGameStats()
		}
	}
}

// generateAndDisplayMap creates the map image with country fills and displays it
func (a *App) generateAndDisplayMap() error {
	var width, height int

	// Use screen auto-detection if enabled
	if a.config.AutoDetectScreen {
		detWidth, detHeight, err := screen.AutoDetectMapSize()
		if err != nil {
			// Fall back to configured size
			width = a.config.MapWidth
			if width <= 0 {
				width = 1000
			}
			height = width / 2
		} else {
			// Stabilize resolution: only update if change is > 10 pixels to prevent jitter
			if a.lastAutoWidth == 0 || math.Abs(float64(detWidth-a.lastAutoWidth)) > 10 || math.Abs(float64(detHeight-a.lastAutoHeight)) > 10 {
				a.lastAutoWidth = detWidth
				a.lastAutoHeight = detHeight
			}
			width = a.lastAutoWidth
			height = a.lastAutoHeight
		}
	} else {
		// Use configured map width
		width = a.config.MapWidth
		if width <= 0 {
			width = 1000
		}
		height = width / 2
	}

	// Process connections and update country hit counts
	connections := a.monitor.GetConnections()
	recentCountries := make(map[string]bool)

	for _, conn := range connections {
		location, err := a.geoip.Lookup(conn.RemoteIP)
		if err != nil {
			continue
		}

		// Determine country using Natural Earth data if available
		var countryName string
		if a.naturalEarth != nil {
			// Use Natural Earth for precise country detection
			countryName = a.naturalEarth.FindCountryAtPoint(location.Latitude, location.Longitude)
			if countryName == "" && location.Country != "" {
				// Fall back to GeoIP country if Natural Earth doesn't find it
				countryName = location.Country
			}
		} else {
			// Use GeoIP country data
			countryName = location.Country
		}

		if countryName == "" {
			continue
		}

		// Add hit to country (only once per update cycle per country)
		if !recentCountries[countryName] {
			// Update location country to match Natural Earth result for logging
			if a.naturalEarth != nil {
				location.Country = countryName
			}

			// Log the hit with detailed information
			a.logHit(conn, location, width, height)

			// Check if this is the first visit to this country
			wasFirstVisit := !a.gameState.HasCountry(countryName)

			// Use the new method that checks for target status
			becameBoring, wasTarget := a.gameState.AddCountryHitWithTargetCheck(countryName)
			recentCountries[countryName] = true

			// Handle fastest traveler achievement if country became boring and was target
			if becameBoring && wasTarget {
				achievementID := a.achievements.UnlockFastestTravelerAchievement(countryName)

				if achievementID != "" {
					slog.Info("🚀 Fastest Traveler Achievement earned automatically!",
						"country", countryName,
						"achievement_id", achievementID,
						"reason", "reached_10_hits_while_target",
					)
				}

				// Immediately select a new target country
				a.SelectRandomTargetCountry()

				newTarget, _ := a.gameState.GetTargetCountry()
				if newTarget != "" {
					slog.Info("🎯 New target selected after automatic fastest traveler achievement",
						"new_target", newTarget,
						"previous_target", countryName,
					)
				}
			}

			// Update achievements if this was the first visit to this country
			if wasFirstVisit {
				totalCountriesVisited := len(a.gameState.countries)
				newUnlocks := a.achievements.UpdateProgress(countryName, totalCountriesVisited)

				// Log any new achievement unlocks
				for _, achievementID := range newUnlocks {
					slog.Info("🏆 Achievement unlocked!", "achievement_id", achievementID)
				}
			}
		}
	}

	var outputImg image.Image
	var err error

	// Generate map with Natural Earth data if available
	if a.naturalEarth != nil {
		// Get current hit counts for all countries
		hitCountries := make(map[string]int)
		a.gameState.mutex.RLock()
		for country, state := range a.gameState.countries {
			hitCountries[country] = state.HitCount
		}
		targetCountry, _ := a.gameState.GetTargetCountry()
		a.gameState.mutex.RUnlock()

		// Get boring countries for flag backgrounds
		boringCountries := a.getBoringCountries()

		// Render map with Natural Earth data
		outputImg, err = resources.RenderNaturalEarthMap(a.naturalEarth, width, height, a.config.Black, hitCountries, targetCountry, a.flagManager, a.fontManager, boringCountries, recentCountries)
		if err != nil {
			logging.LogError("render Natural Earth map", err)
			return err
		}
	} else {
		// Fall back to drawing on the pre-loaded world map
		if a.worldMap == nil {
			return fmt.Errorf("world map not loaded")
		}

		bounds := a.worldMap.Bounds()
		mapWidth := bounds.Dx()
		mapHeight := bounds.Dy()

		// Create output image
		outputImg = image.NewRGBA(image.Rect(0, 0, mapWidth, mapHeight))
		draw.Draw(outputImg.(*image.RGBA), outputImg.Bounds(), a.worldMap, image.Point{}, draw.Src)

		// Draw country fills based on hit counts (legacy method)
		a.drawCountryFills(outputImg.(*image.RGBA), mapWidth, mapHeight)
	}

	// Draw connection points for active connections
	rgbaImg, ok := outputImg.(*image.RGBA)
	if !ok {
		// Convert to RGBA if necessary
		bounds := outputImg.Bounds()
		rgbaImg = image.NewRGBA(bounds)
		draw.Draw(rgbaImg, bounds, outputImg, bounds.Min, draw.Src)
	}

	for _, conn := range connections {
		location, err := a.geoip.Lookup(conn.RemoteIP)
		if err != nil {
			continue
		}

		// Convert lat/lng to map coordinates
		x, y := a.latLngToMapCoords(location.Latitude, location.Longitude, width, height)

		// Draw small connection point
		a.drawCircle(rgbaImg, int(x), int(y), 2, color.RGBA{255, 255, 255, 255})
	}

	// Draw game status rectangle
	a.drawGameStatusRectangle(rgbaImg, width, height, recentCountries)

	// Encode once to buffer, then write to disk and cache for the HTTP server
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, rgbaImg); err != nil {
		return fmt.Errorf("failed to encode map image: %w", err)
	}
	pngBytes := buf.Bytes()

	// Save the image to output path
	outputPath := filepath.Join(a.outputDir, "iptw.png")
	if err := os.WriteFile(outputPath, pngBytes, 0644); err != nil {
		return fmt.Errorf("failed to save map image: %w", err)
	}

	// Cache PNG bytes for the local HTTP server (protected by mutex to prevent races)
	a.mapPNGMu.Lock()
	a.lastMapPNG = pngBytes
	a.mapPNGMu.Unlock()

	// Diagnostic: Save periodic snapshots (every 10 seconds for verification)
	if time.Now().Unix()%10 == 0 {
		diagDir := filepath.Join(a.outputDir, "..", "diagnostics")
		_ = os.MkdirAll(diagDir, 0755)
		diagPath := filepath.Join(diagDir, fmt.Sprintf("map_%s.png", time.Now().Format("20060102_150405")))
		if err := os.WriteFile(diagPath, pngBytes, 0644); err == nil {
			slog.Debug("Diagnostic snapshot saved", "path", diagPath)
		}
	}

	a.configMu.RLock()
	updateWallpaper := a.config.UpdateWallpaper
	a.configMu.RUnlock()

	if updateWallpaper {
		// Backup original wallpaper before first change
		if !a.wallpaperBackedUp && a.wallpaperBackedUpError == nil {
			backupPath, err := background.BackupCurrentWallpaper(a.outputDir)
			if err != nil {
				slog.Warn("Failed to backup original wallpaper - restore functionality will not be available", "error", err)
				a.wallpaperBackedUpError = err

			} else {
				a.originalWallpaper = backupPath
				a.wallpaperBackedUp = true
				slog.Info("💾 Original wallpaper backed up successfully")
			}
		}

		// Display using macOS Preview or similar
		if err := background.SetDesktopBackground(outputPath); err != nil {
			slog.Error("Failed to set desktop background", "error", err)
		}
	}

	return nil
}

// latLngToMapCoords converts latitude/longitude to map pixel coordinates
func (a *App) latLngToMapCoords(lat, lng float64, mapWidth, mapHeight int) (float64, float64) {
	// Convert longitude (-180 to 180) to x coordinate (0 to width)
	x := (lng + 180) * float64(mapWidth) / 360

	// Convert latitude (90 to -90) to y coordinate (0 to height)
	y := (90 - lat) * float64(mapHeight) / 180

	return x, y
}

// drawCircle draws a filled circle on the image
func (a *App) drawCircle(img *image.RGBA, centerX, centerY, radius int, col color.Color) {
	for y := centerY - radius; y <= centerY+radius; y++ {
		for x := centerX - radius; x <= centerX+radius; x++ {
			if (x-centerX)*(x-centerX)+(y-centerY)*(y-centerY) <= radius*radius {
				if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
					img.Set(x, y, col)
				}
			}
		}
	}
}

// drawGameStatusRectangle draws a game status rectangle with game information using embedded fonts
func (a *App) drawGameStatusRectangle(img *image.RGBA, mapWidth, mapHeight int, recentCountries map[string]bool) {
	// Get game statistics
	a.gameState.mutex.RLock()
	visitedCount := len(a.gameState.countries)
	boringCount := 0
	targetCountry, _ := a.gameState.GetTargetCountry()
	a.gameState.mutex.RUnlock()

	// Get achievement count
	unlockedAchievements := a.achievements.GetUnlockedAchievements()
	achievementCount := len(unlockedAchievements)

	// Prepare text lines for the game status
	lines := []string{
		"GAME STATUS",
		fmt.Sprintf("Countries visited: %d", visitedCount),
		fmt.Sprintf("Achievements: %d", achievementCount),
	}

	// Add target country line
	if targetCountry != "" {
		if len(targetCountry) > 20 {
			lines = append(lines, fmt.Sprintf("Let's visit: %.17s...", targetCountry))
		} else {
			lines = append(lines, fmt.Sprintf("Let's visit: %s", targetCountry))
		}
	} else {
		lines = append(lines, "Let's visit: None")
	}

	// Add status message
	if visitedCount == 0 {
		lines = append(lines, "Start browsing to begin!")
	} else if boringCount > visitedCount/2 {
		lines = append(lines, "Explore new places!")
	} else {
		lines = append(lines, "Keep exploring!")
	}

	// Calculate rectangle dimensions relative to image size
	fontSize := math.Floor(float64(mapHeight)*0.025 + 0.5) // Round to nearest integer for sharper fonts
	if fontSize < 12 {
		fontSize = 12
	}
	padding := int(float64(mapWidth) * 0.015) // 1.5% of image width for more padding
	lineHeight := int(fontSize * 1.5)         // Increased line height from 1.2 to 1.5 for better spacing

	// Minimum width relative to image size
	rectWidth := int(float64(mapWidth) * 0.25) // Increased from 20% to 25% for more width
	// Add extra height for better text display - include padding and extra space for proper rendering
	rectHeight := padding*3 + len(lines)*lineHeight // Added extra padding multiplier

	// Adjust width based on text content for better sizing
	for _, line := range lines {
		estimatedWidth := len(line)*int(fontSize*0.65) + padding*2 // Slightly increased character width estimate
		if estimatedWidth > rectWidth {
			rectWidth = estimatedWidth
		}
	}

	// Position the rectangle - use configured position if available, otherwise use auto positioning
	var rectX, rectY int

	if a.config.StatsX >= 0 && a.config.StatsY >= 0 {
		// Use manually configured position
		rectX = a.config.StatsX
		rectY = a.config.StatsY
	} else {
		// Use automatic positioning (original behavior) at bottom-left corner with relative margins
		leftMargin := int(float64(mapWidth) * 0.15)    // 15% of image width
		bottomMargin := int(float64(mapHeight) * 0.15) // 15% of image height
		rectX = leftMargin
		rectY = mapHeight - rectHeight - bottomMargin
	}

	// CRITICAL: Ensure the rectangle stays within image bounds regardless of positioning mode
	if rectX+rectWidth > mapWidth {
		rectX = mapWidth - rectWidth
	}
	if rectX < 5 {
		rectX = 5
	}
	if rectY+rectHeight > mapHeight {
		rectY = mapHeight - rectHeight
	}
	if rectY < 5 {
		rectY = 5
	}

	// Use the simple game info rectangle function from resources package
	// The function will automatically calculate dimensions and use appropriate theme
	if err := resources.DrawGameInfoRectangle(img, a.fontManager, rectX, rectY, rectWidth, rectHeight, lines, a.getGameInfoConfig(a.config.Black, fontSize, padding)); err != nil {
		// Log error if font rendering fails - the map will still be generated without the status rectangle
		slog.Warn("Font rendering failed, status rectangle not displayed", "error", err)
	}
}

// getGameInfoConfig returns the configuration for the game info rectangle
func (a *App) getGameInfoConfig(darkTheme bool, fontSize float64, padding int) resources.GameInfoConfig {
	if darkTheme {
		return resources.GameInfoConfig{
			BackgroundColor: color.RGBA{20, 20, 20, 240},    // Darker background for better contrast
			TextColor:       color.RGBA{255, 255, 255, 255}, // Pure white text
			BorderColor:     color.RGBA{150, 150, 150, 255}, // Lighter border
			FontSize:        fontSize,
			Padding:         padding,
			BorderWidth:     2, // Thicker border for better visibility
		}
	} else {
		return resources.GameInfoConfig{
			BackgroundColor: color.RGBA{255, 255, 255, 240}, // Lighter background
			TextColor:       color.RGBA{0, 0, 0, 255},       // Pure black text for maximum contrast
			BorderColor:     color.RGBA{100, 100, 100, 255}, // Darker border
			FontSize:        fontSize,
			Padding:         padding,
			BorderWidth:     2, // Thicker border for better visibility
		}
	}
}

// drawCountryFills draws country fills based on hit counts
func (a *App) drawCountryFills(img *image.RGBA, mapWidth, mapHeight int) {
	// Get all connections to determine country locations
	connections := a.monitor.GetConnections()
	countryLocations := make(map[string][]image.Point)

	// Group connection points by country
	for _, conn := range connections {
		location, err := a.geoip.Lookup(conn.RemoteIP)
		if err != nil || location.Country == "" {
			continue
		}

		x, y := a.latLngToMapCoords(location.Latitude, location.Longitude, mapWidth, mapHeight)
		point := image.Point{X: int(x), Y: int(y)}
		countryLocations[location.Country] = append(countryLocations[location.Country], point)
	}

	// Draw fills for countries with hits
	for country, points := range countryLocations {
		fillColor := a.gameState.GetCountryColor(country)
		if fillColor.A == 0 {
			continue // Skip transparent (no hits)
		}

		// Create a region around the country's connection points
		a.fillCountryRegion(img, points, fillColor, mapWidth, mapHeight)
	}
}

// fillCountryRegion fills a region around the given points with the specified color
func (a *App) fillCountryRegion(img *image.RGBA, points []image.Point, fillColor color.RGBA, mapWidth, mapHeight int) {
	if len(points) == 0 {
		return
	}

	// Calculate bounding box of all points
	minX, maxX := points[0].X, points[0].X
	minY, maxY := points[0].Y, points[0].Y

	for _, p := range points {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	// Expand the region a bit
	radius := 30 // Adjust this to control fill area size
	minX = maxInt(0, minX-radius)
	maxX = minInt(mapWidth-1, maxX+radius)
	minY = maxInt(0, minY-radius)
	maxY = minInt(mapHeight-1, maxY+radius)

	// Fill the region using a simple flood fill approach
	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2

	a.floodFill(img, centerX, centerY, fillColor, mapWidth, mapHeight, 50)
}

// floodFill performs a bounded flood fill
func (a *App) floodFill(img *image.RGBA, startX, startY int, fillColor color.RGBA, mapWidth, mapHeight, maxRadius int) {
	// Simple circular fill instead of complex flood fill for performance
	for y := startY - maxRadius; y <= startY+maxRadius; y++ {
		for x := startX - maxRadius; x <= startX+maxRadius; x++ {
			if x >= 0 && x < mapWidth && y >= 0 && y < mapHeight {
				dx := x - startX
				dy := y - startY
				distance := math.Sqrt(float64(dx*dx + dy*dy))

				if distance <= float64(maxRadius) {
					// Blend with existing pixel
					existing := img.RGBAAt(x, y)
					blended := a.blendColors(existing, fillColor)
					img.Set(x, y, blended)
				}
			}
		}
	}
}

// blendColors blends two RGBA colors
func (a *App) blendColors(base, overlay color.RGBA) color.RGBA {
	if overlay.A == 0 {
		return base
	}
	if overlay.A == 255 {
		return overlay
	}

	alpha := float64(overlay.A) / 255.0
	invAlpha := 1.0 - alpha

	return color.RGBA{
		R: uint8(float64(base.R)*invAlpha + float64(overlay.R)*alpha),
		G: uint8(float64(base.G)*invAlpha + float64(overlay.G)*alpha),
		B: uint8(float64(base.B)*invAlpha + float64(overlay.B)*alpha),
		A: uint8(math.Max(float64(base.A), float64(overlay.A))),
	}
}

// Helper functions
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// logGameStats logs current game statistics
func (a *App) logGameStats() {
	a.gameState.mutex.RLock()
	total := 0
	occupied := 0
	totalHits := 0

	targetCountry := a.gameState.targetCountry
	targetSetAt := a.gameState.targetSetAt
	a.gameState.mutex.RUnlock()

	slog.Debug("=== GAME STATISTICS ===")

	// Log target country information
	if targetCountry != "" {
		timeRemaining := time.Duration(a.config.TargetInterval)*time.Minute - time.Since(targetSetAt)
		if timeRemaining > 0 {
			slog.Debug("Current target",
				"country", targetCountry,
				"minutes_remaining", timeRemaining.Minutes(),
			)
		} else {
			slog.Warn("Target change overdue",
				"country", targetCountry,
			)
		}
	} else {
		slog.Debug("No active target - all countries hit")
	}

	a.gameState.mutex.RLock()
	defer a.gameState.mutex.RUnlock()

	for country, state := range a.gameState.countries {
		total++
		totalHits += state.HitCount

		if state.Boring {
			occupied++
			slog.Debug("Country boring",
				"country", country,
				"hits", state.HitCount,
				"last_hit", state.LastHit.Format("15:04:05"),
			)
		} else {
			slog.Debug("Country interesting",
				"country", country,
				"hits", state.HitCount,
				"last_hit", state.LastHit.Format("15:04:05"),
			)
		}
	}

	if total > 0 {
		overvisitedRate := float64(occupied) / float64(total) * 100
		logging.LogGameStats(total, occupied, totalHits, overvisitedRate)
	} else {
		slog.Info("No countries visited yet - start browsing to begin your virtual travels!")
	}

	slog.Debug("=== END STATISTICS ===")
}

// ResetGame resets the game state
func (a *App) ResetGame() {
	a.gameState.mutex.Lock()
	defer a.gameState.mutex.Unlock()

	a.gameState.countries = make(map[string]*CountryGameState)
	slog.Info("Game state reset")
}

// FetchBoringCountries returns a list of boring countries
func (a *App) FetchBoringCountries() []string {
	a.gameState.mutex.RLock()
	defer a.gameState.mutex.RUnlock()

	var boring []string
	for country, state := range a.gameState.countries {
		if state.Boring {
			boring = append(boring, country)
		}
	}
	return boring
}

// SetTargetCountry sets a new target country
func (gs *GameState) SetTargetCountry(country string) {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	gs.targetCountry = country
	gs.targetSetAt = time.Now()
}

// GetTargetCountry returns the current target country
func (gs *GameState) GetTargetCountry() (string, time.Time) {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	return gs.targetCountry, gs.targetSetAt
}

// SelectRandomTargetCountry selects a random unhit country as the new target
func (a *App) SelectRandomTargetCountry() {
	if a.naturalEarth == nil {
		return
	}

	// Get list of all countries from Natural Earth data
	allCountries := make([]string, 0, len(a.naturalEarth.Countries))
	for _, country := range a.naturalEarth.Countries {
		allCountries = append(allCountries, country.Name)
	}

	// Filter out countries that have been hit
	a.gameState.mutex.RLock()
	unhitCountries := make([]string, 0)
	for _, countryName := range allCountries {
		if _, exists := a.gameState.countries[countryName]; !exists {
			unhitCountries = append(unhitCountries, countryName)
		}
	}
	a.gameState.mutex.RUnlock()

	// If no unhit countries remain, clear the target
	if len(unhitCountries) == 0 {
		a.gameState.SetTargetCountry("")
		slog.Info("No more unhit countries available for targeting")
		return
	}

	// Select a random unhit country
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	targetIndex := rng.Intn(len(unhitCountries))
	newTarget := unhitCountries[targetIndex]

	a.gameState.SetTargetCountry(newTarget)
	logging.LogTarget(newTarget, len(unhitCountries))
}

// logHit logs detailed information about a network hit
func (a *App) logHit(conn network.Connection, location *geoip.Location, mapWidth, mapHeight int) {
	// Get current country state
	countryState := a.gameState.GetCountryState(location.Country)
	currentHits := 0
	if countryState != nil {
		currentHits = countryState.HitCount
	}

	// Calculate map coordinates
	mapX, mapY := a.latLngToMapCoords(location.Latitude, location.Longitude, mapWidth, mapHeight)

	// Determine city name or use "Unknown" if not available
	cityName := location.City
	if cityName == "" {
		cityName = "Unknown"
	}

	// Add to recent hits
	hit := RecentHit{
		Country:  location.Country,
		City:     cityName,
		Protocol: conn.Protocol,
		Domain:   conn.RemoteIP, // Default to IP, will be updated by async reverse DNS
		Time:     time.Now(),
	}

	// Perform async reverse DNS lookup
	go func(ip string, hitIndex int) {
		names, err := net.LookupAddr(ip)
		if err == nil && len(names) > 0 {
			a.recentHitsMu.Lock()
			// Need to verify the index is still valid as slice might have shifted
			// Instead of index, we'll just update if we find the IP in recent hits
			for i := range a.recentHits {
				if a.recentHits[i].Domain == ip {
					// Clean up the trailing dot from reverse DNS
					name := strings.TrimSuffix(names[0], ".")
					a.recentHits[i].Domain = name
				}
			}
			a.recentHitsMu.Unlock()
		}
	}(conn.RemoteIP, 0)

	a.recentHitsMu.Lock()
	a.recentHits = append([]RecentHit{hit}, a.recentHits...)
	if len(a.recentHits) > 10 {
		a.recentHits = a.recentHits[:10]
	}
	a.recentHitsMu.Unlock()

	// Log visit with appropriate detail level based on log level
	logging.LogVisit(conn.Protocol, cityName, location.Country, conn.RemoteIP, conn.RemotePort, currentHits, currentHits+1)

	// Verbose logging with coordinates (debug level)
	logging.LogVisitVerbose(conn.Protocol, cityName, location.Country, conn.RemoteIP, conn.RemotePort,
		conn.LocalIP, conn.LocalPort, currentHits, currentHits+1,
		location.Latitude, location.Longitude, mapX, mapY, mapWidth, mapHeight)

	// Check if country becomes too boring from too many visits
	if currentHits+1 >= 10 {
		logging.LogOvervisited(location.Country)
	} else if currentHits+1 >= 7 {
		logging.LogCritical(location.Country, currentHits+1)
	}
}

// GetGameState returns a pointer to the game state for server access
func (a *App) GetGameState() *GameState {
	return a.gameState
}

// GetAchievementManager returns a pointer to the achievement manager for server access
func (a *App) GetAchievementManager() *achievements.AchievementManager {
	return a.achievements
}

// getBoringCountries returns a map of countries that are marked as boring
func (a *App) getBoringCountries() map[string]bool {
	a.gameState.mutex.RLock()
	defer a.gameState.mutex.RUnlock()

	boringCountries := make(map[string]bool)
	for countryName, state := range a.gameState.countries {
		if state.Boring {
			boringCountries[countryName] = true
		}
	}
	return boringCountries
}

// HandleFastestTravelerAchievement handles the logic for fastest traveler achievements
// This should be called when a user marks a country as boring
func (a *App) HandleFastestTravelerAchievement(countryName string) {
	// Check if this country was the target and mark it as boring
	wasTarget, _ := a.gameState.MarkCountryAsBoring(countryName)

	if wasTarget {
		// Unlock the fastest traveler achievement for this country
		achievementID := a.achievements.UnlockFastestTravelerAchievement(countryName)

		if achievementID != "" {
			slog.Info("🚀 Fastest Traveler Achievement earned!",
				"country", countryName,
				"achievement_id", achievementID,
			)
		}

		// Immediately select a new target country
		a.SelectRandomTargetCountry()

		newTarget, _ := a.gameState.GetTargetCountry()
		if newTarget != "" {
			slog.Info("🎯 New target selected after fastest traveler achievement",
				"new_target", newTarget,
				"previous_target", countryName,
			)
		}
	}
}

// Stop stops the application gracefully
func (a *App) Stop() {
	a.running = false
}

// Shutdown performs cleanup operations including wallpaper restoration
func (a *App) Shutdown() {
	slog.Info("🛑 Shutting down IP Travel Wallpaper...")

	// Stop the application
	a.Stop()

	// Close the local HTTP server listener
	if a.serverListener != nil {
		if err := a.serverListener.Close(); err != nil {
			slog.Warn("Failed to close local HTTP server listener", "error", err)
		}
	}

	// Restore original wallpaper if we backed it up
	if a.HasWallpaperBackup() {
		slog.Info("🔄 Restoring original wallpaper...")
		if err := a.RestoreOriginalWallpaper(); err != nil {
			slog.Error("Failed to restore original wallpaper during shutdown", "error", err)
		}
	} else {
		slog.Info("No wallpaper backup available - leaving current wallpaper as is")
	}

	slog.Info("👋 IP Travel Wallpaper shutdown complete")
}

// HasWallpaperBackup returns whether a wallpaper backup is available
func (a *App) HasWallpaperBackup() bool {
	return a.wallpaperBackedUp && a.originalWallpaper != ""
}

// RestoreOriginalWallpaper restores the original wallpaper if available
func (a *App) RestoreOriginalWallpaper() error {
	if !a.wallpaperBackedUp || a.originalWallpaper == "" {
		return fmt.Errorf("no wallpaper backup available")
	}

	if err := background.RestoreWallpaper(a.originalWallpaper); err != nil {
		return fmt.Errorf("failed to restore wallpaper: %w", err)
	}

	// Delete the backup file after successful restoration
	if err := os.Remove(a.originalWallpaper); err != nil {
		slog.Warn("Failed to delete wallpaper backup file after restore", "path", a.originalWallpaper, "error", err)
	} else {
		slog.Info("🗑️  Wallpaper backup file deleted after successful restore")
	}

	// Reset state
	a.originalWallpaper = ""
	a.wallpaperBackedUp = false

	slog.Info("✅ Original wallpaper restored successfully")
	return nil
}
