import { useCallback, useEffect, useRef, useState } from 'react';

export interface WebSocketState {
  isConnected: boolean;
  isConnecting: boolean;
  error: string | null;
}

export interface UseWebSocketOptions {
  url: string | (() => WebSocket);
  onOpen?: () => void;
  onMessage?: (event: MessageEvent) => void;
  onClose?: (event: CloseEvent) => void;
  onError?: (error: Event) => void;
  autoConnect?: boolean;
  reconnectAttempts?: number;
  reconnectDelay?: number;
}

export interface UseWebSocketReturn {
  state: WebSocketState;
  connect: () => void;
  disconnect: () => void;
  send: (data: string) => boolean;
  ws: WebSocket | null;
}

/**
 * Shared WebSocket hook with automatic reconnection and lifecycle management
 */
export function useWebSocket(options: UseWebSocketOptions): UseWebSocketReturn {
  const {
    url,
    onOpen,
    onMessage,
    onClose,
    onError,
    autoConnect = false,
    reconnectAttempts = 5,
    reconnectDelay = 2000
  } = options;

  const [state, setState] = useState<WebSocketState>({
    isConnected: false,
    isConnecting: false,
    error: null
  });

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const mountedRef = useRef(true);

  // Cleanup function
  const cleanup = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (wsRef.current) {
      wsRef.current.onopen = null;
      wsRef.current.onmessage = null;
      wsRef.current.onclose = null;
      wsRef.current.onerror = null;
      
      if (wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.close(1000, 'Component unmounting');
      }
      wsRef.current = null;
    }
  }, []);

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (!mountedRef.current) return;
    
    // Clean up any existing connection first
    cleanup();
    
    if (state.isConnecting || state.isConnected) {
      return;
    }

    setState(prev => ({
      ...prev,
      isConnecting: true,
      error: null
    }));

    try {
      const ws = typeof url === 'function' ? url() : new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = (event) => {
        if (!mountedRef.current) return;
        
        setState(prev => ({
          ...prev,
          isConnected: true,
          isConnecting: false,
          error: null
        }));
        reconnectAttemptsRef.current = 0;
        onOpen?.();
      };

      ws.onmessage = (event) => {
        if (!mountedRef.current) return;
        onMessage?.(event);
      };

      ws.onclose = (event) => {
        if (!mountedRef.current) return;
        
        setState(prev => ({
          ...prev,
          isConnected: false,
          isConnecting: false
        }));

        // Attempt to reconnect if it wasn't a manual disconnect
        if (event.code !== 1000 && reconnectAttemptsRef.current < reconnectAttempts) {
          reconnectAttemptsRef.current++;
          
          // Clear any existing timeout before setting a new one
          if (reconnectTimeoutRef.current) {
            clearTimeout(reconnectTimeoutRef.current);
          }
          
          reconnectTimeoutRef.current = setTimeout(() => {
            if (!mountedRef.current) return;
            reconnectTimeoutRef.current = null;
            connect();
          }, reconnectDelay);
        } else {
          onClose?.(event);
        }
      };

      ws.onerror = (error) => {
        if (!mountedRef.current) return;
        
        console.error('WebSocket error:', error);
        setState(prev => ({
          ...prev,
          error: 'Connection failed',
          isConnecting: false,
          isConnected: false
        }));
        onError?.(error);
      };

    } catch (error) {
      if (!mountedRef.current) return;
      
      console.error('Failed to create WebSocket:', error);
      setState(prev => ({
        ...prev,
        error: 'Failed to connect',
        isConnecting: false,
        isConnected: false
      }));
    }
  }, [url, onOpen, onMessage, onClose, onError, reconnectAttempts, reconnectDelay, cleanup]);

  // Disconnect from WebSocket
  const disconnect = useCallback(() => {
    cleanup();
    if (mountedRef.current) {
      setState(prev => ({
        ...prev,
        isConnected: false,
        isConnecting: false,
        error: null
      }));
    }
  }, [cleanup]);

  // Send message
  const send = useCallback((data: string): boolean => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.warn('Cannot send message: WebSocket not connected');
      return false;
    }

    try {
      wsRef.current.send(data);
      return true;
    } catch (error) {
      console.error('Failed to send WebSocket message:', error);
      return false;
    }
  }, []);

  // Auto-connect on mount if enabled
  useEffect(() => {
    if (autoConnect) {
      connect();
    }

    // Cleanup on unmount
    return () => {
      mountedRef.current = false;
      cleanup();
    };
  }, []); // Empty dependency array - only run on mount/unmount

  return {
    state,
    connect,
    disconnect,
    send,
    ws: wsRef.current
  };
}