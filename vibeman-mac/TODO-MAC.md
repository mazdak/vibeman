# Vibeman Mac App TODO List

## Overview
The Mac app is a lightweight Swift-based menu bar application that manages the Vibeman server and web UI. It provides a native macOS experience with features like auto-updates, login items, and WebKit integration.

## High Priority Tasks 🔴

### 1. Update Server Command Integration ✅ COMPLETED
- [x] Update `startGoServer()` to use new `vibeman server start` command instead of `vibeman server`
- [x] Add support for `vibeman server stop` command in cleanup
- [x] Update server arguments to match new command structure (`--port` vs `--port=`)
- [x] Add proper daemon mode support (--daemon flag)

### 2. Fix Binary and Resource Paths ✅ COMPLETED
- [x] Update `getVibemanExecutablePath()` to look for correct binary name (`vibeman` not `vibeman-server`)
- [x] Fix development path resolution (currently looking in wrong relative paths)
- [x] Update bundle resource paths to match actual app structure
- [x] Add proper error handling when binary is not found

### 3. Update ProcessManager for New Architecture ✅ COMPLETED
- [x] Update `ProcessManager.swift` to work with new server commands
- [x] Fix binary name from `vibeman-server` to `vibeman`
- [x] Update health check endpoint to match actual server (`/health` vs `/api/health`)
- [x] Add support for new server status endpoint

### 4. Fix Web Server Integration ✅ COMPLETED
- [x] Update Bun web server startup to use correct path resolution
- [x] Add proper environment variable setup for web UI port
- [x] Ensure web UI can communicate with Go server on correct ports
- [x] Add error handling for missing Bun installation

## Medium Priority Tasks 🟡

### 5. Update Configuration Management ✅ COMPLETED
- [x] Update `VibemanConfig.swift` to match current TOML structure
- [x] Add support for new configuration fields (services config path)
- [x] Fix TOML parsing to use proper library instead of regex (using TOMLKit)
- [x] Add support for XDG config directory standards

### 6. Improve Error Handling and User Feedback ✅ PARTIALLY COMPLETED
- [x] Add user-friendly error messages when server fails to start
- [x] Show server status in menu bar (running/stopped/error) - via tooltips
- [ ] Improve connection retry logic with exponential backoff

### 7. Update Build Scripts
- [ ] Update `build.sh` to build correct Go binary
- [ ] Fix resource bundling to include web UI files
- [ ] Update code signing and notarization scripts
- [ ] Add proper version injection from Git tags

### 8. Fix Auto-Update System
- [ ] Update Sparkle integration for macOS 14+ compatibility
- [ ] Create proper appcast.xml for GitHub releases
- [ ] Add update channel selection (stable/beta)
- [ ] Fix feed URL to point to correct repository

## Low Priority Tasks 🟢

### 9. UI/UX Improvements
- [ ] Update menu bar icon for better visibility on all macOS versions
- [ ] Add dark mode support for error views
- [ ] Improve WebView error pages with better styling
- [ ] Add keyboard shortcuts for common actions

### 10. Developer Experience
- [ ] Add unit tests for critical components
- [ ] Improve documentation and code comments

### 11. Performance Optimizations
- [ ] Lazy load WebView until first window open
- [ ] Reduce memory footprint when running in background
- [ ] Optimize server health check frequency
- [ ] Add proper cleanup on app termination

### 12. Additional Features
- [ ] Create preference pane for server configuration

## Technical Debt 🔧

### 13. Code Quality
- [ ] Replace manual TOML parsing with proper Swift library
- [ ] Remove deprecated APIs (NSUserNotification)
- [ ] Update to latest Swift concurrency patterns
- [ ] Fix all compiler warnings

### 14. Testing
- [ ] Add unit tests for ProcessManager
- [ ] Add integration tests for server lifecycle
- [ ] Create UI tests for critical user flows
- [ ] Add CI/CD pipeline for automated testing

### 15. Documentation
- [ ] Update README with current build instructions
- [ ] Document configuration options
- [ ] Create user guide for Mac app features
- [ ] Add troubleshooting guide

## New Features Added 🎉

1. **JavaScript Bridge**: Created native bridge for web UI integration
   - Added `native-bridge.ts` TypeScript module for web UI
   - Supports opening files in Finder
   - Native macOS notifications
   - Permission handling
   
2. **Health Monitoring**: Server health checks with status indicators
   - Menu bar tooltip shows server status
   - Automatic health check after server startup
   
3. **Comprehensive Tests**: Added test coverage for key functionality
   - ProcessManagerTests with error handling tests
   - VibemanConfigTests for TOML parsing
   - WebViewControllerTests for JavaScript bridge
   
4. **Web UI Integration**: The web UI now knows about the native bridge
   - Created example component showing bridge usage
   - Type-safe TypeScript interface for bridge methods

## Implementation Notes

### Current Architecture
- Menu bar app (no dock icon)
- Manages two processes: Go server and Bun web server
- WebKit-based UI with JavaScript bridge
- Sparkle for auto-updates
- Login items support

### Key Components
1. **VibemanApp.swift**: Main app delegate and UI management
2. **ProcessManager.swift**: Server process lifecycle (currently unused in main app)
3. **WebViewController.swift**: WebKit integration with JS bridge
4. **VibemanConfig.swift**: Configuration management
5. **CLIInstaller.swift**: CLI tool installation
6. **LoginItemsManager.swift**: Start at login functionality

### Build Requirements
- macOS 12.0+
- Swift 5.9+
- Xcode 14.0+
- Vibeman Go binary
- Bun for web UI

### Distribution
- Code signing required
- Notarization for macOS 10.15+
- DMG or ZIP distribution
- Sparkle appcast for updates

## Next Steps

1. Fix critical path issues (binary names, command structure)
2. Update configuration to match current server
3. Improve error handling and user feedback
4. Add comprehensive testing
5. Prepare for distribution (signing, notarization)
