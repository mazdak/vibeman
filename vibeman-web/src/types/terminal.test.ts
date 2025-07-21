import { describe, test, expect } from 'bun:test';
import type {
  TerminalState,
  TerminalServerMessage,
  TerminalClientMessage,
  ClientStdinMessage,
  ClientResizeMessage,
  ClientPingMessage,
  ServerStdoutMessage,
  ServerStderrMessage,
  ServerExitMessage,
  ServerPongMessage
} from './terminal';

describe('Terminal Types', () => {
  describe('TerminalState', () => {
    test('has correct shape for initial state', () => {
      const state: TerminalState = {
        isConnected: false,
        isConnecting: false,
        error: null,
        worktreeId: 'test-worktree'
      };
      
      expect(state.isConnected).toBe(false);
      expect(state.isConnecting).toBe(false);
      expect(state.error).toBeNull();
      expect(state.worktreeId).toBe('test-worktree');
    });
    
    test('has correct shape for connected state', () => {
      const state: TerminalState = {
        isConnected: true,
        isConnecting: false,
        error: null,
        worktreeId: 'test-worktree'
      };
      
      expect(state.isConnected).toBe(true);
      expect(state.isConnecting).toBe(false);
    });
    
    test('has correct shape for error state', () => {
      const state: TerminalState = {
        isConnected: false,
        isConnecting: false,
        error: 'Connection failed',
        worktreeId: 'test-worktree'
      };
      
      expect(state.error).toBe('Connection failed');
    });
  });

  describe('Client Message Types', () => {
    test('ClientStdinMessage has correct shape', () => {
      const message: ClientStdinMessage = {
        type: 'stdin',
        data: 'ls -la\n'
      };
      
      expect(message.type).toBe('stdin');
      expect(message.data).toBe('ls -la\n');
    });
    
    test('ClientResizeMessage has correct shape', () => {
      const message: ClientResizeMessage = {
        type: 'resize',
        cols: 80,
        rows: 24
      };
      
      expect(message.type).toBe('resize');
      expect(message.cols).toBe(80);
      expect(message.rows).toBe(24);
    });
    
    test('ClientPingMessage has correct shape', () => {
      const message: ClientPingMessage = {
        type: 'ping'
      };
      
      expect(message.type).toBe('ping');
    });
    
    test('TerminalClientMessage union type works', () => {
      const stdinMessage: TerminalClientMessage = {
        type: 'stdin',
        data: 'test'
      };
      
      const resizeMessage: TerminalClientMessage = {
        type: 'resize',
        cols: 100,
        rows: 30
      };
      
      const pingMessage: TerminalClientMessage = {
        type: 'ping'
      };
      
      expect(stdinMessage.type).toBe('stdin');
      expect(resizeMessage.type).toBe('resize');
      expect(pingMessage.type).toBe('ping');
    });
  });

  describe('Server Message Types', () => {
    test('ServerStdoutMessage has correct shape', () => {
      const message: ServerStdoutMessage = {
        type: 'stdout',
        data: 'Hello from terminal'
      };
      
      expect(message.type).toBe('stdout');
      expect(message.data).toBe('Hello from terminal');
    });
    
    test('ServerStderrMessage has correct shape', () => {
      const message: ServerStderrMessage = {
        type: 'stderr',
        data: 'Error occurred'
      };
      
      expect(message.type).toBe('stderr');
      expect(message.data).toBe('Error occurred');
    });
    
    test('ServerExitMessage has correct shape', () => {
      const message: ServerExitMessage = {
        type: 'exit',
        exitCode: 0
      };
      
      expect(message.type).toBe('exit');
      expect(message.exitCode).toBe(0);
    });
    
    test('ServerPongMessage has correct shape', () => {
      const message: ServerPongMessage = {
        type: 'pong'
      };
      
      expect(message.type).toBe('pong');
    });
    
    test('TerminalServerMessage union type works', () => {
      const stdoutMessage: TerminalServerMessage = {
        type: 'stdout',
        data: 'output'
      };
      
      const stderrMessage: TerminalServerMessage = {
        type: 'stderr',
        data: 'error'
      };
      
      const exitMessage: TerminalServerMessage = {
        type: 'exit',
        exitCode: 1
      };
      
      const pongMessage: TerminalServerMessage = {
        type: 'pong'
      };
      
      expect(stdoutMessage.type).toBe('stdout');
      expect(stderrMessage.type).toBe('stderr');
      expect(exitMessage.type).toBe('exit');
      expect(pongMessage.type).toBe('pong');
    });
  });

  describe('Message Protocol Validation', () => {
    test('stdin message can be serialized and parsed', () => {
      const original: ClientStdinMessage = {
        type: 'stdin',
        data: 'echo "Hello World"'
      };
      
      const serialized = JSON.stringify(original);
      const parsed = JSON.parse(serialized) as ClientStdinMessage;
      
      expect(parsed.type).toBe(original.type);
      expect(parsed.data).toBe(original.data);
    });
    
    test('resize message can be serialized and parsed', () => {
      const original: ClientResizeMessage = {
        type: 'resize',
        cols: 120,
        rows: 30
      };
      
      const serialized = JSON.stringify(original);
      const parsed = JSON.parse(serialized) as ClientResizeMessage;
      
      expect(parsed.type).toBe(original.type);
      expect(parsed.cols).toBe(original.cols);
      expect(parsed.rows).toBe(original.rows);
    });
    
    test('stdout message can be serialized and parsed', () => {
      const original: ServerStdoutMessage = {
        type: 'stdout',
        data: 'Command output with special chars: \n\t\r'
      };
      
      const serialized = JSON.stringify(original);
      const parsed = JSON.parse(serialized) as ServerStdoutMessage;
      
      expect(parsed.type).toBe(original.type);
      expect(parsed.data).toBe(original.data);
    });
    
    test('exit message can be serialized and parsed', () => {
      const original: ServerExitMessage = {
        type: 'exit',
        exitCode: 127
      };
      
      const serialized = JSON.stringify(original);
      const parsed = JSON.parse(serialized) as ServerExitMessage;
      
      expect(parsed.type).toBe(original.type);
      expect(parsed.exitCode).toBe(original.exitCode);
    });
  });

  describe('Type Discrimination', () => {
    test('can discriminate client message types', () => {
      const messages: TerminalClientMessage[] = [
        { type: 'stdin', data: 'test' },
        { type: 'resize', cols: 80, rows: 24 },
        { type: 'ping' }
      ];
      
      messages.forEach(message => {
        switch (message.type) {
          case 'stdin':
            expect(message.data).toBeDefined();
            expect('cols' in message).toBe(false);
            break;
          case 'resize':
            expect(message.cols).toBeDefined();
            expect(message.rows).toBeDefined();
            expect('data' in message).toBe(false);
            break;
          case 'ping':
            expect('data' in message).toBe(false);
            expect('cols' in message).toBe(false);
            break;
        }
      });
    });
    
    test('can discriminate server message types', () => {
      const messages: TerminalServerMessage[] = [
        { type: 'stdout', data: 'output' },
        { type: 'stderr', data: 'error' },
        { type: 'exit', exitCode: 0 },
        { type: 'pong' }
      ];
      
      messages.forEach(message => {
        switch (message.type) {
          case 'stdout':
          case 'stderr':
            expect(message.data).toBeDefined();
            expect('exitCode' in message).toBe(false);
            break;
          case 'exit':
            expect(message.exitCode).toBeDefined();
            expect('data' in message).toBe(false);
            break;
          case 'pong':
            expect('data' in message).toBe(false);
            expect('exitCode' in message).toBe(false);
            break;
        }
      });
    });
  });

  describe('Edge Cases', () => {
    test('handles empty data strings', () => {
      const message: ServerStdoutMessage = {
        type: 'stdout',
        data: ''
      };
      
      expect(message.data).toBe('');
      expect(message.type).toBe('stdout');
    });
    
    test('handles zero exit code', () => {
      const message: ServerExitMessage = {
        type: 'exit',
        exitCode: 0
      };
      
      expect(message.exitCode).toBe(0);
    });
    
    test('handles negative exit codes', () => {
      const message: ServerExitMessage = {
        type: 'exit',
        exitCode: -1
      };
      
      expect(message.exitCode).toBe(-1);
    });
    
    test('handles very small terminal dimensions', () => {
      const message: ClientResizeMessage = {
        type: 'resize',
        cols: 1,
        rows: 1
      };
      
      expect(message.cols).toBe(1);
      expect(message.rows).toBe(1);
    });
    
    test('handles very large terminal dimensions', () => {
      const message: ClientResizeMessage = {
        type: 'resize',
        cols: 999,
        rows: 999
      };
      
      expect(message.cols).toBe(999);
      expect(message.rows).toBe(999);
    });
  });
});