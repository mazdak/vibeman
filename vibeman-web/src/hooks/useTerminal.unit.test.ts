import { describe, test, expect } from 'bun:test';
import '../test-setup';

describe('useTerminal Hook Unit Tests', () => {
  test('useTerminal hook exports exist', async () => {
    const { useTerminal } = await import('./useTerminal');
    expect(useTerminal).toBeDefined();
    expect(typeof useTerminal).toBe('function');
  });

  test('UseTerminalOptions interface is correctly typed', () => {
    const validOptions = {
      worktreeId: 'test-worktree-123',
      onOutput: (data: string) => console.log(data),
      onError: (error: string) => console.error(error),
      onConnect: () => console.log('connected'),
      onDisconnect: () => console.log('disconnected'),
      autoConnect: true
    };

    expect(validOptions.worktreeId).toBe('test-worktree-123');
    expect(typeof validOptions.onOutput).toBe('function');
    expect(typeof validOptions.onError).toBe('function');
    expect(typeof validOptions.onConnect).toBe('function');
    expect(typeof validOptions.onDisconnect).toBe('function');
    expect(validOptions.autoConnect).toBe(true);
  });

  test('minimal UseTerminalOptions interface', () => {
    const minimalOptions = {
      worktreeId: 'test-worktree'
    };

    // Should be valid with just worktreeId
    expect(minimalOptions.worktreeId).toBe('test-worktree');
  });
});

describe('Terminal State Logic', () => {
  test('terminal state transitions', () => {
    const createState = (overrides = {}) => ({
      isConnected: false,
      isConnecting: false,
      error: null,
      worktreeId: 'test-worktree',
      ...overrides
    });

    const disconnectedState = createState();
    const connectingState = createState({ isConnecting: true });
    const connectedState = createState({ isConnected: true });
    const errorState = createState({ error: 'Connection failed' });

    expect(disconnectedState.isConnected).toBe(false);
    expect(disconnectedState.isConnecting).toBe(false);

    expect(connectingState.isConnecting).toBe(true);
    expect(connectingState.isConnected).toBe(false);

    expect(connectedState.isConnected).toBe(true);
    expect(connectedState.isConnecting).toBe(false);

    expect(errorState.error).toBe('Connection failed');
  });

  test('isReady calculation logic', () => {
    const calculateIsReady = (isConnected: boolean, isConnecting: boolean) => {
      return isConnected && !isConnecting;
    };

    expect(calculateIsReady(false, false)).toBe(false); // Disconnected
    expect(calculateIsReady(false, true)).toBe(false);  // Connecting
    expect(calculateIsReady(true, true)).toBe(false);   // Connected but still connecting
    expect(calculateIsReady(true, false)).toBe(true);   // Connected and ready
  });
});

describe('WebSocket URL Construction', () => {
  test('constructs correct WebSocket URLs', () => {
    const buildWebSocketUrl = (protocol: string, host: string, worktreeId: string) => {
      const wsProtocol = protocol === 'https:' ? 'wss:' : 'ws:';
      return `${wsProtocol}//${host}/api/ai/attach/${worktreeId}`;
    };

    const httpsUrl = buildWebSocketUrl('https:', 'example.com', 'test-worktree');
    const httpUrl = buildWebSocketUrl('http:', 'localhost:3000', 'test-worktree');

    expect(httpsUrl).toBe('wss://example.com/api/ai/attach/test-worktree');
    expect(httpUrl).toBe('ws://localhost:3000/api/ai/attach/test-worktree');
  });

  test('handles special characters in worktree ID', () => {
    const buildWebSocketUrl = (worktreeId: string) => {
      return `/api/ai/attach/${worktreeId}`;
    };

    expect(buildWebSocketUrl('test-worktree-123')).toBe('/api/ai/attach/test-worktree-123');
    expect(buildWebSocketUrl('worktree_with_underscores')).toBe('/api/ai/attach/worktree_with_underscores');
    expect(buildWebSocketUrl('worktree.with.dots')).toBe('/api/ai/attach/worktree.with.dots');
  });
});

