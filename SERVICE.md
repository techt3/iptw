# Service Management Guide

IP Travel Wallpaper (iptw) includes cross-platform background service management capabilities that allow the application to run automatically as a system service on macOS, Linux, and Windows.

## Service Commands

### Installation
```bash
iptw -install-service
```
Installs iptw as a background service that starts automatically:
- **macOS**: Creates a LaunchAgent in `~/Library/LaunchAgents/`
- **Linux**: Creates a systemd user service in `~/.config/systemd/user/`
- **Windows**: Creates a Windows service using the Service Control Manager

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
- **Log Files**: 
  - Output: `~/Library/Logs/iptw.out.log`
  - Errors: `~/Library/Logs/iptw.err.log`
- **Auto-start**: Starts automatically on user login
- **Management**: Uses `launchctl` commands

### Linux (systemd user service)
- **Service File**: `~/.config/systemd/user/iptw.service`
- **Auto-start**: Starts automatically on user login
- **Management**: Uses `systemctl --user` commands
- **Enable Lingering**: For service to start without login:
  ```bash
  sudo loginctl enable-linger $USER
  ```

### Windows (System Service)
- **Service Name**: `iptw`
- **Display Name**: `IP Travel Wallpaper`
- **Auto-start**: Starts automatically on system boot
- **Management**: Uses Windows Service Control Manager (`sc` commands)
- **Permissions**: May require administrator privileges

## Service Behavior

When running as a service, iptw:
- Runs with the `-force` flag to bypass singleton checks
- Continuously monitors network connections
- Updates the desktop wallpaper based on discovered countries
- Automatically restarts on failure (platform-dependent)
- Logs activity to platform-specific locations

## Troubleshooting

### Service Not Starting
1. Check service status: `iptw -service-status`
2. View logs (platform-specific locations above)
3. Ensure executable has proper permissions
4. Try manual start: `iptw -start-service`

### Permission Issues
- **macOS/Linux**: Service runs under current user account
- **Windows**: May require administrator privileges for installation

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
