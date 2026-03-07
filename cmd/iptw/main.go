package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"iptw/internal/config"
	"iptw/internal/geoip"
	"iptw/internal/gui"
	"iptw/internal/logging"
	"iptw/internal/network"
	"iptw/internal/singleton"
)

// Version information set during build
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	var configPath string
	var forceStart bool
	var showVersion bool
	flag.StringVar(&configPath, "config", "", "Path to config file (default: ~/.config/iptw/iptwrc)")
	flag.BoolVar(&forceStart, "force", false, "Force start even if another instance appears to be running")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()

	// Handle version request
	if showVersion {
		fmt.Printf("IPTW (IP Travel Wallpaper) %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		return
	}

	// Create singleton lock to ensure only one instance runs
	lock, err := singleton.NewLock("iptw")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create singleton lock: %v\n", err)
		os.Exit(1)
	}

	// Attempt to acquire the lock (unless force flag is used)
	if !forceStart {
		if err := lock.Acquire(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Please check if another instance is already running and stop it before starting a new one.\n")
			fmt.Fprintf(os.Stderr, "If you're sure no other instance is running, you may need to manually remove the lock file.\n")
			fmt.Fprintf(os.Stderr, "Alternatively, use the --force flag to bypass this check.\n")
			os.Exit(1)
		}

		// Ensure lock is released when application exits
		defer func() {
			if releaseErr := lock.Release(); releaseErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to release singleton lock: %v\n", releaseErr)
			}
		}()
	} else {
		fmt.Println("Warning: Force start enabled - skipping singleton check")
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup logging based on config
	logging.SetupLogger(cfg.LogLevel)

	// Initialize GeoIP database (using embedded database)
	geoipDB, err := geoip.NewDatabase("")
	if err != nil {
		slog.Error("Failed to initialize embedded GeoIP database", "error", err)
		os.Exit(1)
	}
	defer func() { _ = geoipDB.Close() }()

	// Initialize network monitor
	netMon := network.NewMonitor()

	// Create GUI application
	app, err := gui.NewApp(cfg, geoipDB, netMon)
	if err != nil {
		slog.Error("Failed to create application", "error", err)
		os.Exit(1)
	}

	// Ensure clean shutdown when the application exits
	defer app.Shutdown()

	fmt.Println("Starting IP Travel Wallpaper (iptw)...")
	if err := app.Run(); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}
