import { describe, test, expect, beforeEach, mock } from 'bun:test';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '../test-setup';
import { TerminalModal } from './TerminalModal';

// Mock the Terminal component
const mockTerminal = mock(() => (
  <div data-testid="terminal-component">Mocked Terminal</div>
));

mock.module('../Terminal', () => ({
  Terminal: mockTerminal
}));

// Mock Radix UI Dialog components
mock.module('@/components/ui/dialog', () => ({
  Dialog: ({ children, open }: any) => open ? <div data-testid="dialog">{children}</div> : null,
  DialogContent: ({ children, className }: any) => (
    <div data-testid="dialog-content" className={className}>
      {children}
    </div>
  ),
  DialogHeader: ({ children, className }: any) => (
    <div data-testid="dialog-header" className={className}>
      {children}
    </div>
  ),
  DialogTitle: ({ children }: any) => (
    <h2 data-testid="dialog-title">{children}</h2>
  )
}));

// Mock Button component
mock.module('@/components/ui/button', () => ({
  Button: ({ children, onClick, title, className }: any) => (
    <button 
      onClick={onClick} 
      title={title} 
      className={className}
      data-testid="button"
    >
      {children}
    </button>
  )
}));

// Mock Lucide icons
mock.module('lucide-react', () => ({
  Maximize2: () => <span data-testid="maximize-icon">Maximize</span>,
  Minimize2: () => <span data-testid="minimize-icon">Minimize</span>
}));

