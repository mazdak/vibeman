import { describe, test, expect, beforeEach, afterEach, mock, spyOn } from 'bun:test';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '../test-setup';
import { Terminal } from './Terminal';
import type { Terminal as XTerminal } from '@xterm/xterm';

// Mock xterm.js
const mockTerminal = {
  dispose: mock(() => {}),
  open: mock(() => {}),
  onData: mock(() => ({ dispose: mock(() => {}) })),
  onResize: mock(() => ({ dispose: mock(() => {}) })),
  resize: mock(() => {}),
  write: mock(() => {}),
  writeln: mock(() => {}),
  focus: mock(() => {}),
  blur: mock(() => {}),
  clear: mock(() => {}),
  loadAddon: mock(() => {}), // Add missing loadAddon method
  rows: 24,
  cols: 80,
  element: document.createElement('div'),
  textarea: document.createElement('textarea'),
  _core: {
    viewport: {
      onRequestScrollToBottom: {
        fire: mock(() => {})
      }
    }
  }
} as unknown as XTerminal;

const mockFitAddon = {
  activate: mock(() => {}),
  dispose: mock(() => {}),
  fit: mock(() => {}),
  proposeDimensions: mock(() => ({ cols: 80, rows: 24 }))
};

const mockWebLinksAddon = {
  activate: mock(() => {}),
  dispose: mock(() => {})
};

// Mock the xterm imports
mock.module('@xterm/xterm', () => ({
  Terminal: mock(() => mockTerminal)
}));

mock.module('@xterm/addon-fit', () => ({
  FitAddon: mock(() => mockFitAddon)
}));

mock.module('@xterm/addon-web-links', () => ({
  WebLinksAddon: mock(() => mockWebLinksAddon)
}));

// Mock useTerminal hook
const mockUseTerminal = {
  state: {
    isConnected: false,
    isConnecting: false,
    error: null,
    worktreeId: 'test-worktree'
  },
  connect: mock(() => {}),
  disconnect: mock(() => {}),
  sendInput: mock(() => {}),
  resize: mock(() => {}),
  isReady: false
};

mock.module('../hooks/useTerminal', () => ({
  useTerminal: mock(() => mockUseTerminal)
}));

// Mock useMounted hook
mock.module('../shared/hooks', () => ({
  useMounted: mock(() => ({ current: true }))
}));

