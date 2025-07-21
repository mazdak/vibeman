import XCTest
@testable import VibemanKit

@available(macOS 13.0, *)
final class LoginItemsManagerTests: XCTestCase {
    var loginItemsManager: LoginItemsManager!
    
    override func setUpWithError() throws {
        try super.setUpWithError()
        loginItemsManager = LoginItemsManager(serviceIdentifier: "com.vibeman.app.helper.test")
    }
    
    override func tearDownWithError() throws {
        loginItemsManager = nil
        try super.tearDownWithError()
    }
    
    func testInitialization() {
        // Test that the manager initializes correctly
        XCTAssertNotNil(loginItemsManager)
        XCTAssertFalse(loginItemsManager.isEnabled) // Should start as false
    }
    
    func testStatusCheck() {
        // Test status checking functionality
        loginItemsManager.checkCurrentStatus()
        
        // Note: In a test environment, the login item won't be found
        // This is expected behavior
        XCTAssertFalse(loginItemsManager.isAvailable)
    }
    
    func testErrorHandling() async {
        // Test that attempting to enable in test environment handles errors gracefully
        let result = await loginItemsManager.enable()
        
        // Should fail in test environment but not crash
        XCTAssertFalse(result)
        XCTAssertNotNil(loginItemsManager.lastError)
    }
    
    func testToggleOperation() async {
        // Test toggle functionality
        let initialState = loginItemsManager.isEnabled
        let result = await loginItemsManager.toggle()
        
        // In test environment, should fail but not crash
        XCTAssertFalse(result)
    }
    
    func testErrorMessages() {
        // Test error message generation
        XCTAssertNil(loginItemsManager.getErrorMessage()) // No error initially
        
        // Create an error state and verify message is generated
        loginItemsManager.lastError = .notSupported
        let message = loginItemsManager.getErrorMessage()
        XCTAssertNotNil(message)
        XCTAssertTrue(message!.contains("not supported"))
    }
}

// Legacy manager tests for older macOS versions
final class LegacyLoginItemsManagerTests: XCTestCase {
    var legacyManager: LegacyLoginItemsManager!
    
    override func setUpWithError() throws {
        try super.setUpWithError()
        legacyManager = LegacyLoginItemsManager(helperBundleIdentifier: "com.vibeman.app.helper.test")
    }
    
    override func tearDownWithError() throws {
        legacyManager = nil
        try super.tearDownWithError()
    }
    
    func testLegacyInitialization() {
        XCTAssertNotNil(legacyManager)
    }
    
    func testLegacyOperations() {
        // Test that legacy operations don't crash
        XCTAssertFalse(legacyManager.isEnabled())
        XCTAssertFalse(legacyManager.enable())
        XCTAssertFalse(legacyManager.disable())
    }
}