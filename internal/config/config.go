package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config represents the application configuration
type Config struct {
	MapWidth         int    `config:"map_width"`
	AutoDetectScreen bool   `config:"auto_detect_screen"`
	Black            bool   `config:"black"`
	UpdateInterval   int    `config:"update_interval"`
	TargetInterval   int    `config:"target_interval"` // Minutes between target changes
	LogLevel         string `config:"log_level"`       // debug, info, warn, error
	StatsX           int    `config:"stats_x"`         // X position of stats rectangle (-1 for auto)
	StatsY           int    `config:"stats_y"`         // Y position of stats rectangle (-1 for auto)
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		MapWidth:         1000,
		AutoDetectScreen: true, // Default to auto-detection
		Black:            false,
		UpdateInterval:   1,
		TargetInterval:   5,      // New target every 5 minutes
		LogLevel:         "info", // Default log level
		StatsX:           -1,     // -1 means auto-position (default behavior)
		StatsY:           -1,     // -1 means auto-position (default behavior)
	}
}

// LoadConfig loads configuration from file or creates default
func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "iptw")
	configPath := filepath.Join(configDir, "iptwrc")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// If config file doesn't exist, create default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := cfg.Save(configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return cfg, nil
	}

	// Read existing config file
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	cfg := DefaultConfig()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {

		case "map_width":
			if val, err := strconv.Atoi(value); err == nil {
				cfg.MapWidth = val
			}
		case "auto_detect_screen":
			cfg.AutoDetectScreen = value == "true"
		case "black":
			cfg.Black = value == "true"
		case "update_interval":
			if val, err := strconv.Atoi(value); err == nil {
				cfg.UpdateInterval = val
			}
		case "target_interval":
			if val, err := strconv.Atoi(value); err == nil {
				cfg.TargetInterval = val
			}
		case "log_level":
			// Validate log level
			switch value {
			case "debug", "info", "warn", "error":
				cfg.LogLevel = value
			default:
				cfg.LogLevel = "info" // Default to info for invalid values
			}
		case "stats_x":
			if val, err := strconv.Atoi(value); err == nil {
				cfg.StatsX = val
			}
		case "stats_y":
			if val, err := strconv.Atoi(value); err == nil {
				cfg.StatsY = val
			}
		}
	}

	return cfg, scanner.Err()
}

// Save saves the configuration to file
func (c *Config) Save(configPath string) error {
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, `
map_width %d
auto_detect_screen %t
black %t
update_interval %d
target_interval %d
log_level %s
stats_x %d
stats_y %d
`, c.MapWidth, c.AutoDetectScreen, c.Black, c.UpdateInterval, c.TargetInterval, c.LogLevel, c.StatsX, c.StatsY)

	return err
}
