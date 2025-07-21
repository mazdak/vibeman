import { readFileSync, existsSync } from 'fs';
import { homedir } from 'os';
import { join } from 'path';

interface ServerConfig {
  port: number;
}

interface VibemanConfig {
  server: ServerConfig;
}

function loadConfig(): VibemanConfig {
  const configPath = join(homedir(), '.config', 'vibeman', 'config.toml');
  
  // Default config
  const defaultConfig: VibemanConfig = {
    server: {
      port: 8080
    }
  };
  
  if (!existsSync(configPath)) {
    return defaultConfig;
  }
  
  try {
    const configContent = readFileSync(configPath, 'utf-8');
    
    // Simple TOML parsing for port
    const portMatch = configContent.match(/port\s*=\s*(\d+)/);
    if (portMatch) {
      defaultConfig.server.port = parseInt(portMatch[1], 10);
    }
    
    return defaultConfig;
  } catch (error) {
    console.error('Failed to load config:', error);
    return defaultConfig;
  }
}

export function getServerPorts() {
  const config = loadConfig();
  const basePort = config.server.port;
  
  return {
    backendPort: basePort,
    webUIPort: basePort + 1
  };
}