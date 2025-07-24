package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"iptw/internal/client"
	"iptw/internal/config"
	"iptw/internal/geoip"
	"iptw/internal/gui"
	"iptw/internal/logging"
	"iptw/internal/network"
	"iptw/internal/server"
	"iptw/internal/service"
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
	var installService bool
	var uninstallService bool
	var startService bool
	var stopService bool
	var statusService bool
	var showVersion bool

	// Server/Client mode flags
	var serverMode bool
	var serverPort string
	var clientMode bool
	var clientServer string
	var clientWatch bool
	var clientInterval int
	var clientAchievements bool
	var clientCountries bool
	var clientShutdown bool

	flag.StringVar(&configPath, "config", "", "Path to config file (default: ~/.config/iptw/iptwrc)")
	flag.BoolVar(&forceStart, "force", false, "Force start even if another instance appears to be running")
	flag.BoolVar(&installService, "install-service", false, "Install as background service (macOS/Linux/Windows)")
	flag.BoolVar(&uninstallService, "uninstall-service", false, "Uninstall background service")
	flag.BoolVar(&startService, "start-service", false, "Start the background service")
	flag.BoolVar(&stopService, "stop-service", false, "Stop the background service")
	flag.BoolVar(&statusService, "service-status", false, "Check service status")
	flag.BoolVar(&showVersion, "version", false, "Show version information")

	// Server/Client mode flags
	flag.BoolVar(&serverMode, "server", true, "Run in server mode (with HTTP statistics server)")
	flag.StringVar(&serverPort, "port", "32782", "Server port for statistics HTTP server")
	flag.BoolVar(&clientMode, "client", false, "Run in client mode (fetch stats from remote server)")
	flag.StringVar(&clientServer, "server-url", "http://127.0.0.1:32782", "Server URL for client mode")
	flag.BoolVar(&clientWatch, "watch", false, "Watch mode: continuously poll and display stats")
	flag.IntVar(&clientInterval, "interval", 30, "Poll interval in seconds for watch mode")
	flag.BoolVar(&clientAchievements, "achievements", false, "Show achievements in client mode")
	flag.BoolVar(&clientCountries, "countries", false, "Show country details in client mode")
	flag.BoolVar(&clientShutdown, "shutdown", false, "Shutdown the remote server in client mode")
	flag.Parse()

	// Handle version request
	if showVersion {
		fmt.Printf("IPTW (IP Travel Wallpaper) %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		return
	}

	// Handle service management commands
	if installService || uninstallService || startService || stopService || statusService {
		sm, err := service.NewServiceManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create service manager: %v\n", err)
			os.Exit(1)
		}

		switch {
		case installService:
			if err := sm.Install(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to install service: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Service installation completed successfully!")
			fmt.Println("Use 'iptw -start-service' to start the service.")
			return

		case uninstallService:
			if err := sm.Uninstall(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to uninstall service: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Service uninstallation completed successfully!")
			return

		case startService:
			if err := sm.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to start service: %v\n", err)
				os.Exit(1)
			}
			return

		case stopService:
			if err := sm.Stop(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to stop service: %v\n", err)
				os.Exit(1)
			}
			return

		case statusService:
			isRunning, err := sm.Status()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to check service status: %v\n", err)
				os.Exit(1)
			}
			if isRunning {
				fmt.Println("✅ Service is running")
			} else {
				fmt.Println("❌ Service is not running")
			}
			return
		}
	}

	// Handle client mode
	if clientMode {
		c := client.NewClient(clientServer)

		// First check if server is healthy
		if err := c.CheckHealth(); err != nil {
			fmt.Fprintf(os.Stderr, "Server health check failed: %v\n", err)
			os.Exit(1)
		}

		switch {
		case clientWatch:
			interval := time.Duration(clientInterval) * time.Second
			if err := c.WatchStats(interval); err != nil {
				fmt.Fprintf(os.Stderr, "Watch mode failed: %v\n", err)
				os.Exit(1)
			}
			return

		case clientAchievements:
			if err := c.PrintAchievements(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to fetch achievements: %v\n", err)
				os.Exit(1)
			}
			return

		case clientCountries:
			if err := c.PrintCountries(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to fetch countries: %v\n", err)
				os.Exit(1)
			}
			return

		case clientShutdown:
			if err := c.Shutdown(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to shutdown server: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Server shutdown request sent successfully to %s\n", clientServer)
			return

		default:
			// Default: show statistics
			if err := c.PrintStats(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to fetch stats: %v\n", err)
				os.Exit(1)
			}
			return
		}
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
	defer geoipDB.Close()

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

	// Start in server mode if requested
	if serverMode {
		fmt.Printf("Starting IP Travel Wallpaper (iptw) with statistics server on port %s...\n", serverPort)

		// Start the statistics server in a goroutine
		srv := server.NewServer(app, cfg, serverPort)
		go func() {
			if err := srv.Start(); err != nil {
				slog.Error("Statistics server error", "error", err)
			}
		}()

		// Run the main application
		if err := app.Run(); err != nil {
			slog.Error("Application error", "error", err)
			os.Exit(1)
		}
	} else {
		// Default mode: run the GUI application without server
		fmt.Println("Starting IP Travel Wallpaper (iptw)...")
		if err := app.Run(); err != nil {
			slog.Error("Application error", "error", err)
			os.Exit(1)
		}
	}
}
