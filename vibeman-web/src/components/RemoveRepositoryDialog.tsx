import React from 'react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from './ui/dialog';
import { Button } from './ui/button';
import { AlertTriangle, Database, AlertCircle, GitBranch } from 'lucide-react';

interface RemoveRepositoryDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  repositoryName: string;
  repositoryPath: string;
  activeWorktrees: Array<{ id: string; name: string; branch: string }>;
  onConfirm: (removeType: 'untrack' | 'delete') => void;
  isRemoving?: boolean;
}

export function RemoveRepositoryDialog({ 
  open, 
  onOpenChange, 
  repositoryName,
  repositoryPath,
  activeWorktrees,
  onConfirm,
  isRemoving = false
}: RemoveRepositoryDialogProps) {
  const hasActiveWorktrees = activeWorktrees.length > 0;

  const handleUntrack = () => {
    onConfirm('untrack');
  };

  const handleDelete = () => {
    onConfirm('delete');
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="text-xl font-semibold flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-amber-500" />
            Remove Repository
          </DialogTitle>
          <DialogDescription className="mt-2">
            Choose how to remove this repository from Vibeman
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4 py-4">
          <div className="bg-slate-50 dark:bg-slate-900/50 rounded-lg p-4 space-y-2">
            <div className="flex items-center gap-2 text-sm">
              <Database className="w-4 h-4 text-slate-600 dark:text-slate-400" />
              <span className="text-slate-600 dark:text-slate-400">Repository:</span>
              <span className="font-semibold text-slate-900 dark:text-slate-100">{repositoryName}</span>
            </div>
            <div className="text-sm">
              <span className="text-slate-600 dark:text-slate-400">Path:</span>
              <p className="font-mono text-xs text-slate-700 dark:text-slate-300 mt-1 break-all">
                {repositoryPath}
              </p>
            </div>
          </div>

          {hasActiveWorktrees && (
            <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-4 space-y-2">
              <div className="flex items-start gap-2">
                <AlertCircle className="w-5 h-5 text-amber-600 dark:text-amber-400 mt-0.5" />
                <div className="flex-1">
                  <p className="font-semibold text-amber-800 dark:text-amber-300 text-sm">
                    Warning: This repository has {activeWorktrees.length} active worktree{activeWorktrees.length > 1 ? 's' : ''}
                  </p>
                  <ul className="mt-2 space-y-1">
                    {activeWorktrees.map((wt) => (
                      <li key={wt.id} className="flex items-center gap-2 text-sm text-amber-700 dark:text-amber-400">
                        <GitBranch className="w-3 h-3" />
                        <span className="font-medium">{wt.name}</span>
                        <span className="text-xs font-mono">({wt.branch})</span>
                      </li>
                    ))}
                  </ul>
                  <p className="mt-2 text-xs text-amber-600 dark:text-amber-500">
                    These worktrees must be removed before deleting the repository
                  </p>
                </div>
              </div>
            </div>
          )}

          <div className="space-y-3">
            <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
              <h4 className="font-semibold text-blue-800 dark:text-blue-300 text-sm mb-2">
                Option 1: Remove from Vibeman only
              </h4>
              <p className="text-sm text-blue-700 dark:text-blue-400 mb-3">
                Stop tracking this repository in Vibeman but keep all files on disk
              </p>
              <Button
                onClick={handleUntrack}
                disabled={isRemoving || hasActiveWorktrees}
                variant="outline"
                className="w-full border-blue-300 dark:border-blue-700 hover:bg-blue-100 dark:hover:bg-blue-900/30"
              >
                Remove from Vibeman
              </Button>
            </div>

            {!hasActiveWorktrees && (
              <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                <h4 className="font-semibold text-red-800 dark:text-red-300 text-sm mb-2">
                  Option 2: Delete from disk
                </h4>
                <p className="text-sm text-red-700 dark:text-red-400 mb-1">
                  Permanently delete the repository and all its files
                </p>
                <p className="text-xs text-red-600 dark:text-red-500 mb-3">
                  ⚠️ This action cannot be undone!
                </p>
                <Button
                  onClick={handleDelete}
                  disabled={isRemoving}
                  variant="destructive"
                  className="w-full"
                >
                  Delete Repository
                </Button>
              </div>
            )}
          </div>
        </div>
        
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isRemoving}
          >
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}