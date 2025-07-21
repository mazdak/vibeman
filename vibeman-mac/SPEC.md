# Vibeman Mac App Specification

## Overview

The Vibeman Mac app is a native macOS menu bar application that provides a seamless desktop experience for the Vibeman development environment management system. It wraps the Go server and React web UI in a convenient, always-accessible interface.

## Architecture

```
┌─────────────────────────────────────────┐
│            Menu Bar App                 │
│         (Swift/SwiftUI)                 │
├─────────────────────────────────────────┤
│                                         │
│  ┌───────────────┐  ┌─────────────┐   │
│  │  Go Server    │  │   Bun Web   │   │
│  │  (embedded)   │  │   Server    │   │
│  └───────┬───────┘  └──────┬──────┘   │
│          │                  │          │
│          └──────────┬───────┘          │
│                     ▼                   │
│           ┌─────────────────┐          │
│           │    WebView      │          │
│           │  (WKWebView)    │          │
│           └─────────────────┘          │
└─────────────────────────────────────────┘
```

## Features

### Core Functionality
- **Menu Bar Integration**: Lives in the macOS menu bar for quick access
- **Server Management**: Automatically starts and manages the Vibeman Go server
- **Web UI Hosting**: Runs the Bun development server for the React UI
- **WebView Display**: Native window with embedded web browser for the UI
- **Auto-Launch**: Option to start at login via macOS Login Items

### User Interface
- **Menu Bar Icon**: Custom icon that adapts to light/dark mode
- **Status Menu**: Quick access to:
  - Open main window
  - Install CLI tool
  - Check for updates
  - Start at login toggle
  - Quit application
- **Main Window**: Full-featured web UI in a native window
  - Resizable with minimum size constraints
  - Native window controls
  - WebView navigation

### System Integration
- **CLI Installation**: One-click installation of the `vibeman` CLI tool to `/usr/local/bin`
- **Auto-Updates**: Sparkle framework integration for automatic updates
- **Configuration**: Reads from XDG config directory (`~/.config/vibeman/`)
- **Process Management**: Graceful startup and shutdown of server processes

## Technical Details

### Dependencies
- **Language**: Swift 5.9+
- **Platform**: macOS 12.0+
- **Frameworks**:
  - SwiftUI for UI
  - WebKit for WebView
  - Sparkle for auto-updates
  - ServiceManagement for login items
  - VibemanKit (internal framework)

### Components

#### VibemanApp.swift
- Main application entry point
- Menu bar app setup (no dock icon)
- Window management
- Server process lifecycle

#### VibemanKit Framework
- Process management for Go server
- Health monitoring
- Auto-restart capabilities
- Thread-safe operations

#### Key Features
- **Server Discovery**: Locates vibeman executable in:
  - App bundle resources
  - Development paths
  - System PATH
- **Web App Discovery**: Finds vibeman-web directory in:
  - App bundle resources
  - Development paths
  - Current directory

### Configuration

The app reads configuration from `~/.config/vibeman/config.toml`:
```toml
[server]
basePort = 8080      # Go server port
webUIPort = 8081     # Web UI port
```

### Build Process

1. **Development Build**:
   ```bash
   cd vibeman-mac
   ./build-dev.sh
   ```

2. **Release Build**:
   ```bash
   ./build.sh
   ```

3. **Testing**:
   ```bash
   ./build-and-test.sh
   ```

### Distribution

- **Format**: .app bundle or .dmg
- **Signing**: Requires Apple Developer certificate
- **Notarization**: Required for distribution outside Mac App Store
- **Updates**: GitHub-hosted appcast.xml for Sparkle

## User Experience

### First Launch
1. App appears in menu bar
2. Automatically starts Go server and web UI
3. Shows success/error notifications
4. Ready for use

### Typical Usage
1. Click menu bar icon
2. Select "Open Vibeman" to show main window
3. Use web UI for all Vibeman functionality
4. Window can be closed without quitting app
5. App continues running in background

### CLI Integration
- Menu option to install CLI tool
- Requires admin password (sudo)
- Creates symlink to embedded vibeman binary
- Enables command-line usage alongside GUI

## Security Considerations

- **Entitlements**: 
  - Network client/server for API communication
  - File access for repository management
- **Sandbox**: Currently disabled for file system access
- **Code Signing**: Required for distribution
- **Notarization**: Required for macOS Gatekeeper

## Future Enhancements

1. **Native UI Elements**: 
   - Quick repository switcher in menu
   - Native notifications for long-running operations
   - Touch Bar support (if applicable)

2. **Performance**:
   - Lazy loading of web UI
   - Background server health monitoring
   - Resource usage optimization

3. **Integration**:
   - System-wide hotkeys
   - Finder extension for repository folders
   - Quick Look preview for worktree status