describe('Terminal Component', () => {
  beforeEach(() => {
    // Reset all mocks
    mock.restore();
    
    // Reset mock terminal state
    mockUseTerminal.state = {
      isConnected: false,
      isConnecting: false,
      error: null,
      worktreeId: 'test-worktree'
    };
    mockUseTerminal.isReady = false;
    
    // Clear mock call history
    Object.values(mockTerminal).forEach(fn => {
      if (typeof fn === 'function' && 'mockClear' in fn) {
        fn.mockClear();
      }
    });
    
    Object.values(mockFitAddon).forEach(fn => {
      if (typeof fn === 'function' && 'mockClear' in fn) {
        fn.mockClear();
      }
    });
    
    Object.values(mockUseTerminal).forEach(fn => {
      if (typeof fn === 'function' && 'mockClear' in fn) {
        fn.mockClear();
      }
    });
  });

  afterEach(() => {
    mock.restore();
  });

  test('renders terminal container with loading state', () => {
    mockUseTerminal.state.isConnecting = true;
    
    render(<Terminal worktreeId="test-worktree" />);
    
    expect(screen.getByText('Connecting to AI terminal...')).toBeInTheDocument();
    expect(screen.getByRole('status')).toBeInTheDocument();
  });

  test('renders terminal container with error state', () => {
    mockUseTerminal.state.error = 'Connection failed';
    
    render(<Terminal worktreeId="test-worktree" />);
    
    expect(screen.getByText('Connection failed')).toBeInTheDocument();
    expect(screen.getByText('Retry')).toBeInTheDocument();
  });

  test('renders connected terminal', async () => {
    mockUseTerminal.state.isConnected = true;
    mockUseTerminal.isReady = true;
    
    render(<Terminal worktreeId="test-worktree" />);
    
    await waitFor(() => {
      expect(mockTerminal.open).toHaveBeenCalled();
    });
    
    expect(screen.getByRole('button', { name: /clear/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /disconnect/i })).toBeInTheDocument();
  });

  test('calls connect on mount when autoConnect is true', () => {
    render(<Terminal worktreeId="test-worktree" autoConnect />);
    
    expect(mockUseTerminal.connect).toHaveBeenCalled();
  });

  test('does not call connect on mount when autoConnect is false', () => {
    render(<Terminal worktreeId="test-worktree" autoConnect={false} />);
    
    expect(mockUseTerminal.connect).not.toHaveBeenCalled();
  });

  test('handles retry button click', async () => {
    mockUseTerminal.state.error = 'Connection failed';
    
    render(<Terminal worktreeId="test-worktree" />);
    
    const retryButton = screen.getByText('Retry');
    await userEvent.click(retryButton);
    
    expect(mockUseTerminal.connect).toHaveBeenCalled();
  });

  test('handles clear button click', async () => {
    mockUseTerminal.state.isConnected = true;
    mockUseTerminal.isReady = true;
    
    render(<Terminal worktreeId="test-worktree" />);
    
    await waitFor(() => {
      expect(mockTerminal.open).toHaveBeenCalled();
    });
    
    const clearButton = screen.getByRole('button', { name: /clear/i });
    await userEvent.click(clearButton);
    
    expect(mockTerminal.clear).toHaveBeenCalled();
  });

  test('handles disconnect button click', async () => {
    mockUseTerminal.state.isConnected = true;
    mockUseTerminal.isReady = true;
    
    render(<Terminal worktreeId="test-worktree" />);
    
    await waitFor(() => {
      expect(mockTerminal.open).toHaveBeenCalled();
    });
    
    const disconnectButton = screen.getByRole('button', { name: /disconnect/i });
    await userEvent.click(disconnectButton);
    
    expect(mockUseTerminal.disconnect).toHaveBeenCalled();
  });

  test('handles fullscreen toggle', async () => {
    const onToggleFullscreen = mock(() => {});
    mockUseTerminal.state.isConnected = true;
    mockUseTerminal.isReady = true;
    
    render(
      <Terminal 
        worktreeId="test-worktree" 
        onToggleFullscreen={onToggleFullscreen} 
      />
    );
    
    await waitFor(() => {
      expect(mockTerminal.open).toHaveBeenCalled();
    });
    
    const fullscreenButton = screen.getByRole('button', { name: /fullscreen/i });
    await userEvent.click(fullscreenButton);
    
    expect(onToggleFullscreen).toHaveBeenCalled();
  });

  test('applies mobile-specific settings', () => {
    // Mock window.innerWidth to simulate mobile
    const originalInnerWidth = window.innerWidth;
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: 500
    });
    
    mockUseTerminal.state.isConnected = true;
    mockUseTerminal.isReady = true;
    
    render(<Terminal worktreeId="test-worktree" />);
    
    // Verify terminal was created (mobile settings are applied during creation)
    expect(mockTerminal.open).toHaveBeenCalled();
    
    // Restore original width
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: originalInnerWidth
    });
  });

  test('handles resize events', async () => {
    mockUseTerminal.state.isConnected = true;
    mockUseTerminal.isReady = true;
    
    render(<Terminal worktreeId="test-worktree" />);
    
    await waitFor(() => {
      expect(mockTerminal.open).toHaveBeenCalled();
    });
    
    // Simulate window resize
    fireEvent(window, new Event('resize'));
    
    await waitFor(() => {
      expect(mockFitAddon.fit).toHaveBeenCalled();
    });
  });

  test('cleans up resources on unmount', () => {
    mockUseTerminal.state.isConnected = true;
    mockUseTerminal.isReady = true;
    
    const { unmount } = render(<Terminal worktreeId="test-worktree" />);
    
    unmount();
    
    expect(mockUseTerminal.disconnect).toHaveBeenCalled();
    expect(mockTerminal.dispose).toHaveBeenCalled();
  });

  test('handles worktree ID changes', () => {
    const { rerender } = render(<Terminal worktreeId="test-worktree-1" />);
    
    // Change worktree ID
    rerender(<Terminal worktreeId="test-worktree-2" />);
    
    // Should disconnect and reconnect with new worktree ID
    expect(mockUseTerminal.disconnect).toHaveBeenCalled();
  });

  test('supports custom CSS classes', () => {
    const { container } = render(
      <Terminal worktreeId="test-worktree" className="custom-terminal" />
    );
    
    expect(container.firstChild).toHaveClass('custom-terminal');
  });

  test('shows connection status correctly', () => {
    // Test connecting state
    mockUseTerminal.state.isConnecting = true;
    const { rerender } = render(<Terminal worktreeId="test-worktree" />);
    expect(screen.getByText('Connecting to AI terminal...')).toBeInTheDocument();
    
    // Test connected state
    mockUseTerminal.state.isConnecting = false;
    mockUseTerminal.state.isConnected = true;
    mockUseTerminal.isReady = true;
    rerender(<Terminal worktreeId="test-worktree" />);
    expect(screen.queryByText('Connecting to AI terminal...')).not.toBeInTheDocument();
    
    // Test error state
    mockUseTerminal.state.isConnected = false;
    mockUseTerminal.state.error = 'Connection timeout';
    mockUseTerminal.isReady = false;
    rerender(<Terminal worktreeId="test-worktree" />);
    expect(screen.getByText('Connection timeout')).toBeInTheDocument();
  });

  test('handles keyboard shortcuts', async () => {
    mockUseTerminal.state.isConnected = true;
    mockUseTerminal.isReady = true;
    
    render(<Terminal worktreeId="test-worktree" />);
    
    await waitFor(() => {
      expect(mockTerminal.open).toHaveBeenCalled();
    });
    
    const container = screen.getByRole('region', { name: /terminal/i });
    
    // Test Ctrl+C (should send interrupt signal)
    await userEvent.keyboard('{Control>}c{/Control}');
    
    // Test Ctrl+L (should clear terminal)
    await userEvent.keyboard('{Control>}l{/Control}');
    
    // Verify terminal interactions
    expect(mockTerminal.focus).toHaveBeenCalled();
  });
});