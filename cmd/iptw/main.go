package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"

	"iptw/internal/config"
	"iptw/internal/geoip"
	"iptw/internal/gui"
	"iptw/internal/ispdetect"
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
	var forceStart bool
	var showVersion bool
	var foreground bool
	var pprofAddr string
	flag.BoolVar(&forceStart, "force", false, "Force start even if another instance appears to be running")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&foreground, "foreground", false, "Run in the foreground (keep terminal attached)")
	flag.StringVar(&pprofAddr, "pprof", "", "Enable pprof profiling server on the given address (e.g. 127.0.0.1:6060)")
	flag.Parse()

	// Handle version request
	if showVersion {
		fmt.Printf("IPTW (IP Travel Wallpaper) %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		return
	}

	// On macOS/Linux: detach from the terminal so the user can close the
	// launching shell.  The process re-execs itself with --foreground and
	// the parent exits immediately.  This is a no-op on Windows.
	maybeDaemonize(foreground)

	// Start pprof server if requested.
	if pprofAddr != "" {
		go func() {
			slog.Info("pprof profiling server starting", "addr", pprofAddr,
				"hint", "go tool pprof http://"+pprofAddr+"/debug/pprof/profile")
			if err := http.ListenAndServe(pprofAddr, nil); err != nil { //nolint:gosec
				slog.Error("pprof server stopped", "error", err)
			}
		}()
	}

	// On Windows GUI builds, set up file logging immediately so that any
	// startup failure is recorded even before the config is read.
	if closer := logging.SetupWindowsFileLogger(slog.LevelDebug); closer != nil {
		defer func() { _ = closer.Close() }()
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

	if err := run(); err != nil {
		fatalError("Application Error", err.Error())
	}
}

// run contains the main application logic. Returning an error (instead of
// calling os.Exit) ensures that all deferred cleanup in main() executes,
// most importantly releasing the singleton lock file.
func run() error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

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

	// If skip_isp is enabled, detect the user's ISP CIDR ranges at startup.
	// This runs synchronously so the filter is active before the first tick;
	// failures are non-fatal — a warning is logged and the feature is skipped.
	if cfg.SkipISP {
		cidrs, err := ispdetect.DetectISPCIDRsAuto()
		if err != nil {
			slog.Warn("skip_isp: ISP detection failed, feature disabled", "error", err)
		} else {
			netMon.SetISPCIDRs(cidrs)
		}
	}

	// Create GUI application
	app, err := gui.NewApp(cfg, geoipDB, netMon)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}
	defer app.Shutdown()

	slog.Info("Starting IP Travel Wallpaper (iptw)")
	return app.Run()
}
