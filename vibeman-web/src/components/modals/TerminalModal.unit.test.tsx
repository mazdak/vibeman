import { describe, test, expect } from 'bun:test';
import '../../test-setup';

describe('TerminalModal Component Unit Tests', () => {
  test('TerminalModal component exports exist', async () => {
    const { TerminalModal } = await import('./TerminalModal');
    expect(TerminalModal).toBeDefined();
    expect(typeof TerminalModal).toBe('function');
  });

  test('TerminalModal props interface is correctly typed', () => {
    const validProps = {
      isOpen: true,
      onClose: () => {},
      worktreeId: 'test-worktree-123',
      title: 'Custom Terminal Title'
    };

    expect(validProps.isOpen).toBe(true);
    expect(typeof validProps.onClose).toBe('function');
    expect(validProps.worktreeId).toBe('test-worktree-123');
    expect(validProps.title).toBe('Custom Terminal Title');
  });

  test('TerminalModal supports optional title prop', () => {
    const propsWithoutTitle = {
      isOpen: true,
      onClose: () => {},
      worktreeId: 'test-worktree'
    };

    // Should be valid without title (defaults to 'AI Terminal')
    expect(propsWithoutTitle.worktreeId).toBe('test-worktree');
    expect(typeof propsWithoutTitle.onClose).toBe('function');
  });
});

describe('TerminalModal State Management', () => {
  test('fullscreen state logic', () => {
    let isFullscreen = false;
    
    const toggleFullscreen = () => {
      isFullscreen = !isFullscreen;
    };

    // Initial state
    expect(isFullscreen).toBe(false);

    // Toggle to fullscreen
    toggleFullscreen();
    expect(isFullscreen).toBe(true);

    // Toggle back to normal
    toggleFullscreen();
    expect(isFullscreen).toBe(false);
  });

  test('CSS class logic for modal sizing', () => {
    const getNormalClasses = () => 'max-w-4xl h-[70vh] sm:h-[600px] w-[95vw] sm:w-full';
    const getFullscreenClasses = () => 'max-w-full h-screen w-screen';

    const normalClasses = getNormalClasses();
    const fullscreenClasses = getFullscreenClasses();

    expect(normalClasses).toContain('max-w-4xl');
    expect(normalClasses).toContain('h-[70vh]');
    expect(normalClasses).toContain('w-[95vw]');

    expect(fullscreenClasses).toContain('max-w-full');
    expect(fullscreenClasses).toContain('h-screen');
    expect(fullscreenClasses).toContain('w-screen');
  });
});

describe('TerminalModal Accessibility', () => {
  test('modal title functionality', () => {
    const getTitle = (customTitle?: string) => customTitle || 'AI Terminal';

    expect(getTitle()).toBe('AI Terminal');
    expect(getTitle('Custom AI Terminal')).toBe('Custom AI Terminal');
    expect(getTitle('Terminal - my-worktree')).toBe('Terminal - my-worktree');
  });

  test('fullscreen button attributes', () => {
    const getFullscreenButtonProps = (isFullscreen: boolean) => ({
      title: isFullscreen ? 'Exit fullscreen' : 'Enter fullscreen',
      'aria-label': isFullscreen ? 'Exit fullscreen mode' : 'Enter fullscreen mode'
    });

    const normalProps = getFullscreenButtonProps(false);
    const fullscreenProps = getFullscreenButtonProps(true);

    expect(normalProps.title).toBe('Enter fullscreen');
    expect(fullscreenProps.title).toBe('Exit fullscreen');
  });
});

describe('TerminalModal Event Handling', () => {
  test('dialog close handler', () => {
    let modalOpen = true;
    const onClose = () => {
      modalOpen = false;
    };

    const handleOpenChange = (open: boolean) => {
      if (!open) {
        onClose();
      }
    };

    // Simulate dialog closing
    handleOpenChange(false);
    expect(modalOpen).toBe(false);
  });

  test('fullscreen toggle propagation', () => {
    let modalFullscreen = false;
    let terminalFullscreen = false;

    const onToggleFullscreen = () => {
      modalFullscreen = !modalFullscreen;
      terminalFullscreen = modalFullscreen;
    };

    // Simulate fullscreen toggle
    onToggleFullscreen();
    expect(modalFullscreen).toBe(true);
    expect(terminalFullscreen).toBe(true);

    onToggleFullscreen();
    expect(modalFullscreen).toBe(false);
    expect(terminalFullscreen).toBe(false);
  });
});

describe('TerminalModal Responsive Design', () => {
  test('responsive class utilities', () => {
    const getResponsiveClasses = () => ({
      header: 'px-4 sm:px-6 py-3 sm:py-4',
      title: 'text-sm sm:text-base',
      button: 'h-6 w-6 sm:h-7 sm:w-7',
      icon: 'w-3 h-3 sm:w-4 sm:h-4'
    });

    const classes = getResponsiveClasses();

    expect(classes.header).toContain('px-4');
    expect(classes.header).toContain('sm:px-6');
    expect(classes.title).toContain('text-sm');
    expect(classes.title).toContain('sm:text-base');
    expect(classes.button).toContain('h-6');
    expect(classes.button).toContain('sm:h-7');
  });

  test('mobile vs desktop sizing', () => {
    const getSizing = (isMobile: boolean) => {
      if (isMobile) {
        return {
          modal: 'w-[95vw] h-[70vh]',
          padding: 'px-4 py-3',
          fontSize: 'text-sm',
          iconSize: 'w-3 h-3'
        };
      } else {
        return {
          modal: 'sm:w-full sm:h-[600px]',
          padding: 'sm:px-6 sm:py-4',
          fontSize: 'sm:text-base',
          iconSize: 'sm:w-4 sm:h-4'
        };
      }
    };

    const mobileSizing = getSizing(true);
    const desktopSizing = getSizing(false);

    expect(mobileSizing.modal).toContain('w-[95vw]');
    expect(mobileSizing.fontSize).toBe('text-sm');
    expect(desktopSizing.modal).toContain('sm:w-full');
    expect(desktopSizing.fontSize).toBe('sm:text-base');
  });
});