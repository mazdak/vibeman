import Cocoa
import WebKit
import UserNotifications
import os.log

/// Enhanced WebViewController with comprehensive WKWebView integration for Vibeman
public class WebViewController: NSViewController {
    // MARK: - Properties
    private var webView: WKWebView!
    private var progressIndicator: NSProgressIndicator!
    private var errorView: NSView!
    private var errorLabel: NSTextField!
    private var retryButton: NSButton!
    
    private var vibemanURL: String {
        // Load config to get the port
        if let config = try? VibemanConfigManager.load() {
            return "http://localhost:\(config.server.webUIPort)"
        }
        return "http://localhost:8081" // fallback
    }
    private var isLoading = false
    private let logger = Logger(subsystem: "com.vibeman.app", category: "WebViewController")
    
    // MARK: - Lifecycle
    public override func loadView() {
        view = NSView()
        setupWebView()
        setupErrorView()
        setupProgressIndicator()
        setupConstraints()
    }
    
    public override func viewDidLoad() {
        super.viewDidLoad()
        loadVibemanApp()
    }
    
    // MARK: - Setup Methods
    private func setupWebView() {
        let configuration = WKWebViewConfiguration()
        
        // Configure preferences for modern WebKit
        if #available(macOS 11.0, *) {
            let preferences = WKWebpagePreferences()
            preferences.allowsContentJavaScript = true
            configuration.defaultWebpagePreferences = preferences
        }
        
        // Enable developer extras in debug builds
        #if DEBUG
        configuration.preferences.setValue(true, forKey: "developerExtrasEnabled")
        #endif
        
        // Create webview
        webView = WKWebView(frame: .zero, configuration: configuration)
        webView.translatesAutoresizingMaskIntoConstraints = false
        webView.navigationDelegate = self
        webView.uiDelegate = self
        
        // Add JavaScript bridge
        setupJavaScriptBridge()
        
