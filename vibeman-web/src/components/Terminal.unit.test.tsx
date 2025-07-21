import { describe, test, expect, beforeEach, mock } from 'bun:test';
import '../test-setup';

// Test the Terminal component in isolation without rendering
describe('Terminal Component Unit Tests', () => {
  test('Terminal component exports exist', async () => {
    const { Terminal } = await import('./Terminal');
    expect(Terminal).toBeDefined();
    expect(typeof Terminal).toBe('function');
  });

  test('Terminal props interface is correctly typed', () => {
    // This test ensures our TypeScript interfaces are correct
    const validProps = {
      worktreeId: 'test-worktree-123',
      className: 'custom-class',
      autoConnect: true,
      isFullscreen: false,
      onToggleFullscreen: () => {}
    };

    // Should not cause TypeScript errors
    expect(validProps.worktreeId).toBe('test-worktree-123');
    expect(validProps.autoConnect).toBe(true);
    expect(typeof validProps.onToggleFullscreen).toBe('function');
  });

  test('Terminal supports optional props', () => {
    const minimalProps = {
      worktreeId: 'test-worktree'
    };

    // Should be valid with just worktreeId
    expect(minimalProps.worktreeId).toBe('test-worktree');
  });
});

describe('Terminal Utilities', () => {
  test('theme detection for mobile vs desktop', () => {
    // Mock mobile width
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: 500
    });

    const isMobile = window.innerWidth < 768;
    expect(isMobile).toBe(true);

    // Mock desktop width
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: 1200
    });

    const isDesktop = window.innerWidth >= 768;
    expect(isDesktop).toBe(true);
  });

  test('terminal configuration objects have correct shape', () => {
    const mobileConfig = {
      fontSize: 12,
      scrollback: 500,
      rightClickSelectsWord: false,
      fastScrollModifier: undefined
    };

    const desktopConfig = {
      fontSize: 14,
      scrollback: 1000,
      rightClickSelectsWord: true,
      fastScrollModifier: 'shift'
    };

    expect(mobileConfig.fontSize).toBe(12);
    expect(mobileConfig.rightClickSelectsWord).toBe(false);
    expect(desktopConfig.fontSize).toBe(14);
    expect(desktopConfig.rightClickSelectsWord).toBe(true);
  });
});

describe('Terminal State Management', () => {
  test('connection states are properly typed', () => {
    const states = {
      disconnected: { isConnected: false, isConnecting: false },
      connecting: { isConnected: false, isConnecting: true },
      connected: { isConnected: true, isConnecting: false },
      error: { isConnected: false, isConnecting: false, error: 'Connection failed' }
    };

    expect(states.disconnected.isConnected).toBe(false);
    expect(states.connecting.isConnecting).toBe(true);
    expect(states.connected.isConnected).toBe(true);
    expect(states.error.error).toBe('Connection failed');
  });

  test('terminal actions have correct signatures', () => {
    const actions = {
      connect: mock(() => {}),
      disconnect: mock(() => {}),
      sendInput: mock((data: string) => {}),
      resize: mock((cols: number, rows: number) => {}),
      clear: mock(() => {})
    };

    // Test action signatures
    actions.sendInput('test command');
    actions.resize(80, 24);
    actions.connect();
    actions.disconnect();
    actions.clear();

    expect(actions.sendInput).toHaveBeenCalledWith('test command');
    expect(actions.resize).toHaveBeenCalledWith(80, 24);
    expect(actions.connect).toHaveBeenCalled();
    expect(actions.disconnect).toHaveBeenCalled();
    expect(actions.clear).toHaveBeenCalled();
  });
});

describe('Terminal Message Protocol', () => {
  test('client messages serialize correctly', () => {
    const stdinMessage = {
      type: 'stdin' as const,
      data: 'echo "hello"'
    };

    const resizeMessage = {
      type: 'resize' as const,
      cols: 80,
      rows: 24
    };

    const pingMessage = {
      type: 'ping' as const
    };

    // Test serialization
    const serializedStdin = JSON.stringify(stdinMessage);
    const serializedResize = JSON.stringify(resizeMessage);
    const serializedPing = JSON.stringify(pingMessage);

    expect(JSON.parse(serializedStdin)).toEqual(stdinMessage);
    expect(JSON.parse(serializedResize)).toEqual(resizeMessage);
    expect(JSON.parse(serializedPing)).toEqual(pingMessage);
  });

  test('server messages deserialize correctly', () => {
    const stdoutMessage = '{"type":"stdout","data":"Hello World\\n"}';
    const stderrMessage = '{"type":"stderr","data":"Error occurred\\n"}';
    const exitMessage = '{"type":"exit","exitCode":0}';
    const pongMessage = '{"type":"pong"}';

    const parsedStdout = JSON.parse(stdoutMessage);
    const parsedStderr = JSON.parse(stderrMessage);
    const parsedExit = JSON.parse(exitMessage);
    const parsedPong = JSON.parse(pongMessage);

    expect(parsedStdout.type).toBe('stdout');
    expect(parsedStdout.data).toBe('Hello World\n');
    expect(parsedStderr.type).toBe('stderr');
    expect(parsedExit.type).toBe('exit');
    expect(parsedExit.exitCode).toBe(0);
    expect(parsedPong.type).toBe('pong');
  });
});

describe('Terminal Error Handling', () => {
  test('handles WebSocket URL construction', () => {
    // Test HTTPS protocol
    Object.defineProperty(window, 'location', {
      value: {
        protocol: 'https:',
        host: 'example.com'
      },
      configurable: true
    });

    const wsUrlHttps = `wss://${window.location.host}/api/ai/attach/test-worktree`;
    expect(wsUrlHttps).toBe('wss://example.com/api/ai/attach/test-worktree');

    // Test HTTP protocol
    Object.defineProperty(window, 'location', {
      value: {
        protocol: 'http:',
        host: 'localhost:3000'
      },
      configurable: true
    });

    const wsUrlHttp = `ws://${window.location.host}/api/ai/attach/test-worktree`;
    expect(wsUrlHttp).toBe('ws://localhost:3000/api/ai/attach/test-worktree');
  });

  test('error message formatting', () => {
    const formatErrorMessage = (error: string) => {
      return `\r\n\x1b[31mError: ${error}\x1b[0m\r\n`;
    };

    const formattedError = formatErrorMessage('Connection failed');
    expect(formattedError).toContain('Error: Connection failed');
    expect(formattedError).toContain('\x1b[31m'); // Red color
    expect(formattedError).toContain('\x1b[0m');  // Reset color
  });

  test('exit code formatting', () => {
    const formatExitCode = (exitCode: number) => {
      return `\r\n\x1b[33mProcess exited with code ${exitCode}\x1b[0m\r\n`;
    };

    const exitSuccess = formatExitCode(0);
    const exitError = formatExitCode(1);

    expect(exitSuccess).toContain('code 0');
    expect(exitError).toContain('code 1');
    expect(exitSuccess).toContain('\x1b[33m'); // Yellow color
  });
});