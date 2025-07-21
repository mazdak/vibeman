import Foundation
import Security
import AuthenticationServices
import Cocoa
import os.log

/// Secure CLI installer that handles privileged operations for installing the vibeman CLI tool
public class CLIInstaller: NSObject {
    
    // MARK: - Constants
    
    private static let cliInstallPath = "/usr/local/bin/vibeman"
    private static let backupPath = "/usr/local/bin/vibeman.backup"
    
    // MARK: - Properties
    
    private let logger = Logger(subsystem: "com.vibeman.app", category: "CLIInstaller")
    private weak var parentWindow: NSWindow?
    
    // MARK: - Initialization
    
    public init(parentWindow: NSWindow? = nil) {
        self.parentWindow = parentWindow
        super.init()
    }
    
    // MARK: - Public Interface
    
    /// Installation status of the CLI tool
    public enum InstallationStatus {
        case notInstalled
        case installed(version: String)
        case outdated(currentVersion: String, bundledVersion: String)
        case unknown
    }
    
    /// Installation result
    public enum InstallationResult {
        case success
        case cancelled
        case failed(Error)
    }
    
    /// Check the current installation status of the CLI tool
    public func checkInstallationStatus() -> InstallationStatus {
        guard FileManager.default.fileExists(atPath: Self.cliInstallPath) else {
            return .notInstalled
        }
        
        // Get installed version
        let installedVersion = getInstalledVersion()
        
        // Get bundled version
        guard let bundledVersion = getBundledVersion() else {
            return installedVersion.isEmpty ? .notInstalled : .installed(version: installedVersion)
        }
        
        if installedVersion.isEmpty {
            return .unknown
        }
        
        if installedVersion != bundledVersion {
            return .outdated(currentVersion: installedVersion, bundledVersion: bundledVersion)
        }
        
        return .installed(version: installedVersion)
    }
    
    /// Install or update the CLI tool
    public func installCLI(completion: @escaping (InstallationResult) -> Void) {
        logger.info("Starting CLI installation process")
        
        // Validate bundled executable exists
        guard let bundledExecutablePath = getBundledExecutablePath() else {
            let error = CLIInstallerError.bundledExecutableNotFound
            logger.error("Bundled executable not found")
            completion(.failed(error))
            return
        }
        
        // Show confirmation dialog
        showInstallationConfirmationDialog { [weak self] confirmed in
            guard confirmed else {
                completion(.cancelled)
                return
            }
            
            self?.performInstallation(from: bundledExecutablePath, completion: completion)
        }
    }
    
    /// Uninstall the CLI tool
    public func uninstallCLI(completion: @escaping (InstallationResult) -> Void) {
        logger.info("Starting CLI uninstallation process")
        
        let status = checkInstallationStatus()
        switch status {
        case .notInstalled:
            completion(.success) // Already uninstalled
            return
        default:
            break
        }
        
        // Show confirmation dialog
        showUninstallationConfirmationDialog { [weak self] confirmed in
            guard confirmed else {
                completion(.cancelled)
                return
            }
            
            self?.performUninstallation(completion: completion)
        }
    }
    
    // MARK: - Private Implementation
    
    private func getBundledExecutablePath() -> URL? {
        let bundlePath = Bundle.main.bundlePath
        
        let executablePath = URL(fileURLWithPath: bundlePath)
            .appendingPathComponent("Contents")
            .appendingPathComponent("Resources")
            .appendingPathComponent("vibeman")
        
        guard FileManager.default.fileExists(atPath: executablePath.path) else {
            return nil
        }
        
        return executablePath
    }
    
    private func getInstalledVersion() -> String {
        let task = Process()
        task.executableURL = URL(fileURLWithPath: Self.cliInstallPath)
        task.arguments = ["--version"]
        
        let pipe = Pipe()
        task.standardOutput = pipe
        task.standardError = pipe
        
        do {
            try task.run()
            task.waitUntilExit()
            
            if task.terminationStatus == 0 {
                let data = pipe.fileHandleForReading.readDataToEndOfFile()
                if let output = String(data: data, encoding: .utf8) {
                    return output.trimmingCharacters(in: .whitespacesAndNewlines)
                }
            }
        } catch {
            logger.error("Failed to get installed version: \(error.localizedDescription)")
        }
        
        return ""
    }
    
    private func getBundledVersion() -> String? {
        guard let executablePath = getBundledExecutablePath() else {
            return nil
        }
        
        let task = Process()
        task.executableURL = executablePath
        task.arguments = ["--version"]
        
        let pipe = Pipe()
        task.standardOutput = pipe
        task.standardError = pipe
        
        do {
            try task.run()
            task.waitUntilExit()
            
            if task.terminationStatus == 0 {
                let data = pipe.fileHandleForReading.readDataToEndOfFile()
                if let output = String(data: data, encoding: .utf8) {
                    return output.trimmingCharacters(in: .whitespacesAndNewlines)
                }
            }
        } catch {
            logger.error("Failed to get bundled version: \(error.localizedDescription)")
        }
        
        return nil
    }
    