        view.addSubview(webView)
    }
    
    private func setupErrorView() {
        errorView = NSView()
        errorView.translatesAutoresizingMaskIntoConstraints = false
        errorView.isHidden = true
        
        errorLabel = NSTextField(labelWithString: "")
        errorLabel.translatesAutoresizingMaskIntoConstraints = false
        errorLabel.alignment = .center
        errorLabel.font = NSFont.systemFont(ofSize: 16)
        errorLabel.textColor = .secondaryLabelColor
        
        retryButton = NSButton(title: "Retry", target: self, action: #selector(retryConnection))
        retryButton.translatesAutoresizingMaskIntoConstraints = false
        retryButton.bezelStyle = .rounded
        
        errorView.addSubview(errorLabel)
        errorView.addSubview(retryButton)
        view.addSubview(errorView)
    }
    
    private func setupProgressIndicator() {
        progressIndicator = NSProgressIndicator()
        progressIndicator.translatesAutoresizingMaskIntoConstraints = false
        progressIndicator.style = .spinning
        progressIndicator.controlSize = .regular
        progressIndicator.isHidden = true
        
        view.addSubview(progressIndicator)
    }
    
    private func setupConstraints() {
        NSLayoutConstraint.activate([
            // WebView constraints
            webView.topAnchor.constraint(equalTo: view.topAnchor),
            webView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            webView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            webView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            
            // Error view constraints
            errorView.centerXAnchor.constraint(equalTo: view.centerXAnchor),
            errorView.centerYAnchor.constraint(equalTo: view.centerYAnchor),
            errorView.widthAnchor.constraint(lessThanOrEqualTo: view.widthAnchor, multiplier: 0.8),
            errorView.heightAnchor.constraint(lessThanOrEqualTo: view.heightAnchor, multiplier: 0.5),
            
            // Error label constraints
            errorLabel.topAnchor.constraint(equalTo: errorView.topAnchor),
            errorLabel.leadingAnchor.constraint(equalTo: errorView.leadingAnchor),
            errorLabel.trailingAnchor.constraint(equalTo: errorView.trailingAnchor),
            
            // Retry button constraints
            retryButton.topAnchor.constraint(equalTo: errorLabel.bottomAnchor, constant: 20),
            retryButton.centerXAnchor.constraint(equalTo: errorView.centerXAnchor),
            retryButton.bottomAnchor.constraint(equalTo: errorView.bottomAnchor),
            
            // Progress indicator constraints
            progressIndicator.centerXAnchor.constraint(equalTo: view.centerXAnchor),
            progressIndicator.centerYAnchor.constraint(equalTo: view.centerYAnchor)
        ])
    }
    
    private func setupJavaScriptBridge() {
        let contentController = webView.configuration.userContentController
        contentController.add(self, name: "vibemanBridge")
        
        // Inject JavaScript to create the bridge
        let bridgeScript = """
        window.vibemanBridge = {
            postMessage: function(message) {
                window.webkit.messageHandlers.vibemanBridge.postMessage(message);
            },
            
            // Helper methods for common operations
            openInFinder: function(path) {
                this.postMessage({type: 'openInFinder', path: path});
            },
            
            showNotification: function(title, body) {
                this.postMessage({type: 'notification', title: title, body: body});
            },
            
            requestPermission: function(permission) {
                this.postMessage({type: 'requestPermission', permission: permission});
            }
        };
        """
        
        let userScript = WKUserScript(source: bridgeScript, injectionTime: .atDocumentEnd, forMainFrameOnly: false)
        contentController.addUserScript(userScript)
    }
    
    // MARK: - Public Methods
    public func loadVibemanApp() {
        guard let url = URL(string: vibemanURL) else {
            showError("Invalid URL: \\(vibemanURL)")
            return
        }
        
        showLoading()
        let request = URLRequest(url: url)
        webView.load(request)
        logger.info("Loading Vibeman app from \\(self.vibemanURL)")
    }
    
    public func navigateToPath(_ path: String) {
        let fullURLString = vibemanURL + path
        guard let url = URL(string: fullURLString) else {
            logger.error("Invalid navigation URL: \\(fullURLString)")
            return
        }
        
        let request = URLRequest(url: url)
        webView.load(request)
        logger.info("Navigating to: \\(fullURLString)")
    }
    
    // MARK: - Private Methods
    private func showLoading() {
        isLoading = true
        progressIndicator.isHidden = false
        progressIndicator.startAnimation(nil)
        errorView.isHidden = true
        webView.isHidden = false
    }
    
    private func hideLoading() {
        isLoading = false
        progressIndicator.isHidden = true
        progressIndicator.stopAnimation(nil)
    }
    
    private func showError(_ message: String) {
        hideLoading()
        errorLabel.stringValue = message
        errorView.isHidden = false
        webView.isHidden = true
        logger.error("WebView error: \\(message)")
    }
    
    private func hideError() {
        errorView.isHidden = true
        webView.isHidden = false
    }
    
    @objc private func retryConnection() {
        logger.info("Retrying connection to Vibeman server")
        loadVibemanApp()
    }
}

// MARK: - WKNavigationDelegate
extension WebViewController: WKNavigationDelegate {
    public func webView(_ webView: WKWebView, didStartProvisionalNavigation navigation: WKNavigation!) {
        showLoading()
    }
    
    public func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        hideLoading()
        hideError()
        logger.info("Successfully loaded Vibeman web UI")
    }
    
    public func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        hideLoading()
        let errorMessage = "Failed to load Vibeman UI\\n\\n\\(error.localizedDescription)\\n\\nMake sure the Vibeman server is running on \\(vibemanURL)"
        showError(errorMessage)
    }
    
    public func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
        hideLoading()
        let errorMessage = "Cannot connect to Vibeman server\\n\\n\\(error.localizedDescription)\\n\\nMake sure the server is running on \\(vibemanURL)"
        showError(errorMessage)
    }
    
    public func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction, decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
        // Allow navigation to localhost and file URLs
        if let url = navigationAction.request.url {
            if url.isFileURL || url.host == "localhost" || url.host == "127.0.0.1" {
                decisionHandler(.allow)
                return
            }
            
            // For external URLs, open in default browser
            NSWorkspace.shared.open(url)
            decisionHandler(.cancel)
            return
        }
        
        decisionHandler(.allow)
    }
}

