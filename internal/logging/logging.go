// Package logging provides centralized logging configuration using slog
package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// SetupLogger configures the global slog logger with the specified level
func SetupLogger(levelStr string) {
	var level slog.Level

	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo // Default to info
	}

	// Create a text handler with custom options
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Custom time format
			if a.Key == slog.TimeKey {
				return slog.String("time", a.Value.Time().Format("15:04:05"))
			}
			return a
		},
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// LogStartup logs application startup information
func LogStartup(appName, version string) {
	slog.Info("Application starting",
		"app", appName,
		"version", version,
	)
}

// LogConfig logs configuration information
func LogConfig(config interface{}) {
	slog.Debug("Configuration loaded", "config", config)
}

// LogVisit logs a network connection visit with detailed information
func LogVisit(protocol, city, country, remoteIP, remotePort string, currentVisits, newVisits int) {
	slog.Debug("Country visited",
		"protocol", protocol,
		"city", city,
		"country", country,
		"remote_ip", remoteIP,
		"remote_port", remotePort,
		"visits_before", currentVisits,
		"visits_after", newVisits,
	)
}

// LogVisitVerbose logs detailed visit information including coordinates
func LogVisitVerbose(protocol, city, country, remoteIP, remotePort, localIP, localPort string,
	currentVisits, newVisits int, lat, lng, mapX, mapY float64, mapWidth, mapHeight int) {
	slog.Debug("Detailed country visit",
		"protocol", protocol,
		"city", city,
		"country", country,
		"remote_ip", remoteIP,
		"remote_port", remotePort,
		"local_ip", localIP,
		"local_port", localPort,
		"visits_before", currentVisits,
		"visits_after", newVisits,
		"latitude", lat,
		"longitude", lng,
		"map_x", mapX,
		"map_y", mapY,
		"map_size", map[string]int{"width": mapWidth, "height": mapHeight},
	)
}

// LogOvervisited logs when a country becomes too boring from too many visits
func LogOvervisited(country string) {
	slog.Debug("Country visited too many times", "country", country, "status", "too_boring")
}

// LogCritical logs when a country is close to becoming too boring
func LogCritical(country string, visits int) {
	slog.Debug("Country close to becoming too boring",
		"country", country,
		"visits", visits,
		"threshold", 10,
	)
}

// LogTarget logs target country selection
func LogTarget(country string, remaining int) {
	slog.Debug("New target selected",
		"target_country", country,
		"unhit_remaining", remaining,
	)
}

// LogGameStats logs travel statistics
func LogGameStats(totalCountries, overvisitedCountries, totalVisits int, overvisitedRate float64) {
	slog.Debug("Travel statistics",
		"countries_visited", totalCountries,
		"countries_overvisited", overvisitedCountries,
		"total_visits", totalVisits,
		"overvisited_rate_percent", overvisitedRate,
	)
}

// LogScreen logs screen detection information
func LogScreen(width, height, displays int) {
	slog.Debug("Screen detected",
		"width", width,
		"height", height,
		"displays", displays,
	)
}

// LogMapSize logs optimal map size calculation
func LogMapSize(optimalWidth, optimalHeight, screenWidth, screenHeight int) {
	slog.Debug("Optimal map size calculated",
		"map_width", optimalWidth,
		"map_height", optimalHeight,
		"screen_width", screenWidth,
		"screen_height", screenHeight,
	)
}

// LogNaturalEarth logs Natural Earth data loading
func LogNaturalEarth(countryCount int) {
	slog.Info("Natural Earth data loaded", "countries", countryCount)
}

// LogMapRender logs map rendering information
func LogMapRender(width, height int, renderer string) {
	slog.Debug("Map rendered",
		"width", width,
		"height", height,
		"renderer", renderer,
	)
}

// LogWallpaper logs wallpaper operations
func LogWallpaper(action, path string) {
	slog.Info("Wallpaper operation",
		"action", action,
		"path", path,
	)
}

// LogError logs errors with context
func LogError(operation string, err error) {
	slog.Error("Operation failed",
		"operation", operation,
		"error", err.Error(),
	)
}

// LogWarning logs warnings with context
func LogWarning(message string, contextData map[string]interface{}) {
	attrs := make([]slog.Attr, 0, len(contextData))
	for k, v := range contextData {
		attrs = append(attrs, slog.Any(k, v))
	}
	slog.LogAttrs(context.Background(), slog.LevelWarn, message, attrs...)
}