    private func showInstallationConfirmationDialog(completion: @escaping (Bool) -> Void) {
        DispatchQueue.main.async { [weak self] in
            let alert = NSAlert()
            alert.messageText = "Install Vibeman CLI Tool"
            
            let status = self?.checkInstallationStatus() ?? .unknown
            switch status {
            case .notInstalled:
                alert.informativeText = """
                This will install the 'vibeman' command-line tool to /usr/local/bin/
                
                You will be prompted for administrator privileges to complete the installation.
                """
            case .installed(let version):
                alert.informativeText = """
                The CLI tool (version \(version)) is already installed.
                
                This will reinstall the current version. You will be prompted for administrator privileges.
                """
            case .outdated(let currentVersion, let bundledVersion):
                alert.informativeText = """
                Update available!
                
                Installed version: \(currentVersion)
                New version: \(bundledVersion)
                
                You will be prompted for administrator privileges to complete the update.
                """
            case .unknown:
                alert.informativeText = """
                This will install or update the 'vibeman' command-line tool to /usr/local/bin/
                
                You will be prompted for administrator privileges to complete the installation.
                """
            }
            
            alert.addButton(withTitle: "Install")
            alert.addButton(withTitle: "Cancel")
            alert.alertStyle = .informational
            
            if let window = self?.parentWindow {
                alert.beginSheetModal(for: window) { response in
                    completion(response == .alertFirstButtonReturn)
                }
            } else {
                let response = alert.runModal()
                completion(response == .alertFirstButtonReturn)
            }
        }
    }
    
    private func showUninstallationConfirmationDialog(completion: @escaping (Bool) -> Void) {
        DispatchQueue.main.async { [weak self] in
            let alert = NSAlert()
            alert.messageText = "Uninstall Vibeman CLI Tool"
            alert.informativeText = """
            This will remove the 'vibeman' command-line tool from /usr/local/bin/
            
            You will be prompted for administrator privileges to complete the removal.
            """
            alert.addButton(withTitle: "Uninstall")
            alert.addButton(withTitle: "Cancel")
            alert.alertStyle = .warning
            
            if let window = self?.parentWindow {
                alert.beginSheetModal(for: window) { response in
                    completion(response == .alertFirstButtonReturn)
                }
            } else {
                let response = alert.runModal()
                completion(response == .alertFirstButtonReturn)
            }
        }
    }
    
    private func performInstallation(from sourcePath: URL, completion: @escaping (InstallationResult) -> Void) {
        // Create installation script
        let script = createInstallationScript(sourcePath: sourcePath.path)
        
        // Execute with admin privileges
        executeScriptWithAdminPrivileges(script: script) { [weak self] success, error in
            DispatchQueue.main.async {
                if success {
                    self?.logger.info("CLI installation completed successfully")
                    self?.showSuccessDialog(message: "Vibeman CLI tool has been installed successfully!")
                    completion(.success)
                } else {
                    let installError = error ?? CLIInstallerError.installationFailed
                    self?.logger.error("CLI installation failed: \(installError.localizedDescription)")
                    self?.showErrorDialog(error: installError)
                    completion(.failed(installError))
                }
            }
        }
    }
    
    private func performUninstallation(completion: @escaping (InstallationResult) -> Void) {
        // Create uninstallation script
        let script = createUninstallationScript()
        
        // Execute with admin privileges
        executeScriptWithAdminPrivileges(script: script) { [weak self] success, error in
            DispatchQueue.main.async {
                if success {
                    self?.logger.info("CLI uninstallation completed successfully")
                    self?.showSuccessDialog(message: "Vibeman CLI tool has been uninstalled successfully!")
                    completion(.success)
                } else {
                    let uninstallError = error ?? CLIInstallerError.uninstallationFailed
                    self?.logger.error("CLI uninstallation failed: \(uninstallError.localizedDescription)")
                    self?.showErrorDialog(error: uninstallError)
                    completion(.failed(uninstallError))
                }
            }
        }
    }
    
    private func createInstallationScript(sourcePath: String) -> String {
        return """
        #!/bin/bash
        set -e
        
        # Validate source file exists and is executable
        if [ ! -f "\(sourcePath)" ]; then
            echo "Error: Source file does not exist: \(sourcePath)"
            exit 1
        fi
        
        if [ ! -x "\(sourcePath)" ]; then
            echo "Error: Source file is not executable: \(sourcePath)"
            exit 1
        fi
        
        # Create backup if file already exists
        if [ -f "\(Self.cliInstallPath)" ]; then
            echo "Creating backup of existing CLI tool..."
            cp "\(Self.cliInstallPath)" "\(Self.backupPath)"
        fi
        
        # Ensure target directory exists
        mkdir -p "$(dirname "\(Self.cliInstallPath)")"
        
        # Copy the new executable
        echo "Installing CLI tool..."
        cp "\(sourcePath)" "\(Self.cliInstallPath)"
        
        # Set proper permissions
        chmod 755 "\(Self.cliInstallPath)"
        
        # Verify installation
        if [ -x "\(Self.cliInstallPath)" ]; then
            echo "Installation completed successfully"
            # Clean up backup on successful installation
            [ -f "\(Self.backupPath)" ] && rm -f "\(Self.backupPath)"
        else
            echo "Error: Installation verification failed"
            # Restore backup if available
            if [ -f "\(Self.backupPath)" ]; then
                echo "Restoring backup..."
                mv "\(Self.backupPath)" "\(Self.cliInstallPath)"
            fi
            exit 1
        fi
        """
    }
    
