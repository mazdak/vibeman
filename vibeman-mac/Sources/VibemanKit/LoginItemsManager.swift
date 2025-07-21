import Foundation
import ServiceManagement
import os.log

/// Manager for handling login items functionality using the modern SMAppService API (macOS 13+)
/// Provides auto-start functionality with proper user consent and permission handling
@available(macOS 13.0, *)
public class LoginItemsManager: ObservableObject {
    private let logger = Logger(subsystem: "com.vibeman.app", category: "LoginItemsManager")
    private let serviceIdentifier: String
    
    /// Current status of the login item
    @Published public private(set) var isEnabled: Bool = false
    
    /// Whether the service is available and can be managed
    @Published public private(set) var isAvailable: Bool = false
    
    /// Last error encountered during operations
    @Published public private(set) var lastError: LoginItemError?
    
    public init(serviceIdentifier: String = "com.vibeman.app.helper") {
        self.serviceIdentifier = serviceIdentifier
        checkCurrentStatus()
    }
    
    /// Check the current status of the login item
    public func checkCurrentStatus() {
        Task { @MainActor in
            let service = SMAppService.loginItem(identifier: serviceIdentifier)
            self.isEnabled = service.status == .enabled
            self.isAvailable = service.status != .notFound
            self.lastError = nil
            
            logger.info("Login item status checked: enabled=\(self.isEnabled), available=\(self.isAvailable)")
        }
    }
    
    /// Enable auto-start at login
    /// - Returns: Success status
    @discardableResult
    public func enable() async -> Bool {
        do {
            let service = SMAppService.loginItem(identifier: serviceIdentifier)
            try await service.register()
            
            await MainActor.run {
                self.isEnabled = true
                self.lastError = nil
            }
            
            logger.info("Login item enabled successfully")
            return true
            
        } catch {
            await MainActor.run {
                self.lastError = .enableFailed(error)
                self.isEnabled = false
            }
            
            logger.error("Failed to enable login item: \(error.localizedDescription)")
            return false
        }
    }
    
    /// Disable auto-start at login
    /// - Returns: Success status
    @discardableResult
    public func disable() async -> Bool {
        do {
            let service = SMAppService.loginItem(identifier: serviceIdentifier)
            try await service.unregister()
            
            await MainActor.run {
                self.isEnabled = false
                self.lastError = nil
            }
            
            logger.info("Login item disabled successfully")
            return true
            
        } catch {
            await MainActor.run {
                self.lastError = .disableFailed(error)
            }
            
            logger.error("Failed to disable login item: \(error.localizedDescription)")
            return false
        }
    }
    
    /// Toggle the current login item state
    /// - Returns: Success status
    @discardableResult
    public func toggle() async -> Bool {
        if isEnabled {
            return await disable()
        } else {
            return await enable()
        }
    }
    
    /// Request user authorization for login items
    /// This shows the system permission dialog
    public func requestAuthorization() async -> Bool {
        do {
            let service = SMAppService.loginItem(identifier: serviceIdentifier)
            
            // The register call will automatically prompt for permission if needed
            try await service.register()
            
            await MainActor.run {
                self.isEnabled = true
                self.lastError = nil
            }
            
            logger.info("User authorization granted and login item enabled")
            return true
            
        } catch {
            await MainActor.run {
                self.lastError = .authorizationFailed(error)
            }
            
            logger.error("Authorization failed: \(error.localizedDescription)")
            return false
        }
    }
    
    /// Get user-friendly error message for display
    public func getErrorMessage() -> String? {
        guard let error = lastError else { return nil }
        
        switch error {
        case .statusCheckFailed:
            return "Unable to check auto-start status"
        case .enableFailed(let underlyingError):
            return getEnableErrorMessage(underlyingError)
        case .disableFailed:
            return "Failed to disable auto-start"
        case .authorizationFailed:
            return "Permission denied. Please allow Vibeman to start at login in System Settings"
        case .notSupported:
            return "Auto-start at login is not supported on this macOS version"
        }
    }
    
    // MARK: - Private Methods
    
    private func getEnableErrorMessage(_ error: Error) -> String {
        let errorString = error.localizedDescription
        
        // Check for common error patterns and provide user-friendly messages
        if errorString.contains("not authorized") || errorString.contains("permission") {
            return "Permission required. Please allow Vibeman to start at login in System Settings > General > Login Items"
        } else if errorString.contains("not found") {
            return "Login helper not found. Please reinstall Vibeman"
        } else if errorString.contains("invalid") {
            return "App configuration error. Please reinstall Vibeman"
        }
        
        return "Failed to enable auto-start: \(errorString)"
    }
}

// MARK: - Error Types

public enum LoginItemError: Error, LocalizedError {
    case statusCheckFailed(Error)
    case enableFailed(Error)
    case disableFailed(Error)
    case authorizationFailed(Error)
    case notSupported
    
    public var errorDescription: String? {
        switch self {
        case .statusCheckFailed(let error):
            return "Status check failed: \(error.localizedDescription)"
        case .enableFailed(let error):
            return "Enable failed: \(error.localizedDescription)"
        case .disableFailed(let error):
            return "Disable failed: \(error.localizedDescription)"
        case .authorizationFailed(let error):
            return "Authorization failed: \(error.localizedDescription)"
        case .notSupported:
            return "Login items not supported on this macOS version"
        }
    }
}

// MARK: - Legacy Support

/// Legacy login items manager for macOS 12 and earlier
/// Uses the older ServiceManagement APIs with reduced functionality
@available(macOS, deprecated: 13.0, message: "Use LoginItemsManager with SMAppService instead")
public class LegacyLoginItemsManager {
    private let logger = Logger(subsystem: "com.vibeman.app", category: "LegacyLoginItemsManager")
    private let helperBundleIdentifier: String
    
    public init(helperBundleIdentifier: String = "com.vibeman.app.helper") {
        self.helperBundleIdentifier = helperBundleIdentifier
    }
    
    /// Check if login item is enabled (legacy method)
    public func isEnabled() -> Bool {
        // For legacy support, we'll use a simplified check
        // This is less reliable than the modern API
        let _ = [
            kCFBundleIdentifierKey as String: helperBundleIdentifier
        ] as CFDictionary
        
        // Note: This is a simplified implementation
        // In a full implementation, you would check the actual login items
        return false
    }
    
    /// Enable login item (legacy method)
    public func enable() -> Bool {
        logger.warning("Legacy login items not fully implemented - consider upgrading to macOS 13+")
        return false
    }
    
    /// Disable login item (legacy method)
    public func disable() -> Bool {
        logger.warning("Legacy login items not fully implemented - consider upgrading to macOS 13+")
        return false
    }
}