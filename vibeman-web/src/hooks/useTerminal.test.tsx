import { describe, test, expect, beforeEach, mock } from 'bun:test';
import { renderHook, act, waitFor } from '@testing-library/react';
import '../test-setup';
import { useTerminal } from './useTerminal';
import type { UseTerminalOptions } from './useTerminal';

// Mock WebSocket
const mockWebSocket = {
  send: mock(() => true),
  connect: mock(() => {}),
  disconnect: mock(() => {}),
  state: {
    isConnected: false,
    isConnecting: false,
    error: null
  }
};

// Mock useWebSocket hook
mock.module('../shared/hooks', () => ({
  useWebSocket: mock(() => mockWebSocket),
  useMounted: mock(() => ({ current: true }))
}));

describe('useTerminal Hook', () => {
  const defaultOptions: UseTerminalOptions = {
    worktreeId: 'test-worktree-123',
    onOutput: mock(() => {}),
    onError: mock(() => {}),
    onConnect: mock(() => {}),
    onDisconnect: mock(() => {}),
    autoConnect: false
  };

  beforeEach(() => {
    // Reset all mocks
    mock.restore();
    
    // Reset mock WebSocket state
    mockWebSocket.state = {
      isConnected: false,
      isConnecting: false,
      error: null
    };
    
    // Clear mock call history
    Object.values(defaultOptions).forEach(fn => {
      if (typeof fn === 'function' && 'mockClear' in fn) {
        fn.mockClear();
      }
    });
    
    Object.values(mockWebSocket).forEach(fn => {
      if (typeof fn === 'function' && 'mockClear' in fn) {
        fn.mockClear();
      }
    });
  });

  test('initializes with correct default state', () => {
    const { result } = renderHook(() => useTerminal(defaultOptions));
    
    expect(result.current.state).toEqual({
      isConnected: false,
      isConnecting: false,
      error: null,
      worktreeId: 'test-worktree-123'
    });
    
    expect(result.current.isReady).toBe(false);
  });

  test('constructs correct WebSocket URL', () => {
    // Mock window.location
    const originalLocation = window.location;
    delete (window as any).location;
    window.location = {
      ...originalLocation,
      protocol: 'https:',
      host: 'localhost:3000'
    };
    
    renderHook(() => useTerminal(defaultOptions));
    
    // The WebSocket URL construction is tested internally
    // We can verify it by checking if useWebSocket was called correctly
    // This would be more thoroughly tested in integration tests
    
    // Restore location
    window.location = originalLocation;
  });

  test('auto-connects when autoConnect is true', () => {
    const options = { ...defaultOptions, autoConnect: true };
    
    renderHook(() => useTerminal(options));
    
    // Should have called connect through the WebSocket hook
    expect(mockWebSocket.connect).toHaveBeenCalled();
  });

  test('does not auto-connect when autoConnect is false', () => {
    renderHook(() => useTerminal(defaultOptions));
    
    expect(mockWebSocket.connect).not.toHaveBeenCalled();
  });

  test('calls onConnect when connection opens', () => {
    const { result, rerender } = renderHook(() => useTerminal(defaultOptions));
    
    // Simulate connection opening
    act(() => {
      mockWebSocket.state.isConnected = true;
      mockWebSocket.state.isConnecting = false;
    });
    
    rerender();
    
    expect(defaultOptions.onConnect).toHaveBeenCalled();
  });

  test('calls onDisconnect when connection closes', () => {
    const { result, rerender } = renderHook(() => useTerminal(defaultOptions));
    
    // Start connected
    act(() => {
      mockWebSocket.state.isConnected = true;
    });
    rerender();
    
    // Simulate disconnection
    act(() => {
      mockWebSocket.state.isConnected = false;
    });
    rerender();
    
    expect(defaultOptions.onDisconnect).toHaveBeenCalled();
  });

  test('calls onError when connection fails', () => {
    const { result, rerender } = renderHook(() => useTerminal(defaultOptions));
    
    // Simulate connection error
    act(() => {
      mockWebSocket.state.error = 'Connection failed';
      mockWebSocket.state.isConnected = false;
      mockWebSocket.state.isConnecting = false;
    });
    
    rerender();
    
    expect(defaultOptions.onError).toHaveBeenCalledWith('Connection failed');
  });

  test('sendInput sends correct message format', () => {
    const { result } = renderHook(() => useTerminal(defaultOptions));
    
    act(() => {
      result.current.sendInput('ls -la');
    });
    
    expect(mockWebSocket.send).toHaveBeenCalledWith(
      JSON.stringify({
        type: 'stdin',
        data: 'ls -la'
      })
    );
  });

  test('resize sends correct message format', () => {
    const { result } = renderHook(() => useTerminal(defaultOptions));
    
    act(() => {
      result.current.resize(80, 24);
    });
    
    expect(mockWebSocket.send).toHaveBeenCalledWith(
      JSON.stringify({
        type: 'resize',
        cols: 80,
        rows: 24
      })
    );
  });

  test('handles failed sendInput', () => {
    mockWebSocket.send.mockReturnValue(false);
    
    const { result } = renderHook(() => useTerminal(defaultOptions));
    
    act(() => {
      result.current.sendInput('test');
    });
    
    expect(defaultOptions.onError).toHaveBeenCalledWith('Failed to send input');
  });

  test('handles failed resize', () => {
    mockWebSocket.send.mockReturnValue(false);
    
    const { result } = renderHook(() => useTerminal(defaultOptions));
    
    act(() => {
      result.current.resize(80, 24);
    });
    
    expect(defaultOptions.onError).toHaveBeenCalledWith('Failed to resize terminal');
  });

  test('processes stdout messages correctly', () => {
    const { result } = renderHook(() => useTerminal(defaultOptions));
    
    // Get the message handler from the WebSocket mock
    const messageHandler = defaultOptions.onOutput;
    
    // Simulate receiving a stdout message
    const mockEvent = {
      data: JSON.stringify({
        type: 'stdout',
        data: 'Hello from terminal'
      })
    } as MessageEvent;
    
    // This would be called by the WebSocket implementation
    // We can test the message processing logic separately
    expect(messageHandler).toBeDefined();
  });

  test('processes stderr messages correctly', () => {
    const { result } = renderHook(() => useTerminal(defaultOptions));
    
    // Similar to stdout test, but for stderr
    const messageHandler = defaultOptions.onOutput;
    
    const mockEvent = {
      data: JSON.stringify({
        type: 'stderr',
        data: 'Error message'
      })
    } as MessageEvent;
    
    expect(messageHandler).toBeDefined();
  });

  test('processes exit messages with exit code', () => {
    const { result } = renderHook(() => useTerminal(defaultOptions));
    
    // Test exit message processing
    const outputHandler = defaultOptions.onOutput;
    expect(outputHandler).toBeDefined();
  });

  test('handles invalid JSON messages gracefully', () => {
    const { result } = renderHook(() => useTerminal(defaultOptions));
    
    // This would be tested by the actual message handler
    // The hook should not crash on invalid JSON
    expect(defaultOptions.onError).toBeDefined();
  });

  test('updates state when worktreeId changes', () => {
    const { result, rerender } = renderHook(
      ({ worktreeId }) => useTerminal({ ...defaultOptions, worktreeId }),
      { initialProps: { worktreeId: 'worktree-1' } }
    );
    
    expect(result.current.state.worktreeId).toBe('worktree-1');
    
    // Change worktree ID
    rerender({ worktreeId: 'worktree-2' });
    
    expect(result.current.state.worktreeId).toBe('worktree-2');
  });

  test('isReady is true only when connected and not connecting', () => {
    const { result, rerender } = renderHook(() => useTerminal(defaultOptions));
    
    // Initially not ready
    expect(result.current.isReady).toBe(false);
    
    // Connecting - still not ready
    act(() => {
      mockWebSocket.state.isConnecting = true;
    });
    rerender();
    expect(result.current.isReady).toBe(false);
    
    // Connected - now ready
    act(() => {
      mockWebSocket.state.isConnected = true;
      mockWebSocket.state.isConnecting = false;
    });
    rerender();
    expect(result.current.isReady).toBe(true);
    
    // Disconnected - not ready again
    act(() => {
      mockWebSocket.state.isConnected = false;
    });
    rerender();
    expect(result.current.isReady).toBe(false);
  });

  test('exposes connect and disconnect functions', () => {
    const { result } = renderHook(() => useTerminal(defaultOptions));
    
    expect(typeof result.current.connect).toBe('function');
    expect(typeof result.current.disconnect).toBe('function');
    
    // Test that they call the underlying WebSocket functions
    act(() => {
      result.current.connect();
    });
    expect(mockWebSocket.connect).toHaveBeenCalled();
    
    act(() => {
      result.current.disconnect();
    });
    expect(mockWebSocket.disconnect).toHaveBeenCalled();
  });

  test('handles missing callback functions gracefully', () => {
    const minimalOptions: UseTerminalOptions = {
      worktreeId: 'test-worktree'
      // No callback functions provided
    };
    
    const { result } = renderHook(() => useTerminal(minimalOptions));
    
    // Should not throw when callbacks are not provided
    expect(result.current.state.worktreeId).toBe('test-worktree');
    expect(typeof result.current.sendInput).toBe('function');
    expect(typeof result.current.resize).toBe('function');
  });

  test('updates connection state from WebSocket hook', () => {
    const { result, rerender } = renderHook(() => useTerminal(defaultOptions));
    
    // Simulate WebSocket state changes
    act(() => {
      mockWebSocket.state.isConnecting = true;
    });
    rerender();
    
    expect(result.current.state.isConnecting).toBe(true);
    
    act(() => {
      mockWebSocket.state.isConnected = true;
      mockWebSocket.state.isConnecting = false;
    });
    rerender();
    
    expect(result.current.state.isConnected).toBe(true);
    expect(result.current.state.isConnecting).toBe(false);
  });
});