    private func createUninstallationScript() -> String {
        return """
        #!/bin/bash
        set -e
        
        # Check if CLI tool exists
        if [ ! -f "\(Self.cliInstallPath)" ]; then
            echo "CLI tool is not installed, nothing to remove"
            exit 0
        fi
        
        # Remove the CLI tool
        echo "Removing CLI tool..."
        rm -f "\(Self.cliInstallPath)"
        
        # Clean up any backup
        [ -f "\(Self.backupPath)" ] && rm -f "\(Self.backupPath)"
        
        echo "Uninstallation completed successfully"
        """
    }
    
    private func executeScriptWithAdminPrivileges(script: String, completion: @escaping (Bool, Error?) -> Void) {
        // Write script to temporary file
        let tempDir = NSTemporaryDirectory()
        let scriptPath = (tempDir as NSString).appendingPathComponent("vibeman_install_\(UUID().uuidString).sh")
        
        do {
            try script.write(toFile: scriptPath, atomically: true, encoding: .utf8)
            
            // Make script executable
            let attributes = [FileAttributeKey.posixPermissions: 0o755]
            try FileManager.default.setAttributes(attributes, ofItemAtPath: scriptPath)
            
            // Execute script with admin privileges using osascript
            let osascriptSource = """
            do shell script "bash '\(scriptPath)'" with administrator privileges
            """
            
            let task = Process()
            task.executableURL = URL(fileURLWithPath: "/usr/bin/osascript")
            task.arguments = ["-e", osascriptSource]
            
            let pipe = Pipe()
            task.standardOutput = pipe
            task.standardError = pipe
            
            task.terminationHandler = { process in
                // Clean up temporary script
                try? FileManager.default.removeItem(atPath: scriptPath)
                
                let success = process.terminationStatus == 0
                var error: Error?
                
                if !success {
                    let data = pipe.fileHandleForReading.readDataToEndOfFile()
                    let errorOutput = String(data: data, encoding: .utf8) ?? "Unknown error"
                    error = CLIInstallerError.scriptExecutionFailed(errorOutput)
                }
                
                completion(success, error)
            }
            
            try task.run()
            
        } catch {
            // Clean up temporary script on error
            try? FileManager.default.removeItem(atPath: scriptPath)
            completion(false, error)
        }
    }
    
    private func showSuccessDialog(message: String) {
        let alert = NSAlert()
        alert.messageText = "Success"
        alert.informativeText = message
        alert.alertStyle = .informational
        alert.addButton(withTitle: "OK")
        
        if let window = parentWindow {
            alert.beginSheetModal(for: window, completionHandler: nil)
        } else {
            alert.runModal()
        }
    }
    
    private func showErrorDialog(error: Error) {
        let alert = NSAlert()
        alert.messageText = "Installation Error"
        alert.informativeText = error.localizedDescription
        alert.alertStyle = .critical
        alert.addButton(withTitle: "OK")
        
        if let window = parentWindow {
            alert.beginSheetModal(for: window, completionHandler: nil)
        } else {
            alert.runModal()
        }
    }
}

// MARK: - Error Types

public enum CLIInstallerError: LocalizedError {
    case bundledExecutableNotFound
    case installationFailed
    case uninstallationFailed
    case scriptExecutionFailed(String)
    case authorizationFailed
    
    public var errorDescription: String? {
        switch self {
        case .bundledExecutableNotFound:
            return "The vibeman executable could not be found in the application bundle."
        case .installationFailed:
            return "Failed to install the CLI tool. Please try again."
        case .uninstallationFailed:
            return "Failed to uninstall the CLI tool. Please try again."
        case .scriptExecutionFailed(let details):
            return "Installation script failed: \(details)"
        case .authorizationFailed:
            return "Administrator authorization was denied or failed."
        }
    }
    
    public var recoverySuggestion: String? {
        switch self {
        case .bundledExecutableNotFound:
            return "Please reinstall the application or contact support."
        case .installationFailed, .uninstallationFailed:
            return "Make sure you have administrator privileges and try again."
        case .scriptExecutionFailed:
            return "Check the system console for more details or contact support."
        case .authorizationFailed:
            return "Please provide administrator credentials when prompted."
        }
    }
}