// MARK: - WKUIDelegate
extension WebViewController: WKUIDelegate {
    public func webView(_ webView: WKWebView, createWebViewWith configuration: WKWebViewConfiguration, for navigationAction: WKNavigationAction, windowFeatures: WKWindowFeatures) -> WKWebView? {
        // Handle popup windows by opening in default browser
        if let url = navigationAction.request.url {
            NSWorkspace.shared.open(url)
        }
        return nil
    }
    
    public func webView(_ webView: WKWebView, runJavaScriptAlertPanelWithMessage message: String, initiatedByFrame frame: WKFrameInfo, completionHandler: @escaping () -> Void) {
        let alert = NSAlert()
        alert.messageText = "Vibeman"
        alert.informativeText = message
        alert.addButton(withTitle: "OK")
        alert.runModal()
        completionHandler()
    }
    
    public func webView(_ webView: WKWebView, runJavaScriptConfirmPanelWithMessage message: String, initiatedByFrame frame: WKFrameInfo, completionHandler: @escaping (Bool) -> Void) {
        let alert = NSAlert()
        alert.messageText = "Vibeman"
        alert.informativeText = message
        alert.addButton(withTitle: "OK")
        alert.addButton(withTitle: "Cancel")
        let response = alert.runModal()
        completionHandler(response == .alertFirstButtonReturn)
    }
}

// MARK: - WKScriptMessageHandler
extension WebViewController: WKScriptMessageHandler {
    public func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
        guard message.name == "vibemanBridge",
              let messageBody = message.body as? [String: Any],
              let type = messageBody["type"] as? String else {
            return
        }
        
        logger.info("Received JavaScript bridge message: \\(type)")
        
        switch type {
        case "openInFinder":
            if let path = messageBody["path"] as? String {
                openInFinder(path: path)
            }
            
        case "notification":
            if let title = messageBody["title"] as? String,
               let body = messageBody["body"] as? String {
                showNotification(title: title, body: body)
            }
            
        case "requestPermission":
            if let permission = messageBody["permission"] as? String {
                handlePermissionRequest(permission)
            }
            
        default:
            logger.warning("Unknown bridge message type: \\(type)")
        }
    }
    
    private func openInFinder(path: String) {
        let url = URL(fileURLWithPath: path)
        NSWorkspace.shared.selectFile(nil, inFileViewerRootedAtPath: url.path)
        logger.info("Opened path in Finder: \\(path)")
    }
    
    private func showNotification(title: String, body: String) {
        if #available(macOS 10.14, *) {
            // Use modern UserNotifications framework
            let content = UNMutableNotificationContent()
            content.title = title
            content.body = body
            content.sound = .default
            
            let request = UNNotificationRequest(identifier: UUID().uuidString, content: content, trigger: nil)
            UNUserNotificationCenter.current().add(request) { error in
                if let error = error {
                    self.logger.error("Failed to deliver notification: \\(error.localizedDescription)")
                } else {
                    self.logger.info("Delivered notification: \\(title)")
                }
            }
        } else {
            // Fallback for older macOS versions
            let notification = NSUserNotification()
            notification.title = title
            notification.informativeText = body
            notification.soundName = NSUserNotificationDefaultSoundName
            NSUserNotificationCenter.default.deliver(notification)
            logger.info("Delivered notification: \\(title)")
        }
    }
    
    private func handlePermissionRequest(_ permission: String) {
        // Handle different permission requests
        switch permission {
        case "notifications":
            if #available(macOS 10.14, *) {
                UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge]) { granted, error in
                    DispatchQueue.main.async {
                        let script = "window.vibemanBridge.onPermissionResult && window.vibemanBridge.onPermissionResult('notifications', \\(granted))"
                        self.webView.evaluateJavaScript(script, completionHandler: nil)
                        self.logger.info("Notification permission granted: \\(granted)")
                    }
                }
            } else {
                // For older macOS versions, notifications are available by default
                DispatchQueue.main.async {
                    let script = "window.vibemanBridge.onPermissionResult && window.vibemanBridge.onPermissionResult('notifications', true)"
                    self.webView.evaluateJavaScript(script, completionHandler: nil)
                    self.logger.info("Notification permission granted: true (legacy)")
                }
            }
        default:
            logger.warning("Unknown permission request: \\(permission)")
        }
    }
}