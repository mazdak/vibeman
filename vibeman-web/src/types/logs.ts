// Log viewer types for Vibeman
export interface LogEntry {
  id: string;
  timestamp: string;
  level: LogLevel;
  message: string;
  source?: string;
  container?: string;
  metadata?: Record<string, any>;
}

export type LogLevel = 'trace' | 'debug' | 'info' | 'warn' | 'error' | 'fatal';

export interface LogFilter {
  levels: LogLevel[];
  search: string;
  source?: string;
  container?: string;
  startTime?: string;
  endTime?: string;
}

// WebSocket log message types
export interface LogMessage {
  type: 'log' | 'error' | 'connect' | 'disconnect' | 'clear';
  data?: LogEntry;
  error?: string;
}

export interface LogWebSocketMessage extends LogMessage {
  type: 'log';
  data: LogEntry;
}

export interface LogErrorMessage extends LogMessage {
  type: 'error';
  error: string;
}

export interface LogConnectMessage extends LogMessage {
  type: 'connect';
}

export interface LogDisconnectMessage extends LogMessage {
  type: 'disconnect';
}

export interface LogClearMessage extends LogMessage {
  type: 'clear';
}

export type LogWebSocketMessageType = 
  | LogWebSocketMessage 
  | LogErrorMessage 
  | LogConnectMessage 
  | LogDisconnectMessage
  | LogClearMessage;

export interface LogState {
  isConnected: boolean;
  isConnecting: boolean;
  error: string | null;
  environmentId: string | null;
  logs: LogEntry[];
  filteredLogs: LogEntry[];
  filter: LogFilter;
  isAutoScroll: boolean;
  lastLogCount: number;
}

export interface LogViewerProps {
  environmentId: string;
  className?: string;
  height?: string | number;
  showToolbar?: boolean;
  autoConnect?: boolean;
  maxLogs?: number;
}

// Log level configurations
export const LOG_LEVEL_CONFIG: Record<LogLevel, {
  color: string;
  bgColor: string;
  darkColor: string;
  darkBgColor: string;
  priority: number;
}> = {
  trace: {
    color: 'text-gray-600',
    bgColor: 'bg-gray-100',
    darkColor: 'dark:text-gray-400',
    darkBgColor: 'dark:bg-gray-900/50',
    priority: 0
  },
  debug: {
    color: 'text-blue-600',
    bgColor: 'bg-blue-100',
    darkColor: 'dark:text-blue-400',
    darkBgColor: 'dark:bg-blue-900/20',
    priority: 1
  },
  info: {
    color: 'text-green-600',
    bgColor: 'bg-green-100',
    darkColor: 'dark:text-green-400',
    darkBgColor: 'dark:bg-green-900/20',
    priority: 2
  },
  warn: {
    color: 'text-yellow-600',
    bgColor: 'bg-yellow-100',
    darkColor: 'dark:text-yellow-400',
    darkBgColor: 'dark:bg-yellow-900/20',
    priority: 3
  },
  error: {
    color: 'text-red-600',
    bgColor: 'bg-red-100',
    darkColor: 'dark:text-red-400',
    darkBgColor: 'dark:bg-red-900/20',
    priority: 4
  },
  fatal: {
    color: 'text-red-800',
    bgColor: 'bg-red-200',
    darkColor: 'dark:text-red-300',
    darkBgColor: 'dark:bg-red-900/40',
    priority: 5
  }
};

export const LOG_LEVELS: LogLevel[] = ['trace', 'debug', 'info', 'warn', 'error', 'fatal'];

// Default filter
export const DEFAULT_LOG_FILTER: LogFilter = {
  levels: ['info', 'warn', 'error', 'fatal'],
  search: '',
  source: undefined,
  container: undefined,
  startTime: undefined,
  endTime: undefined
};