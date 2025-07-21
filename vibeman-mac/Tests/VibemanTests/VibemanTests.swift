import XCTest
@testable import Vibeman

final class VibemanTests: XCTestCase {
    
    func testAppLaunch() {
        // Basic test to ensure the app can initialize without crashing
        let app = VibemanApp()
        XCTAssertNotNil(app)
    }
    
    func testWindowController() {
        let controller = WindowController()
        XCTAssertNotNil(controller)
    }
    
    // Add more tests as needed
}