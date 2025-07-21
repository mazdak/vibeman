// Package xdg provides XDG Base Directory Specification compliant paths
package xdg

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigDir returns the XDG config directory for vibeman
// Priority: XDG_CONFIG_HOME > ~/.config/vibeman
func ConfigDir() (string, error) {
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "vibeman"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "vibeman"), nil
}

// DataDir returns the XDG data directory for vibeman
// Priority: XDG_DATA_HOME > ~/.local/share/vibeman
func DataDir() (string, error) {
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "vibeman"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".local", "share", "vibeman"), nil
}

// CacheDir returns the XDG cache directory for vibeman
// Priority: XDG_CACHE_HOME > ~/.cache/vibeman
func CacheDir() (string, error) {
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		return filepath.Join(xdgCache, "vibeman"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".cache", "vibeman"), nil
}

// StateDir returns the XDG state directory for vibeman
// Priority: XDG_STATE_HOME > ~/.local/state/vibeman
func StateDir() (string, error) {
	if xdgState := os.Getenv("XDG_STATE_HOME"); xdgState != "" {
		return filepath.Join(xdgState, "vibeman"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".local", "state", "vibeman"), nil
}

// RuntimeDir returns the XDG runtime directory for vibeman
// Priority: XDG_RUNTIME_DIR > /tmp/vibeman-$UID
func RuntimeDir() (string, error) {
	if xdgRuntime := os.Getenv("XDG_RUNTIME_DIR"); xdgRuntime != "" {
		return filepath.Join(xdgRuntime, "vibeman"), nil
	}

	// Fall back to /tmp with user ID
	uid := os.Getuid()
	return filepath.Join("/tmp", fmt.Sprintf("vibeman-%d", uid)), nil
}

// LogsDir returns the directory for storing log files
// Uses state directory as the base
func LogsDir() string {
	stateDir, err := StateDir()
	if err != nil {
		// Fallback to data directory
		dataDir, _ := DataDir()
		return filepath.Join(dataDir, "logs")
	}
	return filepath.Join(stateDir, "logs")
}

// Legacy paths for migration

// LegacyDir returns the old ~/.vibeman directory path for migration purposes
func LegacyDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".vibeman"), nil
}
