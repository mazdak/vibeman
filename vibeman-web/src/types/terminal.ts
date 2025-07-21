// Terminal WebSocket message types for Vibeman AI containers

// Client -> Server messages
export interface ClientMessage {
  type: 'stdin' | 'resize' | 'ping';
  data?: string;
  cols?: number;
  rows?: number;
}

export interface ClientStdinMessage extends ClientMessage {
  type: 'stdin';
  data: string;
}

export interface ClientResizeMessage extends ClientMessage {
  type: 'resize';
  cols: number;
  rows: number;
}

export interface ClientPingMessage extends ClientMessage {
  type: 'ping';
}

// Server -> Client messages
export interface ServerMessage {
  type: 'stdout' | 'stderr' | 'exit' | 'pong';
  data?: string;
  exitCode?: number;
}

export interface ServerStdoutMessage extends ServerMessage {
  type: 'stdout';
  data: string;
}

export interface ServerStderrMessage extends ServerMessage {
  type: 'stderr';
  data: string;
}

export interface ServerExitMessage extends ServerMessage {
  type: 'exit';
  exitCode?: number;
}

export interface ServerPongMessage extends ServerMessage {
  type: 'pong';
}

export type TerminalClientMessage = 
  | ClientStdinMessage 
  | ClientResizeMessage 
  | ClientPingMessage;

export type TerminalServerMessage = 
  | ServerStdoutMessage 
  | ServerStderrMessage 
  | ServerExitMessage 
  | ServerPongMessage;

export interface TerminalState {
  isConnected: boolean;
  isConnecting: boolean;
  error: string | null;
  worktreeId: string;
}

export interface TerminalProps {
  worktreeId: string;
  onClose?: () => void;
  className?: string;
}