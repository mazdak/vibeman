import Foundation
import OSLog

/// Manages the lifecycle of the embedded vibeman Go server process
@objc public final class ProcessManager: NSObject {
    
    // MARK: - Public Properties
    
    /// Current status of the server process
    @objc public var status: ProcessStatus {
        lock.withLock { _status }
    }
    
    /// The port the server is configured to run on
    @objc public let serverPort: UInt16
    
    /// The host the server binds to
    @objc public let serverHost: String
    
    /// Whether the process manager should automatically restart crashed processes
    @objc public var autoRestart: Bool = true
    
    /// Maximum number of restart attempts before giving up
    @objc public var maxRestartAttempts: UInt = 3
    
    /// Delay between restart attempts in seconds
    @objc public var restartDelay: TimeInterval = 2.0
    
    // MARK: - Private Properties
    
    private let logger = Logger(subsystem: "com.vibeman.swift-wrapper", category: "ProcessManager")
    private let lock = NSLock()
    
    private var _status: ProcessStatus = .stopped
    private var process: Process?
    private let processQueue = DispatchQueue(label: "com.vibeman.process-manager", qos: .default)
    
    // MARK: - Initialization
    
    /// Initialize the ProcessManager
    /// - Parameters:
    ///   - port: Port for the server to bind to (default: 8081)
    ///   - host: Host for the server to bind to (default: "localhost")
    @objc public init(port: UInt16 = 8081, host: String = "localhost") {
        self.serverPort = port
        self.serverHost = host
        super.init()
        
        logger.info("ProcessManager initialized for \(host):\(port)")
    }
    
    deinit {
        stopServer()
    }
    
    // MARK: - Public Methods
    
    /// Start the vibeman server process
    /// - Parameter completion: Completion handler called when startup completes or fails
    @objc public func startServer(completion: @escaping (Error?) -> Void) {
        processQueue.async { [weak self] in
            self?._startServer(completion: completion)
        }
    }
    
    /// Stop the vibeman server process
    /// - Parameter completion: Completion handler called when shutdown completes
    @objc public func stopServer(completion: ((Error?) -> Void)? = nil) {
        processQueue.async { [weak self] in
            self?._stopServer(completion: completion)
        }
    }
    
    /// Check if the server is responding to health checks
    /// - Parameter completion: Completion handler with health status
    @objc public func checkHealth(completion: @escaping (Bool, Error?) -> Void) {
        let url = URL(string: "http://\(serverHost):\(serverPort)/api/health")!
        
        let task = URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error {
                completion(false, error)
                return
            }
            
            guard let httpResponse = response as? HTTPURLResponse else {
                completion(false, ProcessError.healthCheckFailed("Invalid response"))
                return
            }
            
            let isHealthy = httpResponse.statusCode == 200
            completion(isHealthy, nil)
        }
        
        task.resume()
    }
    
    /// Get the server URL
    @objc public var serverURL: URL {
        return URL(string: "http://\(serverHost):\(serverPort)")!
    }
    
    // MARK: - Private Implementation
    
    private func _startServer(completion: @escaping (Error?) -> Void) {
        lock.withLock {
            if _status == .running || _status == .starting {
                DispatchQueue.main.async {
                    completion(ProcessError.alreadyRunning)
                }
                return
            }
            _status = .starting
        }
        
        logger.info("Starting vibeman server...")
        
        do {
            let binaryPath = try findServerBinary()
            let process = try createServerProcess(binaryPath: binaryPath)
            
            self.process = process
            try process.run()
            
            lock.withLock {
                _status = .running
            }
            
            logger.info("Server process started with PID: \(process.processIdentifier)")
            
            DispatchQueue.main.async {
                completion(nil)
            }
            
        } catch {
            lock.withLock {
                _status = .stopped
            }
            
            logger.error("Failed to start server: \(error.localizedDescription)")
            
            DispatchQueue.main.async {
                completion(error)
            }
        }
    }
    
    private func _stopServer(completion: ((Error?) -> Void)?) {
        lock.withLock {
            if _status == .stopped || _status == .stopping {
                DispatchQueue.main.async {
                    completion?(nil)
                }
                return
            }
            _status = .stopping
        }
        
        logger.info("Stopping vibeman server...")
        
        if let process = process, process.isRunning {
            process.terminate()
        }
        
        process = nil
        
        lock.withLock {
            _status = .stopped
        }
        
        logger.info("Server stopped")
        
        DispatchQueue.main.async {
            completion?(nil)
        }
    }
    
    private func findServerBinary() throws -> String {
        // Check if we're in a bundle (app)
        if let bundlePath = Bundle.main.executablePath {
            let bundleDir = URL(fileURLWithPath: bundlePath).deletingLastPathComponent()
            let binaryPath = bundleDir.appendingPathComponent("vibeman").path
            
            if FileManager.default.fileExists(atPath: binaryPath) {
                return binaryPath
            }
        }
        
        // Check relative to main bundle
        let bundlePath = Bundle.main.bundlePath
        if !bundlePath.isEmpty {
            let macOSDir = URL(fileURLWithPath: bundlePath).appendingPathComponent("Contents/MacOS")
            let binaryPath = macOSDir.appendingPathComponent("vibeman").path
            
            if FileManager.default.fileExists(atPath: binaryPath) {
                return binaryPath
            }
        }
        
        // Development fallback - check relative to current directory
        let currentDir = FileManager.default.currentDirectoryPath
        let devBinaryPath = "\(currentDir)/vibeman"
        
        if FileManager.default.fileExists(atPath: devBinaryPath) {
            return devBinaryPath
        }
        
        throw ProcessError.binaryNotFound("Could not find vibeman binary")
    }
    
    private func createServerProcess(binaryPath: String) throws -> Process {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: binaryPath)
        
        // Set up arguments for server mode with new command structure
        process.arguments = [
            "server",
            "start",
            "--port", "\(serverPort)",
            "--host", "\(serverHost)"
        ]
        
        // Set up environment
        var environment = ProcessInfo.processInfo.environment
        environment["VIBEMAN_LOG_LEVEL"] = "info"
        environment["VIBEMAN_LOG_FORMAT"] = "json"
        process.environment = environment
        
        return process
    }
}

// MARK: - Process Status

@objc public enum ProcessStatus: Int, CaseIterable, CustomStringConvertible {
    case stopped = 0
    case starting = 1
    case running = 2
    case stopping = 3
    
    public var description: String {
        switch self {
        case .stopped: return "stopped"
        case .starting: return "starting"
        case .running: return "running"
        case .stopping: return "stopping"
        }
    }
}

// MARK: - Process Errors

public enum ProcessError: LocalizedError {
    case binaryNotFound(String)
    case alreadyRunning
    case healthCheckFailed(String)
    case startupTimeout
    case unexpectedTermination(Int32)
    
    public var errorDescription: String? {
        switch self {
        case .binaryNotFound(let message):
            return "Binary not found: \(message)"
        case .alreadyRunning:
            return "Server is already running"
        case .healthCheckFailed(let message):
            return "Health check failed: \(message)"
        case .startupTimeout:
            return "Server startup timed out"
        case .unexpectedTermination(let exitCode):
            return "Server terminated unexpectedly with exit code: \(exitCode)"
        }
    }
}

// MARK: - NSLock Extension

private extension NSLock {
    func withLock<T>(_ block: () throws -> T) rethrows -> T {
        lock()
        defer { unlock() }
        return try block()
    }
}