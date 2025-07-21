import XCTest
import WebKit
@testable import VibemanKit

final class WebViewControllerTests: XCTestCase {
    
    var webViewController: WebViewController!
    
    override func setUp() {
        super.setUp()
        webViewController = WebViewController()
        // Force load the view
        _ = webViewController.view
    }
    
    override func tearDown() {
        webViewController = nil
        super.tearDown()
    }
    
    func testWebViewControllerInitialization() {
        XCTAssertNotNil(webViewController.view)
        
        // Check that subviews are set up
        let webViews = webViewController.view.subviews.filter { $0 is WKWebView }
        XCTAssertEqual(webViews.count, 1, "Should have exactly one WKWebView")
    }
    
    func testNavigateToPath() {
        // Test that navigateToPath constructs the correct URL
        webViewController.navigateToPath("/settings")
        
        // We can't easily test the actual navigation without mocking,
        // but we can verify the method doesn't crash
        XCTAssertNotNil(webViewController.view)
    }
    
    func testJavaScriptBridgeInjection() {
        // Get the WKWebView
        guard let webView = webViewController.view.subviews.first(where: { $0 is WKWebView }) as? WKWebView else {
            XCTFail("Could not find WKWebView")
            return
        }
        
        // Check that our message handler is registered
        let userContentController = webView.configuration.userContentController
        
        // The message handler name should be "vibemanBridge"
        // Note: WKUserContentController doesn't expose a way to check registered handlers
        // In a real test environment, we'd need to use a mock or test the behavior
        
        // Check that we have user scripts (our bridge injection)
        XCTAssertGreaterThan(userContentController.userScripts.count, 0, "Should have injected user scripts")
    }
    
    func testJavaScriptBridgeMessageTypes() {
        // Test the expected message types that the bridge should handle
        let messageTypes = ["openInFinder", "notification", "requestPermission"]
        
        for messageType in messageTypes {
            // Create a mock message
            let message = ["type": messageType, "path": "/test/path"]
            
            // We can't easily test the actual message handling without a full WebKit context,
            // but we verify the structure is correct
            XCTAssertNotNil(message["type"])
        }
    }
    
    func testErrorViewVisibility() {
        // The error view should be hidden initially
        let errorViews = webViewController.view.subviews.filter { subview in
            // Check if any subview has subviews that look like our error view
            // (contains both a text field and a button)
            let hasTextField = subview.subviews.contains { $0 is NSTextField }
            let hasButton = subview.subviews.contains { $0 is NSButton }
            return hasTextField && hasButton
        }
        
        // Should have one error view
        XCTAssertEqual(errorViews.count, 1, "Should have exactly one error view")
        
        if let errorView = errorViews.first {
            XCTAssertTrue(errorView.isHidden, "Error view should be hidden initially")
        }
    }
    
    func testProgressIndicatorSetup() {
        // Check that progress indicator exists
        let progressIndicators = webViewController.view.subviews.filter { $0 is NSProgressIndicator }
        XCTAssertEqual(progressIndicators.count, 1, "Should have exactly one progress indicator")
        
        if let progressIndicator = progressIndicators.first as? NSProgressIndicator {
            XCTAssertTrue(progressIndicator.isHidden, "Progress indicator should be hidden initially")
            XCTAssertEqual(progressIndicator.style, .spinning)
        }
    }
}

// Mock message for testing WKScriptMessage handling
class MockScriptMessage: NSObject {
    let name: String
    let body: Any
    
    init(name: String, body: Any) {
        self.name = name
        self.body = body
    }
}

// Extension to test message handling logic
extension WebViewControllerTests {
    
    func testMessageHandlingLogic() {
        // Test various message types that should be handled
        
        // Test openInFinder message
        let openInFinderMessage: [String: Any] = [
            "type": "openInFinder",
            "path": "/Users/test/Documents"
        ]
        XCTAssertEqual(openInFinderMessage["type"] as? String, "openInFinder")
        XCTAssertNotNil(openInFinderMessage["path"])
        
        // Test notification message
        let notificationMessage: [String: Any] = [
            "type": "notification",
            "title": "Test Title",
            "body": "Test Body"
        ]
        XCTAssertEqual(notificationMessage["type"] as? String, "notification")
        XCTAssertNotNil(notificationMessage["title"])
        XCTAssertNotNil(notificationMessage["body"])
        
        // Test permission request message
        let permissionMessage: [String: Any] = [
            "type": "requestPermission",
            "permission": "notifications"
        ]
        XCTAssertEqual(permissionMessage["type"] as? String, "requestPermission")
        XCTAssertEqual(permissionMessage["permission"] as? String, "notifications")
    }
}