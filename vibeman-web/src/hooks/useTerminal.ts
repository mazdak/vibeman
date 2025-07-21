import { useCallback, useEffect, useState } from 'react';
// WebSocket URL helper - will use direct URL construction since it's just for WebSocket connections
import { useWebSocket, useMounted } from '../shared/hooks';
import type { 
  TerminalState, 
  TerminalServerMessage, 
  TerminalClientMessage,
  ClientStdinMessage, 
  ClientResizeMessage 
} from '../types/terminal';

export interface UseTerminalOptions {
  worktreeId: string;
  onOutput?: (data: string) => void;
  onError?: (error: string) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  autoConnect?: boolean;
}

export interface UseTerminalReturn {
  state: TerminalState;
  connect: () => void;
  disconnect: () => void;
  sendInput: (data: string) => void;
  resize: (cols: number, rows: number) => void;
  isReady: boolean;
}

export function useTerminal(options: UseTerminalOptions): UseTerminalReturn {
  const {
    worktreeId,
    onOutput,
    onError,
    onConnect,
    onDisconnect,
    autoConnect = false
  } = options;

  const mountedRef = useMounted();

  const [state, setState] = useState<TerminalState>({
    isConnected: false,
    isConnecting: false,
    error: null,
    worktreeId
  });

  // WebSocket message handler
  const handleMessage = useCallback((event: MessageEvent) => {
    try {
      const message: TerminalServerMessage = JSON.parse(event.data);
      
      switch (message.type) {
        case 'stdout':
        case 'stderr':
          if (message.data) {
            onOutput?.(message.data);
          }
          break;
        case 'exit':
          if (message.exitCode !== undefined) {
            const exitMessage = `\r\n\x1b[33mProcess exited with code ${message.exitCode}\x1b[0m\r\n`;
            onOutput?.(exitMessage);
          }
          break;
        case 'pong':
          // Handle pong response (for keepalive)
          break;
        default:
          // Unknown message type, ignore
          console.warn('Unknown terminal message type:', message);
      }
    } catch (error) {
      console.error('Failed to parse terminal message:', error);
      onError?.('Failed to parse terminal message');
    }
  }, [onOutput, onError, mountedRef]);

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

  // Construct WebSocket URL directly for AI container access
  const getWebSocketUrl = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    return `${protocol}//${host}/api/ai/attach/${worktreeId}`;
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

  // Send input to terminal
  const sendInput = useCallback((data: string) => {
    const message: ClientStdinMessage = {
      type: 'stdin',
      data
    };

    const success = webSocketControls.send(JSON.stringify(message));
    if (!success) {
      onError?.('Failed to send input');
    }
  }, [onError, webSocketControls.send]);

  // Resize terminal
  const resize = useCallback((cols: number, rows: number) => {
    const message: ClientResizeMessage = {
      type: 'resize',
      cols,
      rows
    };

    const success = webSocketControls.send(JSON.stringify(message));
    if (!success) {
      onError?.('Failed to resize terminal');
    }
  }, [onError, webSocketControls.send]);

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
    sendInput,
    resize,
    isReady: state.isConnected && !state.isConnecting
  };
}