import SwiftUI
import AppKit
import WebKit
import Sparkle
import ServiceManagement
import os.log
import VibemanKit

@main
struct VibemanApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
    
    var body: some Scene {
        // This is a menu bar app, so we just need to define menu commands
        // All windows are created programmatically
        WindowGroup {
            EmptyView()
                .frame(width: 0, height: 0)
                .onAppear {
                    // Hide the empty window immediately
                    NSApplication.shared.windows.first?.orderOut(nil)
                }
        }
        .windowStyle(.hiddenTitleBar)
        .commands {
            CommandGroup(replacing: .appSettings) {
                Button("Preferences...") {
                    appDelegate.openPreferences()
                }
                .keyboardShortcut(",", modifiers: .command)
            }
            CommandGroup(replacing: .windowArrangement) {
                Button("Close Window") {
                    NSApplication.shared.keyWindow?.orderOut(nil)
                }
                .keyboardShortcut("w", modifiers: .command)
            }
        }
    }
}

@MainActor
class AppDelegate: NSObject, NSApplicationDelegate, SPUUpdaterDelegate {
    var statusItem: NSStatusItem?
    private var windowController = WindowController()
    private weak var mainWindow: NSWindow?
    private var mainWindowDelegate: MainWindowDelegate?
    private var serverProcess: Process?
    private var webServerProcess: Process?
    private var vibemanConfig: VibemanConfig?
    private let logger = Logger(subsystem: "com.vibeman.app", category: "AppDelegate")
    private lazy var cliInstaller = CLIInstaller()
    
    // Login items management
    private var loginItemMenuItem: NSMenuItem?
    
