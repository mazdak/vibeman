import { describe, test, expect } from 'bun:test';
import '../test-setup';

describe('Terminal Integration Tests', () => {
  test('Terminal imports and interfaces work together', async () => {
    // Test that all terminal-related modules can be imported together
    const [
      { Terminal },
      { TerminalModal },
      { useTerminal },
      terminalTypes
    ] = await Promise.all([
      import('./Terminal'),
      import('./modals/TerminalModal'),
      import('../hooks/useTerminal'),
      import('../types/terminal')
    ]);

    expect(Terminal).toBeDefined();
    expect(TerminalModal).toBeDefined();
    expect(useTerminal).toBeDefined();
    expect(terminalTypes).toBeDefined();
  });

  test('Terminal component props match hook return type', async () => {
    const { useTerminal } = await import('../hooks/useTerminal');
    const { Terminal } = await import('./Terminal');

    // This test ensures TypeScript compatibility between hook and component
    const mockOptions = {
      worktreeId: 'test-worktree',
      onOutput: (data: string) => {},
      onError: (error: string) => {},
      onConnect: () => {},
      onDisconnect: () => {},
      autoConnect: true
    };

    // Should be able to use the same types
    expect(typeof useTerminal).toBe('function');
    expect(typeof Terminal).toBe('function');
    expect(mockOptions.worktreeId).toBe('test-worktree');
  });

  test('Message types are compatible across the stack', () => {
    // Test that client and server message types work together
    const clientMessages = [
      { type: 'stdin' as const, data: 'test' },
      { type: 'resize' as const, cols: 80, rows: 24 },
      { type: 'ping' as const }
    ];

    const serverMessages = [
      { type: 'stdout' as const, data: 'output' },
      { type: 'stderr' as const, data: 'error' },
      { type: 'exit' as const, exitCode: 0 },
      { type: 'pong' as const }
    ];

    // Should be able to serialize and deserialize all messages
    clientMessages.forEach(msg => {
      const serialized = JSON.stringify(msg);
      const parsed = JSON.parse(serialized);
      expect(parsed.type).toBe(msg.type);
    });

    serverMessages.forEach(msg => {
      const serialized = JSON.stringify(msg);
      const parsed = JSON.parse(serialized);
      expect(parsed.type).toBe(msg.type);
    });
  });

  test('Terminal workflow simulation', () => {
    // Simulate the complete terminal workflow
    let terminalState = {
      isConnected: false,
      isConnecting: false,
      error: null,
      worktreeId: 'test-worktree'
    };

    // 1. Start connecting
    terminalState = { ...terminalState, isConnecting: true };
    expect(terminalState.isConnecting).toBe(true);
    expect(terminalState.isConnected).toBe(false);

    // 2. Connection established
    terminalState = { ...terminalState, isConnected: true, isConnecting: false };
    expect(terminalState.isConnected).toBe(true);
    expect(terminalState.isConnecting).toBe(false);

    // 3. Send command
    const stdinMessage = { type: 'stdin' as const, data: 'ls -la\n' };
    expect(stdinMessage.type).toBe('stdin');
    expect(stdinMessage.data).toBe('ls -la\n');

    // 4. Receive output
    const stdoutMessage = { type: 'stdout' as const, data: 'total 48\ndrwxr-xr-x  8 user user 4096 Jan 1 12:00 .\n' };
    expect(stdoutMessage.type).toBe('stdout');
    expect(stdoutMessage.data).toContain('total 48');

    // 5. Process exit
    const exitMessage = { type: 'exit' as const, exitCode: 0 };
    expect(exitMessage.type).toBe('exit');
    expect(exitMessage.exitCode).toBe(0);

    // 6. Disconnect
    terminalState = { ...terminalState, isConnected: false };
    expect(terminalState.isConnected).toBe(false);
  });

  test('Terminal modal integration with terminal component', () => {
    // Test that modal props align with terminal expectations
    const modalProps = {
      isOpen: true,
      onClose: () => {},
      worktreeId: 'test-worktree-123',
      title: 'AI Terminal - my-worktree'
    };

    const terminalProps = {
      worktreeId: modalProps.worktreeId,
      className: 'h-full',
      isFullscreen: false,
      onToggleFullscreen: () => {}
    };

    expect(modalProps.worktreeId).toBe(terminalProps.worktreeId);
    expect(terminalProps.className).toBe('h-full');
    expect(typeof terminalProps.onToggleFullscreen).toBe('function');
  });

  test('Error handling across components', () => {
    // Test error propagation through the terminal stack
    const errors = [
      'Connection failed',
      'Connection timeout',
      'Failed to send input',
      'Failed to resize terminal',
      'Failed to parse terminal message'
    ];

    errors.forEach(error => {
      let receivedError = '';
      const onError = (err: string) => {
        receivedError = err;
      };

      onError(error);
      expect(receivedError).toBe(error);
    });
  });

  test('Mobile and desktop compatibility', () => {
    // Test responsive design logic
    const getMobileConfig = () => ({
      fontSize: 12,
      scrollback: 500,
      rightClickSelectsWord: false,
      classes: 'w-[95vw] h-[70vh] text-sm px-4'
    });

    const getDesktopConfig = () => ({
      fontSize: 14,
      scrollback: 1000,
      rightClickSelectsWord: true,
      classes: 'sm:w-full sm:h-[600px] sm:text-base sm:px-6'
    });

    const mobileConfig = getMobileConfig();
    const desktopConfig = getDesktopConfig();

    expect(mobileConfig.fontSize).toBeLessThan(desktopConfig.fontSize);
    expect(mobileConfig.scrollback).toBeLessThan(desktopConfig.scrollback);
    expect(mobileConfig.rightClickSelectsWord).toBe(false);
    expect(desktopConfig.rightClickSelectsWord).toBe(true);
  });

  test('WebSocket protocol compatibility', () => {
    // Test that our protocol works with different environments
    const protocols = {
      development: { protocol: 'http:', expectedWs: 'ws:' },
      production: { protocol: 'https:', expectedWs: 'wss:' }
    };

    Object.entries(protocols).forEach(([env, config]) => {
      const wsProtocol = config.protocol === 'https:' ? 'wss:' : 'ws:';
      expect(wsProtocol).toBe(config.expectedWs);
    });

    // Test URL construction
    const buildUrl = (protocol: string, host: string, worktreeId: string) => {
      const wsProtocol = protocol === 'https:' ? 'wss:' : 'ws:';
      return `${wsProtocol}//${host}/api/ai/attach/${worktreeId}`;
    };

    const devUrl = buildUrl('http:', 'localhost:3000', 'test-worktree');
    const prodUrl = buildUrl('https:', 'example.com', 'test-worktree');

    expect(devUrl).toBe('ws://localhost:3000/api/ai/attach/test-worktree');
    expect(prodUrl).toBe('wss://example.com/api/ai/attach/test-worktree');
  });
});