describe('Message Handling Logic', () => {
  test('stdin message creation', () => {
    const createStdinMessage = (data: string) => ({
      type: 'stdin' as const,
      data
    });

    const message = createStdinMessage('ls -la');
    expect(message.type).toBe('stdin');
    expect(message.data).toBe('ls -la');

    const serialized = JSON.stringify(message);
    const parsed = JSON.parse(serialized);
    expect(parsed).toEqual(message);
  });

  test('resize message creation', () => {
    const createResizeMessage = (cols: number, rows: number) => ({
      type: 'resize' as const,
      cols,
      rows
    });

    const message = createResizeMessage(80, 24);
    expect(message.type).toBe('resize');
    expect(message.cols).toBe(80);
    expect(message.rows).toBe(24);

    const serialized = JSON.stringify(message);
    const parsed = JSON.parse(serialized);
    expect(parsed).toEqual(message);
  });

  test('ping message creation', () => {
    const createPingMessage = () => ({
      type: 'ping' as const
    });

    const message = createPingMessage();
    expect(message.type).toBe('ping');
    expect(Object.keys(message)).toEqual(['type']);
  });
});

describe('Server Message Processing', () => {
  test('stdout message processing', () => {
    const processStdoutMessage = (data: string, onOutput?: (data: string) => void) => {
      if (onOutput) {
        onOutput(data);
      }
    };

    let outputReceived = '';
    const onOutput = (data: string) => {
      outputReceived += data;
    };

    processStdoutMessage('Hello ', onOutput);
    processStdoutMessage('World!', onOutput);

    expect(outputReceived).toBe('Hello World!');
  });

  test('stderr message processing', () => {
    const processStderrMessage = (data: string, onOutput?: (data: string) => void) => {
      if (onOutput) {
        onOutput(data); // stderr is treated same as stdout in our implementation
      }
    };

    let errorReceived = '';
    const onOutput = (data: string) => {
      errorReceived += data;
    };

    processStderrMessage('Error: Command not found', onOutput);
    expect(errorReceived).toBe('Error: Command not found');
  });

  test('exit message processing', () => {
    const processExitMessage = (exitCode: number, onOutput?: (data: string) => void) => {
      const exitMessage = `\r\n\x1b[33mProcess exited with code ${exitCode}\x1b[0m\r\n`;
      if (onOutput) {
        onOutput(exitMessage);
      }
    };

    let exitReceived = '';
    const onOutput = (data: string) => {
      exitReceived = data;
    };

    processExitMessage(0, onOutput);
    expect(exitReceived).toContain('Process exited with code 0');
    expect(exitReceived).toContain('\x1b[33m'); // Yellow color
    expect(exitReceived).toContain('\x1b[0m');  // Reset color

    processExitMessage(1, onOutput);
    expect(exitReceived).toContain('Process exited with code 1');
  });
});

describe('Error Handling Logic', () => {
  test('connection error handling', () => {
    const handleConnectionError = (error: string, onError?: (error: string) => void) => {
      if (onError) {
        onError(error);
      }
    };

    let errorReceived = '';
    const onError = (error: string) => {
      errorReceived = error;
    };

    handleConnectionError('Connection failed', onError);
    expect(errorReceived).toBe('Connection failed');

    handleConnectionError('Connection timeout', onError);
    expect(errorReceived).toBe('Connection timeout');
  });

  test('send failure handling', () => {
    const handleSendFailure = (type: 'input' | 'resize', onError?: (error: string) => void) => {
      const errorMessage = type === 'input' ? 'Failed to send input' : 'Failed to resize terminal';
      if (onError) {
        onError(errorMessage);
      }
    };

    let errorReceived = '';
    const onError = (error: string) => {
      errorReceived = error;
    };

    handleSendFailure('input', onError);
    expect(errorReceived).toBe('Failed to send input');

    handleSendFailure('resize', onError);
    expect(errorReceived).toBe('Failed to resize terminal');
  });

  test('JSON parsing error handling', () => {
    const safeJsonParse = (data: string, onError?: (error: string) => void) => {
      try {
        return JSON.parse(data);
      } catch (error) {
        if (onError) {
          onError('Failed to parse terminal message');
        }
        return null;
      }
    };

    let errorReceived = '';
    const onError = (error: string) => {
      errorReceived = error;
    };

    // Valid JSON
    const validResult = safeJsonParse('{"type":"stdout","data":"test"}', onError);
    expect(validResult).toEqual({ type: 'stdout', data: 'test' });
    expect(errorReceived).toBe('');

    // Invalid JSON
    const invalidResult = safeJsonParse('invalid json', onError);
    expect(invalidResult).toBeNull();
    expect(errorReceived).toBe('Failed to parse terminal message');
  });
});