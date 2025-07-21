import Foundation
import TOMLKit

/// Represents the global vibeman configuration
public struct VibemanConfig: Codable {
    public var server: ServerConfig
    public var storage: StorageConfig
    public var services: ServicesConfig?
    
    public struct ServerConfig: Codable {
        public var port: Int
        public var webUIPort: Int
        
        enum CodingKeys: String, CodingKey {
            case port
            case webUIPort = "webui_port"
        }
        
        // For backward compatibility
        public var basePort: Int {
            return port
        }
    }
    
    public struct StorageConfig: Codable {
        public var repositoriesPath: String
        public var worktreesPath: String
        
        enum CodingKeys: String, CodingKey {
            case repositoriesPath = "repositories_path"
            case worktreesPath = "worktrees_path"
        }
    }
    
    public struct ServicesConfig: Codable {
        public var configPath: String?
        
        enum CodingKeys: String, CodingKey {
            case configPath = "config_path"
        }
    }
}

/// Manager for reading and writing Vibeman configuration
public class VibemanConfigManager {
    private static let configDir = "~/.config/vibeman".expandingTildeInPath
    private static let configPath = "\(configDir)/config.toml"
    
    /// Default configuration values matching Go defaults
    public static func defaultConfig() -> VibemanConfig {
        return VibemanConfig(
            server: VibemanConfig.ServerConfig(
                port: 8080,
                webUIPort: 8081
            ),
            storage: VibemanConfig.StorageConfig(
                repositoriesPath: "~/vibeman/repos",
                worktreesPath: "~/vibeman/worktrees"
            ),
            services: VibemanConfig.ServicesConfig(
                configPath: nil
            )
        )
    }
    
    /// Load configuration from disk using proper TOML parsing
    public static func load() throws -> VibemanConfig {
        let fileManager = FileManager.default
        
        // Create config directory if it doesn't exist
        if !fileManager.fileExists(atPath: configDir) {
            try fileManager.createDirectory(atPath: configDir, withIntermediateDirectories: true)
        }
        
        // If config file doesn't exist, return defaults
        if !fileManager.fileExists(atPath: configPath) {
            return defaultConfig()
        }
        
        // Read and parse the TOML file
        let configData = try Data(contentsOf: URL(fileURLWithPath: configPath))
        let tomlString = String(data: configData, encoding: .utf8) ?? ""
        
        do {
            let toml = try TOMLTable(string: tomlString)
            var config = defaultConfig()
            
            // Parse server section
            if let serverTable = toml["server"]?.table {
                if let port = serverTable["port"]?.int {
                    config.server.port = port
                }
                if let webUIPort = serverTable["webui_port"]?.int {
                    config.server.webUIPort = webUIPort
                }
            }
            
            // Parse storage section
            if let storageTable = toml["storage"]?.table {
                if let reposPath = storageTable["repositories_path"]?.string {
                    config.storage.repositoriesPath = reposPath
                }
                if let worktreesPath = storageTable["worktrees_path"]?.string {
                    config.storage.worktreesPath = worktreesPath
                }
            }
            
            // Parse services section
            if let servicesTable = toml["services"]?.table {
                config.services = VibemanConfig.ServicesConfig()
                if let configPath = servicesTable["config_path"]?.string {
                    config.services?.configPath = configPath
                }
            }
            
            return config
            
        } catch {
            // If TOML parsing fails, return defaults
            print("Failed to parse TOML config: \(error)")
            return defaultConfig()
        }
    }
    
    /// Save configuration to disk
    public static func save(_ config: VibemanConfig) throws {
        let fileManager = FileManager.default
        
        // Create config directory if it doesn't exist
        if !fileManager.fileExists(atPath: configDir) {
            try fileManager.createDirectory(atPath: configDir, withIntermediateDirectories: true)
        }
        
        // Create TOML table
        var toml = TOMLTable()
        
        // Server section
        var serverTable = TOMLTable()
        serverTable["port"] = config.server.port
        serverTable["webui_port"] = config.server.webUIPort
        toml["server"] = .table(serverTable)
        
        // Storage section
        var storageTable = TOMLTable()
        storageTable["repositories_path"] = config.storage.repositoriesPath
        storageTable["worktrees_path"] = config.storage.worktreesPath
        toml["storage"] = .table(storageTable)
        
        // Services section (if present)
        if let services = config.services {
            var servicesTable = TOMLTable()
            if let configPath = services.configPath {
                servicesTable["config_path"] = configPath
            }
            toml["services"] = .table(servicesTable)
        }
        
        // Convert to string and write
        let tomlString = toml.string
        try tomlString.write(toFile: configPath, atomically: true, encoding: .utf8)
    }
}

// Helper extension for tilde expansion
private extension String {
    var expandingTildeInPath: String {
        return (self as NSString).expandingTildeInPath
    }
}