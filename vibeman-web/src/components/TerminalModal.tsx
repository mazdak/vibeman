import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Terminal } from './Terminal';
import type { TerminalProps } from '../types/terminal';
import { X, Maximize2, Minimize2 } from 'lucide-react';
import { Button } from './ui/button';

interface TerminalModalProps extends TerminalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  worktreeName?: string;
}

export const TerminalModal: React.FC<TerminalModalProps> = ({
  open,
  onOpenChange,
  environmentId,
  worktreeName,
  className
}) => {
  const [isFullscreen, setIsFullscreen] = useState(false);

  const handleClose = () => {
    onOpenChange(false);
    setIsFullscreen(false);
  };

  const toggleFullscreen = () => {
    setIsFullscreen(!isFullscreen);
  };

  if (!open) return null;

  return (
    <AnimatePresence>
      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          {/* Backdrop */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
            onClick={handleClose}
          />

          {/* Modal Content */}
          <motion.div
            initial={{ 
              opacity: 0,
              scale: 0.95,
              y: 20
            }}
            animate={{ 
              opacity: 1,
              scale: 1,
              y: 0
            }}
            exit={{ 
              opacity: 0,
              scale: 0.95,
              y: 20
            }}
            transition={{
              type: "spring",
              stiffness: 300,
              damping: 30
            }}
            className={`
              relative bg-white dark:bg-slate-900 rounded-xl shadow-2xl border border-slate-200 dark:border-slate-700 overflow-hidden
              ${isFullscreen 
                ? 'w-screen h-screen rounded-none' 
                : 'w-full max-w-5xl h-[600px] max-h-[80vh] mx-4'
              }
            `}
          >
            {/* Modal Header */}
            <div className="flex items-center justify-between px-6 py-4 bg-slate-50 dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
              <div className="flex items-center gap-3">
                <div className="flex items-center gap-2">
                  <div className="w-3 h-3 bg-red-500 rounded-full"></div>
                  <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
                  <div className="w-3 h-3 bg-green-500 rounded-full"></div>
                </div>
                <div className="h-4 w-px bg-slate-300 dark:bg-slate-600"></div>
                <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
                  Terminal {worktreeName ? `- ${worktreeName}` : ''}
                </h2>
              </div>

              <div className="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={toggleFullscreen}
                  className="h-8 w-8 p-0 text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200"
                >
                  {isFullscreen ? (
                    <Minimize2 className="w-4 h-4" />
                  ) : (
                    <Maximize2 className="w-4 h-4" />
                  )}
                </Button>
                
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleClose}
                  className="h-8 w-8 p-0 text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200"
                >
                  <X className="w-4 h-4" />
                </Button>
              </div>
            </div>

            {/* Terminal Container */}
            <div className="flex-1 relative">
              <Terminal
                environmentId={environmentId}
                onClose={handleClose}
                className="absolute inset-0 border-0 rounded-none"
                isFullscreen={isFullscreen}
                onToggleFullscreen={toggleFullscreen}
              />
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
};

export default TerminalModal;