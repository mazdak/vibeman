// Type declarations for xterm.js modules
declare module '@xterm/xterm/css/xterm.css';

// Re-export the main types from xterm.js for better IDE support
export type { Terminal, ITerminalOptions, ITheme } from '@xterm/xterm';
export type { FitAddon } from '@xterm/addon-fit';
export type { WebLinksAddon } from '@xterm/addon-web-links';