    func applicationDidFinishLaunching(_ notification: Notification) {
        // Skip UI initialization in test environment
        let isTestEnvironment = NSClassFromString("XCTestCase") != nil
        if isTestEnvironment {
            logger.info("Test environment detected - skipping UI initialization")
            return
        }
        
        // Setup app configuration
        setupApp()
        
        // Setup login item if enabled
        setupLoginItem()
        
        // Create menu bar item
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        
        if let button = statusItem?.button {
            button.image = createMenuBarIcon()
            button.action = #selector(toggleMainWindow)
            button.target = self
        }
        
        // Create menu
        let menu = NSMenu()
        menu.addItem(NSMenuItem(title: "Open Vibeman", action: #selector(toggleMainWindow), keyEquivalent: ""))
        menu.addItem(NSMenuItem.separator())
        
        menu.addItem(NSMenuItem(title: "Install CLI", action: #selector(installCLI), keyEquivalent: ""))
        menu.addItem(NSMenuItem(title: "Check for Updates...", action: #selector(checkForUpdates), keyEquivalent: ""))
        menu.addItem(NSMenuItem.separator())
        
        setupLoginItemsMenu(menu)
        
        menu.addItem(NSMenuItem.separator())
        menu.addItem(NSMenuItem(title: "Quit Vibeman", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q"))
        
        statusItem?.menu = menu
        
        // Load configuration
        loadConfiguration()
        
        // Start the Go server processes
        startServerProcesses()
        
        // Setup Sparkle for auto-updates
        setupSparkle()
        
        logger.info("Vibeman app launched successfully")
    }
    
    private func setupApp() {
        // Configure app to be a menu bar only app (no dock icon)
        NSApp.setActivationPolicy(.accessory)
        logger.info("Set app activation policy to .accessory (menu bar only)")
    }
    
    private func setupLoginItem() {
        let startAtLogin = UserDefaults.standard.bool(forKey: "startAtLogin")
        
        if startAtLogin {
            // Only try to register if we're in a real app context, not in tests
            if Bundle.main.bundleIdentifier != nil && !isRunningInTests() {
                if #available(macOS 13.0, *) {
                    do {
                        try SMAppService.mainApp.register()
                        logger.info("Successfully registered login item during startup")
                    } catch {
                        logger.error("Failed to register login item during startup: \(error.localizedDescription)")
                    }
                } else {
                    logger.warning("Login items require macOS 13.0 or later")
                }
            }
        }
    }
    
    private func isRunningInTests() -> Bool {
        return NSClassFromString("XCTestCase") != nil
    }
    
    private func hasNotch() -> Bool {
        // Check if the Mac has a notch by looking at the safe area insets
        if #available(macOS 12.0, *) {
            if let screen = NSScreen.main {
                return screen.safeAreaInsets.top > 0
            }
        }
        return false
    }
    
    private func createMenuBarIcon() -> NSImage? {
        // Try loading the template icon (for proper dark/light mode support)
        if let url = Bundle.module.url(forResource: "Vibeman-Menu-Icon-Template", withExtension: "png"),
           let image = NSImage(contentsOf: url) {
            // Adaptive sizing based on whether the Mac has a notch
            let iconSize: CGFloat = hasNotch() ? 22 : 20
            image.size = NSSize(width: iconSize, height: iconSize)
            // Template images automatically adapt to dark/light menu bars
            image.isTemplate = true
            return image
        }
        
        // Fallback to system symbol if custom icon not found
        print("Warning: Could not load Vibeman-Menu-Icon-Template.png, falling back to system icon")
        let config = NSImage.SymbolConfiguration(pointSize: 16, weight: .medium)
        let image = NSImage(systemSymbolName: "server.rack", accessibilityDescription: "Vibeman")?.withSymbolConfiguration(config)
        image?.isTemplate = true
        return image
    }
    
    private func loadConfiguration() {
        do {
            vibemanConfig = try VibemanConfigManager.load()
            logger.info("Loaded configuration: base port = \(self.vibemanConfig?.server.basePort ?? 0)")
        } catch {
            logger.error("Failed to load configuration: \(error.localizedDescription)")
            vibemanConfig = VibemanConfigManager.defaultConfig()
        }
    }
    
    private func startServerProcesses() {
        // Start the main Go server
        startGoServer()
        
        // Start the web server
        startWebServer()
    }
    
    private func startGoServer() {
        guard let executableURL = getVibemanExecutablePath() else {
            logger.error("Could not find vibeman executable")
            showError("Could not find vibeman executable. Please ensure vibeman is built and available.")
            return
        }
        
        let basePort = vibemanConfig?.server.basePort ?? 8080
        
        serverProcess = Process()
        serverProcess?.executableURL = executableURL
        serverProcess?.arguments = ["server", "start", "--port", String(basePort), "--daemon"]
        
        // Set up proper environment
        var environment = ProcessInfo.processInfo.environment
        environment["VIBEMAN_LOG_LEVEL"] = "info"
        serverProcess?.environment = environment
        
        do {
            try serverProcess?.run()
            logger.info("Started vibeman server on port \(basePort)")
            
            // Wait a moment for server to start
            DispatchQueue.main.asyncAfter(deadline: .now() + 2.0) { [weak self] in
                self?.checkServerHealth()
            }
        } catch {
            logger.error("Failed to start vibeman server: \(error.localizedDescription)")
            showError("Failed to start Vibeman server: \(error.localizedDescription)")
        }
    }
    
    private func startWebServer() {
        // Start the Bun web server
        let task = Process()
        task.launchPath = "/usr/bin/env"
        task.arguments = ["bun", "run", "dev"]
        task.currentDirectoryPath = getWebAppPath()
        
        // The Bun server will read the port from config automatically
        
        do {
            try task.run()
            webServerProcess = task
            let webUIPort = vibemanConfig?.server.webUIPort ?? 8081
            logger.info("Started Bun web server on port \(webUIPort)")
        } catch {
            logger.error("Failed to start web server: \(error.localizedDescription)")
        }
    }
    
    private func getVibemanExecutablePath() -> URL? {
        // First try to find the executable relative to the app bundle
        let bundlePath = Bundle.main.bundlePath
        if !bundlePath.isEmpty {
            let executablePath = URL(fileURLWithPath: bundlePath)
                .appendingPathComponent("Contents")
                .appendingPathComponent("Resources")
                .appendingPathComponent("vibeman")
            
            if FileManager.default.fileExists(atPath: executablePath.path) {
                return executablePath
            }
        }
        
        // For development: look for vibeman relative to the current executable
        if let executablePath = Bundle.main.executablePath {
            let executableDir = URL(fileURLWithPath: executablePath).deletingLastPathComponent()
            
            // Try various relative paths from the executable location
            let relativePaths = [
                "../../../../vibeman",    // From .build/x86_64-apple-macosx/debug/Vibeman to project root
                "../../../vibeman",      // From .build/debug/Vibeman to project root
                "../../vibeman",         // Alternative relative path
                "../vibeman",            // Another alternative
                "vibeman"                // Same directory
            ]
            
            for relativePath in relativePaths {
                let vibemanPath = executableDir.appendingPathComponent(relativePath).standardized
                logger.info("Checking for vibeman at: \(vibemanPath.path)")
                if FileManager.default.fileExists(atPath: vibemanPath.path) {
                    logger.info("Found vibeman executable at: \(vibemanPath.path)")
                    return vibemanPath
                }
            }
        }
        
        // Try current working directory
        let currentDir = FileManager.default.currentDirectoryPath
        let vibemanPath = URL(fileURLWithPath: currentDir).appendingPathComponent("vibeman")
        logger.info("Checking current directory for vibeman at: \(vibemanPath.path)")
        if FileManager.default.fileExists(atPath: vibemanPath.path) {
            logger.info("Found vibeman in current directory at: \(vibemanPath.path)")
            return vibemanPath
        }
        
        // Try parent directory explicitly
        let parentPath = URL(fileURLWithPath: currentDir).appendingPathComponent("../vibeman").standardized
        logger.info("Checking parent directory for vibeman at: \(parentPath.path)")
        if FileManager.default.fileExists(atPath: parentPath.path) {
            logger.info("Found vibeman in parent directory at: \(parentPath.path)")
            return parentPath
        }
        
        // Fallback to system PATH
        let task = Process()
        task.launchPath = "/usr/bin/which"
        task.arguments = ["vibeman"]
        
        let pipe = Pipe()
        task.standardOutput = pipe
        
        do {
            try task.run()
            task.waitUntilExit()
            
            if task.terminationStatus == 0 {
                let data = pipe.fileHandleForReading.readDataToEndOfFile()
                if let path = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines) {
                    return URL(fileURLWithPath: path)
                }
            }
        } catch {
            logger.error("Failed to locate vibeman executable: \(error.localizedDescription)")
        }
        
        logger.error("Could not find vibeman executable in any location")
        return nil
    }
    
    private func getWebAppPath() -> String {
        // First try to find the web app relative to the app bundle
        let bundlePath = Bundle.main.bundlePath
        if !bundlePath.isEmpty {
            let webAppPath = URL(fileURLWithPath: bundlePath)
                .appendingPathComponent("Contents")
                .appendingPathComponent("Resources")
                .appendingPathComponent("vibeman-web")
                .path
            
            if FileManager.default.fileExists(atPath: webAppPath) {
                return webAppPath
            }
        }
        
        // For development: look for vibeman-web relative to the current executable
        if let executablePath = Bundle.main.executablePath {
            let executableDir = URL(fileURLWithPath: executablePath).deletingLastPathComponent()
            
            // Try various relative paths from the executable location
            let relativePaths = [
                "../../../../vibeman-web",  // From .build/x86_64-apple-macosx/debug/Vibeman
                "../../../vibeman-web",     // From .build/debug/Vibeman
                "../../vibeman-web",        // Alternative relative path
                "../vibeman-web",           // Another alternative
                "vibeman-web"               // Same directory
            ]
            
            for relativePath in relativePaths {
                let webAppPath = executableDir.appendingPathComponent(relativePath).standardized.path
                if FileManager.default.fileExists(atPath: webAppPath) {
                    logger.info("Found vibeman-web at: \(webAppPath)")
                    return webAppPath
                }
            }
        }
        
        // Last resort: check current working directory
        let currentDir = FileManager.default.currentDirectoryPath
        let webAppPath = URL(fileURLWithPath: currentDir).appendingPathComponent("vibeman-web").path
        if FileManager.default.fileExists(atPath: webAppPath) {
            return webAppPath
        }
        
        logger.error("Could not find vibeman-web directory")
        showError("Could not find vibeman-web directory. Please ensure the web UI is built.")
        return currentDir
    }
    
    private func setupSparkle() {
        // Initialize Sparkle for auto-updates using the delegate pattern
        let updaterController = SPUStandardUpdaterController(startingUpdater: true, updaterDelegate: self, userDriverDelegate: nil)
        
        // Enable automatic checks
        updaterController.updater.automaticallyChecksForUpdates = true
        updaterController.updater.updateCheckInterval = 86400 // 24 hours
        
        // Store reference to prevent deallocation
        objc_setAssociatedObject(self, "updater", updaterController, .OBJC_ASSOCIATION_RETAIN_NONATOMIC)
        logger.info("Sparkle updater initialized with delegate")
    }
    
    // MARK: - SPUUpdaterDelegate
    
    nonisolated func feedURLString(for updater: SPUUpdater) -> String? {
        // GitHub repository-hosted appcast
        let feedURL = "https://raw.githubusercontent.com/mazdak/vibeman/main/appcast.xml"
        return feedURL
    }
    
    @objc func toggleMainWindow() {
        if mainWindow == nil {
            createMainWindow()
        }
        windowController.toggleMainWindow(mainWindow)
    }
    
    private func createMainWindow() {
        // Create the main window with WebView
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 1200, height: 800),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        
        window.title = "Vibeman"
        window.center()
        window.setContentSize(NSSize(width: 1200, height: 800))
        window.minSize = NSSize(width: 1024, height: 768)
        
        // Use enhanced WebViewController instead of basic WebView
        let webViewController = WebViewController()
        window.contentViewController = webViewController
        
        // Configure window delegate for cleanup
        mainWindowDelegate = MainWindowDelegate { [weak self] in
            self?.onMainWindowClosed()
        }
        window.delegate = mainWindowDelegate
        
        mainWindow = window
    }
    
    private func onMainWindowClosed() {
        logger.info("Main window closed - cleaning up references but keeping app running")
        mainWindow = nil
        mainWindowDelegate = nil
    }
    
    @objc func openPreferences() {
        // Open main window if not already open
        if mainWindow == nil {
            createMainWindow()
        }
        
        // Show window and navigate to settings
        windowController.toggleMainWindow(mainWindow)
        
        // Navigate to settings page
        if let webViewController = mainWindow?.contentViewController as? WebViewController {
            webViewController.navigateToPath("/settings")
        }
    }
    
    @objc func installCLI() {
        // Use the CLIInstaller to handle installation
        cliInstaller.installCLI { [weak self] result in
            switch result {
            case .success:
                self?.showSuccess("The vibeman CLI tool has been installed successfully. You can now use 'vibeman' from the terminal.")
            case .failed(let error):
                self?.showError("Failed to install CLI: \(error.localizedDescription)")
            case .cancelled:
                self?.logger.info("CLI installation was cancelled")
            }
        }
    }
    
    @objc func checkForUpdates() {
        // Trigger Sparkle update check
        if let updater = objc_getAssociatedObject(self, "updater") as? SPUStandardUpdaterController {
            logger.info("Triggering Sparkle update check")
            updater.updater.checkForUpdates()
        } else {
            logger.error("Sparkle updater not found - this should not happen")
            // Fallback: show a simple dialog if Sparkle isn't working
            let alert = NSAlert()
            alert.messageText = "Update Check Error"
            alert.informativeText = "The update checker is not properly initialized. Please restart the app and try again."
            alert.alertStyle = .warning
            alert.runModal()
        }
    }
    
    private func showError(_ message: String) {
        let alert = NSAlert()
        alert.messageText = "Error"
        alert.informativeText = message
        alert.alertStyle = .critical
        alert.runModal()
    }
    
    private func showSuccess(_ message: String) {
        let alert = NSAlert()
        alert.messageText = "Success"
        alert.informativeText = message
        alert.alertStyle = .informational
        alert.runModal()
    }
    
    private func checkServerHealth() {
        guard let basePort = vibemanConfig?.server.basePort else { return }
        
        let url = URL(string: "http://localhost:\(basePort)/api/health")!
        let task = URLSession.shared.dataTask(with: url) { [weak self] data, response, error in
            DispatchQueue.main.async {
                if let error = error {
                    self?.logger.error("Server health check failed: \(error.localizedDescription)")
                    self?.updateStatusItem(healthy: false)
                    return
                }
                
                if let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 {
                    self?.logger.info("Server health check passed")
                    self?.updateStatusItem(healthy: true)
                } else {
                    self?.logger.warning("Server health check returned non-200 status")
                    self?.updateStatusItem(healthy: false)
                }
            }
        }
        task.resume()
    }
    
    private func updateStatusItem(healthy: Bool) {
        guard let button = statusItem?.button else { return }
        
        if healthy {
            button.toolTip = "Vibeman - Server Running"
            // Keep the existing icon but you could add a green dot overlay
        } else {
            button.toolTip = "Vibeman - Server Not Responding"
            // Keep the existing icon but you could add a red dot overlay
        }
    }
    
    // MARK: - Login Items Management
    
    private func setupLoginItemsMenu(_ menu: NSMenu) {
        logger.info("Setting up login items menu")
        loginItemMenuItem = NSMenuItem(title: "Start at Login", action: #selector(toggleLoginItems), keyEquivalent: "")
        loginItemMenuItem?.target = self
        menu.addItem(loginItemMenuItem!)
        logger.info("Added login items menu item to menu")
        
        // Update the menu item state
        updateLoginItemsMenuState()
    }
    
    private func updateLoginItemsMenuState() {
        guard let menuItem = loginItemMenuItem else { return }
        
        let startAtLogin = UserDefaults.standard.bool(forKey: "startAtLogin")
        menuItem.state = startAtLogin ? .on : .off
    }
    
    @objc private func toggleLoginItems() {
        if #available(macOS 13.0, *) {
            let currentState = UserDefaults.standard.bool(forKey: "startAtLogin")
            let newState = !currentState
            
            // Update UserDefaults first
            UserDefaults.standard.set(newState, forKey: "startAtLogin")
            
            // Update the login item registration
            do {
                if newState {
                    try SMAppService.mainApp.register()
                    logger.info("Successfully registered login item")
                } else {
                    try SMAppService.mainApp.unregister()
                    logger.info("Successfully unregistered login item")
                }
                
                // Update menu state
                updateLoginItemsMenuState()
                
            } catch {
                logger.error("Failed to update login item: \(error.localizedDescription)")
                
                // Revert UserDefaults on failure
                UserDefaults.standard.set(currentState, forKey: "startAtLogin")
                
                showLoginItemsError("Failed to update login item: \(error.localizedDescription)")
            }
        } else {
            showLoginItemsError("Login items require macOS 13.0 or later")
        }
    }
    
    private func showLoginItemsError(_ message: String) {
        let alert = NSAlert()
        alert.messageText = "Login Items Error"
        alert.informativeText = message
        alert.alertStyle = .warning
        alert.runModal()
    }
    
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        logger.info("applicationShouldTerminateAfterLastWindowClosed called - returning false to keep app running")
        return false // Keep app running in menu bar
    }
    
    func applicationWillTerminate(_ notification: Notification) {
        // Clean up server processes gracefully
        if let serverProcess = serverProcess, serverProcess.isRunning {
            // Try graceful shutdown first
            let shutdownProcess = Process()
            shutdownProcess.launchPath = serverProcess.launchPath
            shutdownProcess.arguments = ["server", "stop"]
            
            do {
                try shutdownProcess.run()
                shutdownProcess.waitUntilExit()
                logger.info("Gracefully stopped vibeman server")
            } catch {
                logger.error("Failed to gracefully stop server, terminating: \(error.localizedDescription)")
                serverProcess.terminate()
            }
        }
        
        if let webProcess = webServerProcess, webProcess.isRunning {
            webProcess.terminate()
            logger.info("Stopped web server process")
        }
        
        logger.info("Vibeman app terminated")
    }
}

// Window controller for managing window display
class WindowController {
    private var previousApp: NSRunningApplication?
    
    func toggleMainWindow(_ window: NSWindow?) {
        guard let window = window else { return }
        
        if window.isVisible {
            // Store the currently active app before hiding
            previousApp = NSWorkspace.shared.frontmostApplication
            window.orderOut(nil)
        } else {
            // Show and activate window
            window.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
        }
    }
    
    func restoreFocusToPreviousApp() {
        previousApp?.activate()
        previousApp = nil
    }
}

// Window delegate for cleanup
private class MainWindowDelegate: NSObject, NSWindowDelegate {
    private let onClose: () -> Void
    
    init(onClose: @escaping () -> Void) {
        self.onClose = onClose
        super.init()
    }
    
    func windowWillClose(_ notification: Notification) {
        onClose()
    }
}