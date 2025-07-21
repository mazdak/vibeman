import XCTest
@testable import VibemanKit

final class ProcessManagerTests: XCTestCase {
    
    func testProcessManagerInitialization() {
        let manager = ProcessManager(port: 8081, host: "localhost")
        XCTAssertEqual(manager.serverPort, 8081)
        XCTAssertEqual(manager.serverHost, "localhost")
        XCTAssertEqual(manager.status, .stopped)
        XCTAssertTrue(manager.autoRestart)
        XCTAssertEqual(manager.maxRestartAttempts, 3)
        XCTAssertEqual(manager.restartDelay, 2.0)
    }
    
    func testServerURL() {
        let manager = ProcessManager(port: 9999, host: "127.0.0.1")
        let expectedURL = URL(string: "http://127.0.0.1:9999")!
        XCTAssertEqual(manager.serverURL, expectedURL)
    }
    
    func testProcessStatusDescriptions() {
        XCTAssertEqual(ProcessStatus.stopped.description, "stopped")
        XCTAssertEqual(ProcessStatus.starting.description, "starting")
        XCTAssertEqual(ProcessStatus.running.description, "running")
        XCTAssertEqual(ProcessStatus.stopping.description, "stopping")
    }
    
    func testProcessErrorDescriptions() {
        let binaryNotFound = ProcessError.binaryNotFound("test message")
        XCTAssertEqual(binaryNotFound.errorDescription, "Binary not found: test message")
        
        let alreadyRunning = ProcessError.alreadyRunning
        XCTAssertEqual(alreadyRunning.errorDescription, "Server is already running")
        
        let healthCheckFailed = ProcessError.healthCheckFailed("connection refused")
        XCTAssertEqual(healthCheckFailed.errorDescription, "Health check failed: connection refused")
        
        let startupTimeout = ProcessError.startupTimeout
        XCTAssertEqual(startupTimeout.errorDescription, "Server startup timed out")
        
        let unexpectedTermination = ProcessError.unexpectedTermination(137)
        XCTAssertEqual(unexpectedTermination.errorDescription, "Server terminated unexpectedly with exit code: 137")
    }
    
    func testHealthCheckURL() {
        let manager = ProcessManager(port: 8080, host: "localhost")
        
        // We can't easily test the actual health check without mocking URLSession,
        // but we can verify the URL construction by checking the server URL
        let expectedBaseURL = "http://localhost:8080"
        XCTAssertEqual(manager.serverURL.absoluteString, expectedBaseURL)
        
        // The actual health check would hit /api/health endpoint
    }
    
    func testAutoRestartConfiguration() {
        let manager = ProcessManager()
        
        // Test default values
        XCTAssertTrue(manager.autoRestart)
        XCTAssertEqual(manager.maxRestartAttempts, 3)
        XCTAssertEqual(manager.restartDelay, 2.0)
        
        // Test that properties are mutable
        manager.autoRestart = false
        manager.maxRestartAttempts = 5
        manager.restartDelay = 5.0
        
        XCTAssertFalse(manager.autoRestart)
        XCTAssertEqual(manager.maxRestartAttempts, 5)
        XCTAssertEqual(manager.restartDelay, 5.0)
    }
}