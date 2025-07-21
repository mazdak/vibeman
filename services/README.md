# Vibeman System Services

This directory contains service files for running vibeman as a system service on Linux (systemd) and macOS (launchd).

## Overview

The service files allow vibeman and vibeman-web to run automatically on system startup and be managed using standard system tools.

## Linux (systemd)

### Installation

```bash
# Install services (requires sudo)
sudo vibeman install-service

# Or specify custom options
sudo vibeman install-service --user myuser --web-dir /path/to/vibeman-web
```

### Management

```bash
# Check status
systemctl status vibeman@$USER vibeman-web@$USER

# Start/stop services
systemctl start vibeman@$USER
systemctl stop vibeman@$USER

# View logs
journalctl -u vibeman@$USER -f
journalctl -u vibeman-web@$USER -f
```

### Uninstallation

```bash
sudo vibeman uninstall-service
```

## macOS (launchd)

### Installation

```bash
# Install services (runs as current user)
vibeman install-service

# Or specify custom options
vibeman install-service --web-dir /path/to/vibeman-web
```

### Management

```bash
# Check status
launchctl list | grep vibeman

# Start/stop services
launchctl start com.vibeman.server
launchctl stop com.vibeman.server

# View logs
tail -f /usr/local/var/log/vibeman/server.log
tail -f /usr/local/var/log/vibeman/web.log
```

### Uninstallation

```bash
vibeman uninstall-service
```

## Service Configuration

### vibeman server
- Runs on port 8080 by default
- Logs to system journal (Linux) or /usr/local/var/log/vibeman/server.log (macOS)
- Automatically restarts on failure

### vibeman-web
- Runs on port 3000 by default
- Requires vibeman server to be running
- Logs to system journal (Linux) or /usr/local/var/log/vibeman/web.log (macOS)
- Automatically restarts on failure

## Manual Installation

If you prefer to install the services manually:

### Linux
1. Copy the service files to /etc/systemd/system/
2. Replace placeholder values (paths, usernames)
3. Run `systemctl daemon-reload`
4. Enable and start services

### macOS
1. Copy the plist files to ~/Library/LaunchAgents/
2. Replace placeholder values (paths, usernames)
3. Load with `launchctl load -w <plist-file>`

## Security Notes

- Services run with limited privileges
- Linux services use systemd security hardening features
- Log files may contain sensitive information - ensure proper permissions