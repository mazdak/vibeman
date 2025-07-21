import React, { useEffect, useRef, useState } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { useTerminal } from '../hooks/useTerminal';
import type { TerminalProps } from '../types/terminal';
import { 
  AlertCircle, 
  Wifi, 
  WifiOff, 
  RefreshCw, 
  X,
  Maximize2,
  Minimize2
} from 'lucide-react';
import { Button } from './ui/button';

// Import xterm.js CSS
import '@xterm/xterm/css/xterm.css';

interface TerminalComponentProps extends TerminalProps {
  isFullscreen?: boolean;
  onToggleFullscreen?: () => void;
}

export const Terminal: React.FC<TerminalComponentProps> = ({
  worktreeId,
  onClose,
  className = '',
  isFullscreen = false,
  onToggleFullscreen
}) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const [isInitialized, setIsInitialized] = useState(false);

  // Get system theme preference
  const [isDarkMode, setIsDarkMode] = useState(() => {
    if (typeof window !== 'undefined') {
      return document.documentElement.classList.contains('dark');
    }
    return false;
  });

  // Monitor theme changes
  useEffect(() => {
    const observer = new MutationObserver(() => {
      setIsDarkMode(document.documentElement.classList.contains('dark'));
    });

    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class']
    });

    return () => observer.disconnect();
  }, []);

  // Terminal theme configuration
  const getTheme = () => ({
    background: isDarkMode ? '#0f172a' : '#ffffff',
    foreground: isDarkMode ? '#e2e8f0' : '#334155',
    cursor: isDarkMode ? '#06b6d4' : '#8b5cf6',
    cursorAccent: isDarkMode ? '#0f172a' : '#ffffff',
    selection: isDarkMode ? '#334155' : '#e2e8f0',
    black: isDarkMode ? '#0f172a' : '#334155',
    red: '#ef4444',
    green: '#10b981',
    yellow: '#f59e0b',
    blue: '#3b82f6',
    magenta: '#8b5cf6',
    cyan: '#06b6d4',
    white: isDarkMode ? '#f1f5f9' : '#64748b',
    brightBlack: isDarkMode ? '#475569' : '#94a3b8',
    brightRed: '#f87171',
    brightGreen: '#34d399',
    brightYellow: '#fbbf24',
    brightBlue: '#60a5fa',
    brightMagenta: '#a78bfa',
    brightCyan: '#22d3ee',
    brightWhite: isDarkMode ? '#ffffff' : '#0f172a'
  });

  // Terminal WebSocket hook
  const {
    state,
    connect,
    disconnect,
    sendInput,
    resize,
    isReady
  } = useTerminal({
    worktreeId: worktreeId,
    onOutput: (data) => {
      if (xtermRef.current) {
        xtermRef.current.write(data);
      }
    },
    onError: (error) => {
      if (xtermRef.current) {
        xtermRef.current.write(`\r\n\x1b[31mError: ${error}\x1b[0m\r\n`);
      }
    },
    onConnect: () => {
      if (xtermRef.current) {
        xtermRef.current.write('\x1b[32mConnected to terminal\x1b[0m\r\n');
        xtermRef.current.focus();
      }
    },
    onDisconnect: () => {
      if (xtermRef.current) {
        xtermRef.current.write('\r\n\x1b[33mDisconnected from terminal\x1b[0m\r\n');
      }
    }
  });

  // Get mobile-friendly settings
  const isMobile = () => {
    if (typeof window === 'undefined') return false;
    return window.innerWidth < 640; // sm breakpoint
  };

  // Initialize terminal
  useEffect(() => {
    if (!terminalRef.current || isInitialized) return;

    const mobile = isMobile();
    
    const terminal = new XTerm({
      theme: getTheme(),
      fontSize: mobile ? 12 : 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      cursorBlink: true,
      cursorStyle: 'block',
      allowTransparency: false,
      convertEol: true,
      scrollback: mobile ? 500 : 1000, // Reduce scrollback on mobile for performance
      tabStopWidth: 4,
      // Mobile-specific settings
      disableStdin: false,
      macOptionIsMeta: true,
      rightClickSelectsWord: !mobile, // Disable on mobile to prevent conflicts with touch
      fastScrollModifier: mobile ? undefined : 'shift' // Disable fast scroll on mobile
    });

    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();

    terminal.loadAddon(fitAddon);
    terminal.loadAddon(webLinksAddon);

    terminal.open(terminalRef.current);
    terminal.write('Vibeman AI Terminal\r\n\x1b[90mConnecting to AI container...\x1b[0m\r\n');

    // Handle terminal input
    terminal.onData((data) => {
      if (isReady) {
        sendInput(data);
      }
    });

    // Handle terminal resize
    terminal.onResize(({ cols, rows }) => {
      if (isReady) {
        resize(cols, rows);
      }
    });

    xtermRef.current = terminal;
    fitAddonRef.current = fitAddon;
    setIsInitialized(true);

    // Fit terminal to container
    setTimeout(() => {
      fitAddon.fit();
    }, 100);

    return () => {
      terminal.dispose();
      xtermRef.current = null;
      fitAddonRef.current = null;
      setIsInitialized(false);
    };
  }, [isInitialized, isReady, sendInput, resize]);

  // Update theme when it changes
  useEffect(() => {
    if (xtermRef.current) {
      xtermRef.current.options.theme = getTheme();
    }
  }, [isDarkMode]);

  // Handle window resize
  useEffect(() => {
    const handleResize = () => {
      if (fitAddonRef.current) {
        setTimeout(() => {
          fitAddonRef.current?.fit();
        }, 100);
      }
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  // Fit terminal when fullscreen changes
  useEffect(() => {
    if (fitAddonRef.current) {
      setTimeout(() => {
        fitAddonRef.current?.fit();
      }, 150);
    }
  }, [isFullscreen]);

  const getConnectionStatusIcon = () => {
    if (state.isConnecting) {
      return <RefreshCw className="w-4 h-4 animate-spin text-blue-500" />;
    }
    if (state.isConnected) {
      return <Wifi className="w-4 h-4 text-green-500" />;
    }
    return <WifiOff className="w-4 h-4 text-red-500" />;
  };

  const getConnectionStatusText = () => {
    if (state.isConnecting) return 'Connecting...';
    if (state.isConnected) return 'Connected';
    if (state.error) return state.error;
    return 'Disconnected';
  };

  const getConnectionStatusColor = () => {
    if (state.isConnecting) return 'text-blue-600 dark:text-blue-400';
    if (state.isConnected) return 'text-green-600 dark:text-green-400';
    return 'text-red-600 dark:text-red-400';
  };

  return (
    <div className={`flex flex-col bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden ${className}`}>
      {/* Terminal Header */}
      <div className="flex items-center justify-between px-2 sm:px-4 py-2 bg-slate-50 dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="flex items-center gap-2 sm:gap-3 min-w-0 flex-1">
          <div className="flex items-center gap-1 sm:gap-2 min-w-0">
            {getConnectionStatusIcon()}
            <span className={`text-xs sm:text-sm font-medium truncate ${getConnectionStatusColor()}`}>
              {getConnectionStatusText()}
            </span>
          </div>
          {state.error && (
            <div className="flex items-center gap-1 text-red-600 dark:text-red-400">
              <AlertCircle className="w-3 h-3 sm:w-4 sm:h-4 flex-shrink-0" />
              <span className="text-xs sm:text-sm hidden sm:inline">Connection Error</span>
            </div>
          )}
        </div>

        <div className="flex items-center gap-1 sm:gap-2 flex-shrink-0">
          {!state.isConnected && !state.isConnecting && (
            <Button
              variant="outline"
              size="sm"
              onClick={connect}
              className="h-6 sm:h-7 px-1 sm:px-2 text-xs"
            >
              <RefreshCw className="w-3 h-3 sm:mr-1" />
              <span className="hidden sm:inline">Connect</span>
            </Button>
          )}
          
          {onToggleFullscreen && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onToggleFullscreen}
              className="h-6 w-6 sm:h-7 sm:w-7 p-0"
              title={isFullscreen ? "Exit fullscreen" : "Enter fullscreen"}
            >
              {isFullscreen ? (
                <Minimize2 className="w-3 h-3" />
              ) : (
                <Maximize2 className="w-3 h-3" />
              )}
            </Button>
          )}
          
          {onClose && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onClose}
              className="h-6 w-6 sm:h-7 sm:w-7 p-0 text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200"
              title="Close terminal"
            >
              <X className="w-3 h-3" />
            </Button>
          )}
        </div>
      </div>

      {/* Terminal Content */}
      <div className="flex-1 relative">
        <div
          ref={terminalRef}
          className="absolute inset-0 p-1 sm:p-2"
          style={{ 
            minHeight: isFullscreen ? '80vh' : '300px',
            // Ensure proper touch handling on mobile
            touchAction: 'none'
          }}
        />
      </div>
    </div>
  );
};

export default Terminal;