package main

import (
	"flag"
	"fmt"
	"log/slog"

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

	// On Windows GUI builds, set up file logging immediately so that any
	// startup failure is recorded even before the config is read.
	if closer := logging.SetupWindowsFileLogger(slog.LevelDebug); closer != nil {
		defer closer.Close()
	}
	slog.Info("IPTW starting", "version", Version)

	// Create singleton lock to ensure only one instance runs
	lock, err := singleton.NewLock("iptw")
	if err != nil {
		fatalError("Startup Error", fmt.Sprintf("Failed to create singleton lock: %v", err))
		return
	}

	// Attempt to acquire the lock (unless force flag is used)
	if !forceStart {
		if err := lock.Acquire(); err != nil {
			fatalError("Already Running",
				"Another instance of IPTW appears to be running.\n\n"+
					err.Error()+"\n\n"+
					"If you are sure no other instance is running, delete the lock file or use --force.")
			return
		}
		// Use defer so the lock is always released even when run() returns an
		// error — os.Exit() is no longer called after this point.
		defer func() {
			if releaseErr := lock.Release(); releaseErr != nil {
				slog.Warn("Failed to release singleton lock", "error", releaseErr)
			}
		}()
	} else {
		slog.Warn("Force start enabled – skipping singleton check")
	}

	// Recover from any unexpected panics so they produce a visible error
	// rather than a silent crash on Windows GUI builds.
	defer func() {
		if r := recover(); r != nil {
			fatalError("Unexpected Crash", fmt.Sprintf("panic: %v", r))
		}
	}()

	if err := run(configPath); err != nil {
		fatalError("Application Error", err.Error())
	}
}

// run contains the main application logic. Returning an error (instead of
// calling os.Exit) ensures that all deferred cleanup in main() executes,
// most importantly releasing the singleton lock file.
func run(configPath string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	_ = configPath // flag reserved for future per-path overrides

	// Re-configure logging now that we know the desired level from config.
	logging.SetupLogger(cfg.LogLevel)

	// Initialize GeoIP database (embedded in binary)
	geoipDB, err := geoip.NewDatabase("")
	if err != nil {
		return fmt.Errorf("failed to initialise embedded GeoIP database: %w", err)
	}
	defer func() { _ = geoipDB.Close() }()

	// Initialize network monitor
	netMon := network.NewMonitor()

	// Create GUI application
	app, err := gui.NewApp(cfg, geoipDB, netMon)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}
	defer app.Shutdown()

	slog.Info("Starting IP Travel Wallpaper (iptw)")
	return app.Run()
}
