# Terminal Components

This directory contains the terminal functionality for the Vibeman Web UI.

## Components

### Terminal
The main terminal component built with xterm.js. Features:
- Dark/light theme support that follows the app theme
- WebSocket connection management
- Terminal resizing
- Connection status indicators
- Error handling and reconnection logic

### TerminalModal
A modal wrapper for the Terminal component that provides:
- Fullscreen toggle
- Modal overlay with backdrop blur
- Window-like controls
- Smooth animations

### InlineTerminal
A card-wrapped version for embedded terminal views:
- Fixed height container
- Card styling that matches the app design
- Optional title

## Hook

### useTerminal
Custom hook for managing WebSocket terminal connections:
- Connection state management
- Auto-reconnection with exponential backoff
- Message handling (input/output/resize/error)
- Cleanup on unmount

## Usage

```tsx
import { TerminalModal } from './components/terminal';

function MyComponent() {
  const [isTerminalOpen, setIsTerminalOpen] = useState(false);
  
  return (
    <>
      <button onClick={() => setIsTerminalOpen(true)}>
        Open Terminal
      </button>
      
      <TerminalModal
        open={isTerminalOpen}
        onOpenChange={setIsTerminalOpen}
        environmentId="env-123"
        worktreeName="my-feature"
      />
    </>
  );
}
```

## WebSocket Protocol

The terminal communicates with the backend via WebSocket at `/api/environments/:id/terminal`.

Expected message format:
```json
{
  "type": "input" | "output" | "resize" | "error" | "connect" | "disconnect",
  "data": "string",
  "cols": 80,
  "rows": 24
}
```

## Features

- ✅ Real-time terminal communication
- ✅ Theme-aware styling (dark/light mode)
- ✅ Terminal resizing
- ✅ Connection status indicators
- ✅ Auto-reconnection
- ✅ Error handling
- ✅ Fullscreen support
- ✅ TypeScript support
- ✅ Proper cleanup
- ✅ Mobile-friendly design