import React from 'react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from './ui/dialog';
import { Button } from './ui/button';
import { AlertTriangle, GitBranch, AlertCircle } from 'lucide-react';

interface WorktreeStatus {
  hasUncommittedChanges: boolean;
  hasUnstagedFiles: boolean;
  hasUnpushedCommits: boolean;
}

interface RemoveWorktreeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  worktreeName: string;
  branchName: string;
  status?: WorktreeStatus;
  onConfirm: () => void;
  isRemoving?: boolean;
}

export function RemoveWorktreeDialog({ 
  open, 
  onOpenChange, 
  worktreeName,
  branchName,
  status,
  onConfirm,
  isRemoving = false
}: RemoveWorktreeDialogProps) {
  const hasWarnings = status && (
    status.hasUncommittedChanges || 
    status.hasUnstagedFiles || 
    status.hasUnpushedCommits
  );

  const handleConfirm = () => {
    onConfirm();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[450px]">
        <DialogHeader>
          <DialogTitle className="text-xl font-semibold flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-amber-500" />
            Remove Worktree
          </DialogTitle>
          <DialogDescription className="mt-2">
            Are you sure you want to remove this worktree?
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4 py-4">
          <div className="bg-slate-50 dark:bg-slate-900/50 rounded-lg p-4 space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <span className="text-slate-600 dark:text-slate-400">Worktree:</span>
              <span className="font-semibold text-slate-900 dark:text-slate-100">{worktreeName}</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <GitBranch className="w-3 h-3 text-slate-600 dark:text-slate-400" />
              <span className="text-slate-600 dark:text-slate-400">Branch:</span>
              <span className="font-mono text-slate-900 dark:text-slate-100">{branchName}</span>
            </div>
          </div>

          {hasWarnings && (
            <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-4 space-y-2">
              <div className="flex items-start gap-2">
                <AlertCircle className="w-5 h-5 text-amber-600 dark:text-amber-400 mt-0.5" />
                <div className="flex-1">
                  <p className="font-semibold text-amber-800 dark:text-amber-300 text-sm">
                    Warning: This worktree has unsaved work
                  </p>
                  <ul className="mt-2 space-y-1 text-sm text-amber-700 dark:text-amber-400">
                    {status?.hasUncommittedChanges && (
                      <li className="flex items-center gap-2">
                        <span className="w-1.5 h-1.5 bg-amber-600 dark:bg-amber-400 rounded-full" />
                        Uncommitted changes
                      </li>
                    )}
                    {status?.hasUnstagedFiles && (
                      <li className="flex items-center gap-2">
                        <span className="w-1.5 h-1.5 bg-amber-600 dark:bg-amber-400 rounded-full" />
                        Unstaged files
                      </li>
                    )}
                    {status?.hasUnpushedCommits && (
                      <li className="flex items-center gap-2">
                        <span className="w-1.5 h-1.5 bg-amber-600 dark:bg-amber-400 rounded-full" />
                        Unpushed commits
                      </li>
                    )}
                  </ul>
                </div>
              </div>
            </div>
          )}

          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
            <p className="text-sm text-red-700 dark:text-red-300">
              This action will:
            </p>
            <ul className="mt-2 space-y-1 text-sm text-red-600 dark:text-red-400">
              <li className="flex items-center gap-2">
                <span className="w-1.5 h-1.5 bg-red-600 dark:bg-red-400 rounded-full" />
                Stop any running containers
              </li>
              <li className="flex items-center gap-2">
                <span className="w-1.5 h-1.5 bg-red-600 dark:bg-red-400 rounded-full" />
                Remove the worktree directory
              </li>
              <li className="flex items-center gap-2">
                <span className="w-1.5 h-1.5 bg-red-600 dark:bg-red-400 rounded-full" />
                Delete the local branch (if not checked out elsewhere)
              </li>
            </ul>
          </div>
        </div>
        
        <DialogFooter className="flex gap-3">
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isRemoving}
          >
            Cancel
          </Button>
          <Button 
            onClick={handleConfirm}
            disabled={isRemoving}
            variant="destructive"
            className={hasWarnings ? "bg-red-600 hover:bg-red-700" : ""}
          >
            {isRemoving ? 'Removing...' : hasWarnings ? 'Remove Anyway' : 'Remove Worktree'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}