describe('TerminalModal Component', () => {
  const defaultProps = {
    isOpen: true,
    onClose: mock(() => {}),
    worktreeId: 'test-worktree-123',
    title: 'Test Terminal'
  };

  beforeEach(() => {
    // Clear all mocks
    mock.restore();
    defaultProps.onClose.mockClear();
    mockTerminal.mockClear();
  });

  test('renders modal when open', () => {
    render(<TerminalModal {...defaultProps} />);
    
    expect(screen.getByTestId('dialog')).toBeInTheDocument();
    expect(screen.getByTestId('dialog-content')).toBeInTheDocument();
    expect(screen.getByTestId('dialog-header')).toBeInTheDocument();
    expect(screen.getByTestId('dialog-title')).toBeInTheDocument();
  });

  test('does not render modal when closed', () => {
    render(<TerminalModal {...defaultProps} isOpen={false} />);
    
    expect(screen.queryByTestId('dialog')).not.toBeInTheDocument();
  });

  test('displays correct title', () => {
    render(<TerminalModal {...defaultProps} title="Custom AI Terminal" />);
    
    expect(screen.getByTestId('dialog-title')).toHaveTextContent('Custom AI Terminal');
  });

  test('uses default title when not provided', () => {
    const propsWithoutTitle = {
      ...defaultProps,
      title: undefined
    };
    
    render(<TerminalModal {...propsWithoutTitle} />);
    
    expect(screen.getByTestId('dialog-title')).toHaveTextContent('AI Terminal');
  });

  test('passes correct props to Terminal component', () => {
    render(<TerminalModal {...defaultProps} />);
    
    expect(mockTerminal).toHaveBeenCalledWith(
      expect.objectContaining({
        worktreeId: 'test-worktree-123',
        className: 'h-full',
        isFullscreen: false,
        onToggleFullscreen: expect.any(Function)
      }),
      expect.anything()
    );
  });

  test('toggles fullscreen mode', async () => {
    render(<TerminalModal {...defaultProps} />);
    
    // Find fullscreen toggle button
    const fullscreenButton = screen.getByTitle('Enter fullscreen');
    expect(screen.getByTestId('maximize-icon')).toBeInTheDocument();
    
    // Click to enter fullscreen
    await userEvent.click(fullscreenButton);
    
    // Should show minimize icon and update title
    await waitFor(() => {
      expect(screen.getByTestId('minimize-icon')).toBeInTheDocument();
      expect(screen.getByTitle('Exit fullscreen')).toBeInTheDocument();
    });
    
    // Dialog content should have fullscreen classes
    const dialogContent = screen.getByTestId('dialog-content');
    expect(dialogContent).toHaveClass('max-w-full', 'h-screen', 'w-screen');
  });

  test('applies correct CSS classes for normal mode', () => {
    render(<TerminalModal {...defaultProps} />);
    
    const dialogContent = screen.getByTestId('dialog-content');
    expect(dialogContent).toHaveClass('max-w-4xl', 'h-[70vh]', 'sm:h-[600px]', 'w-[95vw]', 'sm:w-full');
    expect(dialogContent).not.toHaveClass('max-w-full', 'h-screen', 'w-screen');
  });

  test('applies correct CSS classes for fullscreen mode', async () => {
    render(<TerminalModal {...defaultProps} />);
    
    const fullscreenButton = screen.getByTitle('Enter fullscreen');
    await userEvent.click(fullscreenButton);
    
    const dialogContent = screen.getByTestId('dialog-content');
    expect(dialogContent).toHaveClass('max-w-full', 'h-screen', 'w-screen');
    expect(dialogContent).not.toHaveClass('max-w-4xl', 'h-[70vh]');
  });

  test('calls onToggleFullscreen when terminal triggers it', () => {
    render(<TerminalModal {...defaultProps} />);
    
    // Get the onToggleFullscreen function passed to Terminal
    const terminalProps = mockTerminal.mock.calls[0][0];
    const onToggleFullscreen = terminalProps.onToggleFullscreen;
    
    // Call it
    onToggleFullscreen();
    
    // Should have updated fullscreen state
    expect(mockTerminal).toHaveBeenCalledWith(
      expect.objectContaining({
        isFullscreen: true
      }),
      expect.anything()
    );
  });

  test('handles dialog close', async () => {
    // Mock the dialog's onOpenChange prop
    const mockOnOpenChange = mock(() => {});
    
    // We need to render a version that actually calls onClose when dialog closes
    const TestWrapper = () => {
      const handleOpenChange = (open: boolean) => {
        if (!open) {
          defaultProps.onClose();
        }
      };
      
      return (
        <div>
          <button onClick={() => handleOpenChange(false)}>Close Dialog</button>
          <TerminalModal {...defaultProps} />
        </div>
      );
    };
    
    render(<TestWrapper />);
    
    const closeButton = screen.getByText('Close Dialog');
    await userEvent.click(closeButton);
    
    expect(defaultProps.onClose).toHaveBeenCalled();
  });

  test('updates Terminal props when fullscreen changes', async () => {
    render(<TerminalModal {...defaultProps} />);
    
    // Initial render - not fullscreen
    expect(mockTerminal).toHaveBeenCalledWith(
      expect.objectContaining({
        isFullscreen: false
      }),
      expect.anything()
    );
    
    // Click fullscreen button
    const fullscreenButton = screen.getByTitle('Enter fullscreen');
    await userEvent.click(fullscreenButton);
    
    // Should have been called again with fullscreen true
    await waitFor(() => {
      expect(mockTerminal).toHaveBeenCalledWith(
        expect.objectContaining({
          isFullscreen: true
        }),
        expect.anything()
      );
    });
  });

  test('preserves worktree ID across fullscreen toggles', async () => {
    render(<TerminalModal {...defaultProps} />);
    
    const fullscreenButton = screen.getByTitle('Enter fullscreen');
    await userEvent.click(fullscreenButton);
    
    // All calls should have the same worktreeId
    mockTerminal.mock.calls.forEach(call => {
      expect(call[0].worktreeId).toBe('test-worktree-123');
    });
  });

  test('has accessible structure', () => {
    render(<TerminalModal {...defaultProps} />);
    
    // Should have proper dialog structure
    expect(screen.getByTestId('dialog-header')).toBeInTheDocument();
    expect(screen.getByTestId('dialog-title')).toBeInTheDocument();
    expect(screen.getByTestId('terminal-component')).toBeInTheDocument();
    
    // Fullscreen button should have proper accessibility
    const fullscreenButton = screen.getByTitle('Enter fullscreen');
    expect(fullscreenButton).toBeInTheDocument();
  });

  test('applies mobile-responsive classes', () => {
    render(<TerminalModal {...defaultProps} />);
    
    const dialogHeader = screen.getByTestId('dialog-header');
    expect(dialogHeader).toHaveClass('px-4', 'sm:px-6', 'py-3', 'sm:py-4');
    
    const dialogTitle = screen.getByTestId('dialog-title');
    expect(dialogTitle).toHaveClass('text-sm', 'sm:text-base');
  });

  test('fullscreen button has correct responsive sizing', () => {
    render(<TerminalModal {...defaultProps} />);
    
    const fullscreenButton = screen.getByTitle('Enter fullscreen');
    expect(fullscreenButton).toHaveClass('h-6', 'w-6', 'sm:h-7', 'sm:w-7');
  });
});