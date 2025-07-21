import React, { useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Terminal } from '@/components/Terminal';
import { Button } from '@/components/ui/button';
import { Maximize2, Minimize2 } from 'lucide-react';

interface TerminalModalProps {
  isOpen: boolean;
  onClose: () => void;
  worktreeId: string;
  title?: string;
}

export function TerminalModal({ 
  isOpen, 
  onClose, 
  worktreeId, 
  title = 'AI Terminal' 
}: TerminalModalProps) {
  const [isFullscreen, setIsFullscreen] = useState(false);

  const toggleFullscreen = () => {
    setIsFullscreen(!isFullscreen);
  };

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent 
        className={`p-0 ${
          isFullscreen 
            ? 'max-w-full h-screen w-screen' 
            : 'max-w-4xl h-[70vh] sm:h-[600px] w-[95vw] sm:w-full'
        }`}
      >
        <DialogHeader className="px-4 sm:px-6 py-3 sm:py-4 border-b flex-row items-center justify-between space-y-0">
          <DialogTitle className="text-sm sm:text-base">{title}</DialogTitle>
          <Button
            variant="ghost"
            size="sm"
            onClick={toggleFullscreen}
            className="h-6 w-6 sm:h-7 sm:w-7 p-0"
            title={isFullscreen ? "Exit fullscreen" : "Enter fullscreen"}
          >
            {isFullscreen ? (
              <Minimize2 className="w-3 h-3 sm:w-4 sm:h-4" />
            ) : (
              <Maximize2 className="w-3 h-3 sm:w-4 sm:h-4" />
            )}
          </Button>
        </DialogHeader>
        <div className="flex-1 overflow-hidden">
          <Terminal
            worktreeId={worktreeId}
            className="h-full"
            isFullscreen={isFullscreen}
            onToggleFullscreen={toggleFullscreen}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}