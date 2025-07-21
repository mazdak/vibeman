import React from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { LogViewer } from '@/components/logs/LogViewer';

interface LogModalProps {
  isOpen: boolean;
  onClose: () => void;
  worktreeId: string;
  title?: string;
}

export function LogModal({ isOpen, onClose, worktreeId, title = 'Logs' }: LogModalProps) {
  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-4xl h-[600px] p-0">
        <DialogHeader className="px-6 py-4 border-b">
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <div className="flex-1 overflow-hidden">
          <LogViewer
            environmentId={worktreeId}
            className="h-full"
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}