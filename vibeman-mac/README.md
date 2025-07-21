# VibemanKit

A Swift framework for integrating the vibeman development environment management server into macOS applications.

## Overview

VibemanKit provides a robust process management system for embedding the vibeman Go server into Swift/macOS applications. It handles server lifecycle, health monitoring, automatic restarts, and graceful shutdown.

## Features

- **Process Lifecycle Management**: Start, stop, and restart the embedded vibeman server
- **Health Monitoring**: Automatic health checks with configurable intervals
- **Auto-Restart**: Intelligent restart logic with exponential backoff
- **Delegate Pattern**: Rich event system for monitoring server status
- **Thread-Safe**: All operations are thread-safe and main-queue-aware
- **Error Handling**: Comprehensive error types with descriptive messages
- **Logging**: Built-in structured logging using OSLog

## Requirements

- macOS 12.0+
- Swift 5.9+
- Xcode 14.0+

## Installation

### Swift Package Manager

Add the following to your `Package.swift` file:

```swift
dependencies: [
    .package(path: "./swift-wrapper")
]
```

Or add it through Xcode:
1. File → Add Package Dependencies
2. Enter the local path to the swift-wrapper directory

## Quick Start

### Basic Usage

```swift
import VibemanKit

class AppDelegate: NSObject, NSApplicationDelegate {
    let appLauncher = AppLauncher()
    
    func applicationDidFinishLaunching(_ notification: Notification) {
        appLauncher.launch { error in
            if let error = error {
                print("Failed to start: \(error)")
            } else {
                print("Server running at: \(self.appLauncher.serverURL)")
                // Your app can now connect to the server
            }
        }
    }
}
```

### Advanced Usage with Delegate

```swift
import VibemanKit

class AppDelegate: NSObject, NSApplicationDelegate, ProcessManagerDelegate {
    let appLauncher = AppLauncher(serverPort: 8081)
    
    func applicationDidFinishLaunching(_ notification: Notification) {
        // Configure the process manager
        appLauncher.processManager.delegate = self
        appLauncher.processManager.autoRestart = true
        appLauncher.processManager.maxRestartAttempts = 5
        
        appLauncher.launch { error in
            // Handle startup result
        }
    }
    
    // MARK: - ProcessManagerDelegate
    
    func processManager(_ manager: ProcessManager, didChangeStatus status: ProcessStatus) {
        print("Server status: \(status)")
    }
    
    func processManagerDidStartServer(_ manager: ProcessManager) {
        print("✅ Server started successfully")
    }
    
    func processManager(_ manager: ProcessManager, didStopWithError error: Error?) {
        if let error = error {
            print("❌ Server error: \(error)")
        }
    }
}
```

## API Reference

### AppLauncher

The main class for coordinating application startup and server management.

```swift
class AppLauncher {
    let processManager: ProcessManager
    var autoStartServer: Bool = true
    var startupTimeout: TimeInterval = 30.0
    var waitForServerReady: Bool = true
    
    init(serverPort: UInt16 = 8081, serverHost: String = "localhost")
    
    func launch(completion: @escaping (Error?) -> Void)
    func shutdown(completion: ((Error?) -> Void)? = nil)
    func checkServerReady(completion: @escaping (Bool, Error?) -> Void)
    
    var serverURL: URL { get }
}
```

### ProcessManager

Low-level process management for the vibeman server.

```swift
class ProcessManager {
    weak var delegate: ProcessManagerDelegate?
    var status: ProcessStatus { get }
    let serverPort: UInt16
    let serverHost: String
    var autoRestart: Bool = true
    var maxRestartAttempts: UInt = 3
    var restartDelay: TimeInterval = 2.0
    
    init(port: UInt16 = 8081, host: String = "localhost")
    
    func startServer(completion: @escaping (Error?) -> Void)
    func stopServer(completion: ((Error?) -> Void)? = nil)
    func restartServer(completion: @escaping (Error?) -> Void)
    func checkHealth(completion: @escaping (Bool, Error?) -> Void)
    
    var serverURL: URL { get }
}
```

### ProcessManagerDelegate

```swift
protocol ProcessManagerDelegate: AnyObject {
    func processManager(_ manager: ProcessManager, didChangeStatus status: ProcessStatus)
    
    // Optional methods
    func processManagerDidStartServer(_ manager: ProcessManager)
    func processManager(_ manager: ProcessManager, didStopWithError error: Error?)
    func processManager(_ manager: ProcessManager, willAttemptRestart attempt: UInt, of maxAttempts: UInt)
    func processManager(_ manager: ProcessManager, autoRestartFailedWithError error: Error)
    func processManager(_ manager: ProcessManager, healthCheckFailedWithError error: Error)
    func processManagerDidBecomeHealthy(_ manager: ProcessManager)
}
```

### ProcessStatus

```swift
enum ProcessStatus {
    case stopped
    case starting
    case running
    case stopping
}
```

## Binary Deployment

The framework expects the vibeman server binary to be located at:

1. `Contents/MacOS/vibeman-server` (in app bundle)
2. `./vibeman` (development fallback)

Make sure to include the compiled Go binary in your app bundle's `Contents/MacOS/` directory.

## Configuration

### Server Configuration

```swift
// Custom port and host
let launcher = AppLauncher(serverPort: 3000, serverHost: "127.0.0.1")

// Startup behavior
launcher.autoStartServer = true
launcher.startupTimeout = 60.0
launcher.waitForServerReady = true
```

### Process Management

```swift
// Auto-restart configuration
launcher.processManager.autoRestart = true
launcher.processManager.maxRestartAttempts = 10
launcher.processManager.restartDelay = 5.0
```

## Error Handling

VibemanKit provides comprehensive error types:

### ProcessError

- `binaryNotFound`: Server binary not found
- `alreadyRunning`: Server is already running
- `healthCheckFailed`: Health check failed
- `startupTimeout`: Server startup timed out
- `unexpectedTermination`: Server crashed

### AppLauncherError

- `alreadyLaunching`: Launch already in progress
- `startupTimeout`: Startup timed out
- `serverNotReady`: Server failed to become ready

## Logging

VibemanKit uses OSLog for structured logging:

```swift
import OSLog

// View logs in Console.app by filtering for:
// Subsystem: com.vibeman.swift-wrapper
// Categories: ProcessManager, AppLauncher
```

## Testing

Run tests using Swift Package Manager:

```bash
cd swift-wrapper
swift test
```

Or through Xcode:
1. Open Package.swift in Xcode
2. Product → Test

## Examples

See `Examples.swift` for complete integration examples including:

- Basic app delegate setup
- Advanced configuration with status monitoring
- WebView integration
- Custom server configuration

## Building the Vibeman Binary

To build the vibeman server binary for embedding:

```bash
# From the vibeman project root
cd /Users/mazdak/Code/vibeman
go build -o vibeman-server

# Copy to your app bundle
cp vibeman-server /path/to/your/app.app/Contents/MacOS/
```

## License

See the main vibeman project for license information.