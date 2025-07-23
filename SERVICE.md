# Service Management Guide

IP Travel Wallpaper (iptw) includes cross-### Windows (System Service)
- **Service Name**: `iptw`
- **Display Name**: `IP Travel Wallpaper Server`
- **Command**: `iptw -force -server -port 32782`
- **Auto-start**: Starts automatically on system boot
- **Management**: Uses Windows Service Control Manager (`sc` commands)
- **Permissions**: May require administrator privileges for installation
- **User Context**: Runs as the installing user (required for desktop wallpaper access)
- **Desktop Access**: Service requires user to be logged in for wallpaper changesm background service management capabilities that allow the application to run automatically as a system service with HTTP statistics server functionality on macOS, Linux, and Windows.

## Service Features

When installed as a service, iptw runs with the following configuration:
- **Server Mode**: Enabled with HTTP statistics server
- **Port**: 32782 (default, configurable)
- **Force Start**: Bypasses singleton checks for service operation
- **Background Operation**: Runs without UI interaction

## Service Commands

### Installation
```bash
iptw -install-service
```
Installs iptw as a background service that starts automatically:
- **macOS**: Creates a LaunchAgent in `~/Library/LaunchAgents/`
- **Linux**: Creates a systemd user service in `~/.config/systemd/user/`
- **Windows**: Creates a Windows service using the Service Control Manager

The service runs with server functionality enabled on port 32782, providing HTTP access to statistics and achievements.

### Service Status
```bash
iptw -service-status
```
Check if the service is currently running.

### Start Service
```bash
iptw -start-service
```
Manually start the installed service.

### Stop Service
```bash
iptw -stop-service
```
Manually stop the running service.

### Uninstallation
```bash
iptw -uninstall-service
```
Completely remove the service from the system.

## Platform-Specific Details

### macOS (LaunchAgent)
- **Service File**: `~/Library/LaunchAgents/com.iptw.plist`
- **Command**: `iptw -force -server -port 32782`
- **Log Files**: 
  - Output: `~/Library/Logs/iptw.out.log`
  - Errors: `~/Library/Logs/iptw.err.log`
- **Auto-start**: Starts automatically on user login
- **Management**: Uses `launchctl` commands

### Linux (systemd user service)
- **Service File**: `~/.config/systemd/user/iptw.service`
- **Command**: `iptw -force -server -port 32782`
- **Auto-start**: Starts automatically on user login
- **Management**: Uses `systemctl --user` commands
- **Enable Lingering**: For service to start without login:
  ```bash
  sudo loginctl enable-linger $USER
  ```

### Windows (System Service)
- **Service Name**: `iptw`
- **Display Name**: `IP Travel Wallpaper Server`
- **Command**: `iptw -force -server -port 32782`
- **Auto-start**: Starts automatically on system boot
- **Management**: Uses Windows Service Control Manager (`sc` commands)
- **Permissions**: May require administrator privileges

## Service Behavior

When running as a service, iptw:
- Runs with the `-force` flag to bypass singleton checks
- Runs with the `-server` flag to enable HTTP statistics server
- Uses port `32782` for the HTTP server (configurable)
- Continuously monitors network connections
- Updates the desktop wallpaper based on discovered countries
- Provides HTTP statistics server on port 32782
- Automatically restarts on failure (platform-dependent)
- Logs activity to platform-specific locations

## HTTP Statistics Server

When running as a service, iptw provides an HTTP server on port 32782 with the following endpoints:

- **Health Check**: `GET /health` - Service health status
- **Statistics**: `GET /stats` - Current network statistics and wallpaper info
- **Achievements**: `GET /achievements` - Unlocked achievements
- **Countries**: `GET /countries` - Discovered countries with details

You can access these endpoints via:
```bash
# Check if service is healthy
curl http://localhost:32782/health

# Get current statistics
curl http://localhost:32782/stats

# View achievements
curl http://localhost:32782/achievements

# List countries
curl http://localhost:32782/countries
```

Or use the client mode:
```bash
# View statistics
iptw -client

# Watch live updates
iptw -client -watch

# View achievements
iptw -client -achievements

# View countries
iptw -client -countries
```

## Troubleshooting

### Service Not Starting
1. Check service status: `iptw -service-status`
2. View logs (platform-specific locations above)
3. Ensure executable has proper permissions
4. Try manual start: `iptw -start-service`
5. **Windows**: Ensure user is logged in (required for desktop wallpaper access)

### Permission Issues
- **macOS/Linux**: Service runs under current user account
- **Windows**: May require administrator privileges for installation; service runs as installing user and needs user to be logged in for desktop access

### Service Conflicts
- Only one instance of iptw can run at a time
- Service installation automatically handles singleton management
- Manual runs should use `-force` if service is installed

## Best Practices

1. **Install the service** for automatic startup and continuous operation
2. **Check logs** regularly for any issues or interesting discoveries
3. **Use service management commands** rather than manual process management
4. **Uninstall before major system changes** to avoid orphaned services

## Example Workflow

```bash
# Install and start the service
iptw -install-service

# Check that it's running
iptw -service-status

# View current wallpaper and let it run automatically
# Service will continue updating wallpaper based on network activity

# When needed, stop temporarily
iptw -stop-service

# Restart when ready
iptw -start-service

# Completely remove when no longer needed
iptw -uninstall-service
```
