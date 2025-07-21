import { useCallback, useEffect, useState } from 'react';
// WebSocket URL helper - will use direct URL construction since it's just for WebSocket connections
import { useWebSocket, useMounted } from '../shared/hooks';
import { downloadJSON, downloadText, generateTimestampedFilename } from '../shared/utils';
import type { 
  LogState, 
  LogEntry, 
  LogFilter, 
  LogWebSocketMessageType
} from '../types/logs';
import { DEFAULT_LOG_FILTER } from '../types/logs';

export interface UseLogsOptions {
  worktreeId: string;
  onLogEntry?: (log: LogEntry) => void;
  onError?: (error: string) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  autoConnect?: boolean;
  maxLogs?: number;
  filter?: Partial<LogFilter>;
}

export interface UseLogsReturn {
  state: LogState;
  connect: () => void;
  disconnect: () => void;
  clearLogs: () => void;
  exportLogs: (format?: 'txt' | 'json') => void;
  updateFilter: (filter: Partial<LogFilter>) => void;
  toggleAutoScroll: () => void;
  isReady: boolean;
}

const DEFAULT_MAX_LOGS = 1000;

export function useLogs(options: UseLogsOptions): UseLogsReturn {
  const {
    worktreeId,
    onLogEntry,
    onError,
    onConnect,
    onDisconnect,
    autoConnect = false,
    maxLogs = DEFAULT_MAX_LOGS,
    filter: initialFilter = {}
  } = options;

  const mountedRef = useMounted();

  // Initialize filter with defaults
  const defaultFilter = { ...DEFAULT_LOG_FILTER, ...initialFilter };

  const [state, setState] = useState<LogState>({
    isConnected: false,
    isConnecting: false,
    error: null,
    worktreeId,
    logs: [],
    filteredLogs: [],
    filter: defaultFilter,
    isAutoScroll: true,
    lastLogCount: 0
  });

  // WebSocket message handler
  const handleMessage = useCallback((event: MessageEvent) => {
    try {
      const message: LogWebSocketMessageType = JSON.parse(event.data);
      
      switch (message.type) {
        case 'log':
          if (message.data) {
            const newLog = message.data;
            setState(prev => {
              const newLogs = [...prev.logs, newLog];
              
              // Trim logs if we exceed maxLogs
              const trimmedLogs = newLogs.length > maxLogs 
                ? newLogs.slice(-maxLogs) 
                : newLogs;
              
              return {
                ...prev,
                logs: trimmedLogs,
                lastLogCount: trimmedLogs.length
              };
            });
            onLogEntry?.(newLog);
          }
          break;
        case 'error':
          if (message.error) {
            onError?.(message.error);
            if (mountedRef.current) {
              setState(prev => ({
                ...prev,
                error: message.error || 'Log streaming error'
              }));
            }
          }
          break;
        case 'clear':
          if (mountedRef.current) {
            setState(prev => ({
              ...prev,
              logs: [],
              filteredLogs: [],
              lastLogCount: 0
            }));
          }
          break;
        case 'disconnect':
          webSocketControls.disconnect();
          break;
        default:
          // Unknown message type, ignore
      }
    } catch (error) {
      console.error('Failed to parse log message:', error);
      onError?.('Failed to parse log message');
    }
  }, [onLogEntry, onError, maxLogs, mountedRef]);

  // WebSocket handlers
  const handleOpen = useCallback(() => {
    if (mountedRef.current) {
      setState(prev => ({
        ...prev,
        isConnected: true,
        isConnecting: false,
        error: null
      }));
    }
    onConnect?.();
  }, [onConnect, mountedRef]);

  const handleClose = useCallback(() => {
    if (mountedRef.current) {
      setState(prev => ({
        ...prev,
        isConnected: false,
        isConnecting: false
      }));
    }
    onDisconnect?.();
  }, [onDisconnect, mountedRef]);

  const handleError = useCallback(() => {
    if (mountedRef.current) {
      setState(prev => ({
        ...prev,
        error: 'Connection failed',
        isConnecting: false,
        isConnected: false
      }));
    }
    onError?.('Connection failed');
  }, [onError, mountedRef]);

  // Construct WebSocket URL directly
  const getWebSocketUrl = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    return `${protocol}//${host}/api/worktrees/${worktreeId}/logs`;
  }, [worktreeId]);

  // Use shared WebSocket hook
  const webSocketControls = useWebSocket({
    url: getWebSocketUrl,
    onOpen: handleOpen,
    onMessage: handleMessage,
    onClose: handleClose,
    onError: handleError,
    autoConnect,
    reconnectAttempts: 5,
    reconnectDelay: 2000
  });

  // Update connection states from WebSocket hook
  useEffect(() => {
    if (mountedRef.current) {
      setState(prev => ({
        ...prev,
        isConnected: webSocketControls.state.isConnected,
        isConnecting: webSocketControls.state.isConnecting,
        error: webSocketControls.state.error || prev.error
      }));
    }
  }, [webSocketControls.state, mountedRef]);

  // Filter logs based on current filter settings
  const filterLogs = useCallback((logs: LogEntry[], filter: LogFilter): LogEntry[] => {
    return logs.filter(log => {
      // Level filter
      if (!filter.levels.includes(log.level)) {
        return false;
      }

      // Search filter
      if (filter.search && !log.message.toLowerCase().includes(filter.search.toLowerCase())) {
        return false;
      }

      // Source filter
      if (filter.source && log.source !== filter.source) {
        return false;
      }

      // Container filter
      if (filter.container && log.container !== filter.container) {
        return false;
      }

      // Time range filters
      if (filter.startTime && log.timestamp < filter.startTime) {
        return false;
      }

      if (filter.endTime && log.timestamp > filter.endTime) {
        return false;
      }

      return true;
    });
  }, []);

  // Update filtered logs when logs or filter change
  useEffect(() => {
    const filtered = filterLogs(state.logs, state.filter);
    setState(prev => ({
      ...prev,
      filteredLogs: filtered
    }));
  }, [state.logs, state.filter, filterLogs]);


  // Clear logs
  const clearLogs = useCallback(() => {
    setState(prev => ({
      ...prev,
      logs: [],
      filteredLogs: [],
      lastLogCount: 0
    }));
  }, []);

  // Export logs
  const exportLogs = useCallback((format: 'txt' | 'json' = 'txt') => {
    const logs = state.filteredLogs.length > 0 ? state.filteredLogs : state.logs;
    const filename = generateTimestampedFilename(`vibeman-logs-${worktreeId}`, format);

    if (format === 'json') {
      downloadJSON(logs, filename);
    } else {
      const content = logs.map(log => 
        `[${log.timestamp}] ${log.level.toUpperCase()}: ${log.message}${log.source ? ` (${log.source})` : ''}`
      ).join('\n');
      downloadText(content, filename);
    }
  }, [state.logs, state.filteredLogs, worktreeId]);

  // Update filter
  const updateFilter = useCallback((newFilter: Partial<LogFilter>) => {
    setState(prev => ({
      ...prev,
      filter: { ...prev.filter, ...newFilter }
    }));
  }, []);

  // Toggle auto-scroll
  const toggleAutoScroll = useCallback(() => {
    setState(prev => ({
      ...prev,
      isAutoScroll: !prev.isAutoScroll
    }));
  }, []);

  // Update worktree ID
  useEffect(() => {
    if (mountedRef.current) {
      setState(prev => ({
        ...prev,
        worktreeId
      }));
    }
  }, [worktreeId, mountedRef]);

  return {
    state,
    connect: webSocketControls.connect,
    disconnect: webSocketControls.disconnect,
    clearLogs,
    exportLogs,
    updateFilter,
    toggleAutoScroll,
    isReady: state.isConnected && !state.isConnecting
  };
}