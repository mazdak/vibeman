import React, { useState, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  GitBranch,
  Play,
  Pause,
  RotateCw,
  Trash2,
  Terminal,
  FileText,
  Plus,
  AlertCircle,
  CheckCircle2,
  Clock,
  Loader2,
  Database,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { ServiceStatus } from '@/components/ServiceStatus';
import { TerminalModal } from '@/components/modals/TerminalModal';
import { LogModal } from '@/components/modals/LogModal';
import { useAppState } from '@/shared/context/AppStateContext';
import { getStatusIcon, getStatusColor } from '@/shared/utils/statusUtils';
import { useWorktrees } from '@/hooks/api/useWorktrees';
import { useServices } from '@/hooks/api/useServices';
// Removed old API wrapper import - using React Query hooks instead
import type { Service } from '@/types/legacy';

export function WorktreesTab() {
  const { selectedProject, mountedRef } = useAppState();
  
  // Use the new hook for worktrees
  const {
    worktrees,
    isLoading: worktreesLoading,
    error: worktreesError,
    refetch: refetchWorktrees,
    startWorktree,
    stopWorktree,
    deleteWorktree,
    isStarting,
    isStopping,
    isDeleting,
  } = useWorktrees({ repositoryId: selectedProject?.id });
  
  // Use the new hook for services
  const {
    services,
    isLoading: servicesLoading,
    error: servicesError,
    refetch: refetchServices,
    startService,
    stopService,
    restartService,
    isStarting: isStartingService,
    isStopping: isStoppingService,
    isRestarting: isRestartingService,
  } = useServices();
  
  // Local state
  const [selectedWorktree, setSelectedWorktree] = useState(null);
  const [isCreatingWorktree, setIsCreatingWorktree] = useState(false);
  const [terminalWorktreeId, setTerminalWorktreeId] = useState<string | null>(null);
  const [terminalWorktreeName, setTerminalWorktreeName] = useState<string | null>(null);
  const [logWorktreeId, setLogWorktreeId] = useState<string | null>(null);
  const [logWorktreeName, setLogWorktreeName] = useState<string | null>(null);
  
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  const handleStartService = (id: string) => {
    startService({ path: { id } });
  };

  const handleStopService = (id: string) => {
    stopService({ path: { id } });
  };

  const handleRestartService = (id: string) => {
    restartService({ path: { id } });
  };

  const handleWorktreeAction = async (action: string, id: string) => {
    try {
      // Find the worktree to update
      const worktree = worktrees.find(w => w.id === id);
      if (!worktree) return;

      switch (action) {
        case "start":
          startWorktree({ path: { id } });
          break;
        case "stop":
          stopWorktree({ path: { id } });
          break;
        case "restart":
          // For restart, we stop then start
          stopWorktree({ path: { id } }, {
            onSuccess: () => {
              setTimeout(() => {
                startWorktree({ path: { id } });
              }, 1000);
            }
          });
          break;
        case "delete":
          if (!confirm(`Are you sure you want to delete the worktree "${worktree.name}"?`)) {
            return;
          }
          deleteWorktree(id);
          break;
      }
    } catch (err) {
      console.error(`Failed to ${action} worktree:`, err);
    }
  };

  const StatusIcon = ({ status }: { status: string }) => {
    const Icon = getStatusIcon(status);
    const colorClass = getStatusColor(status);
    const isTransitional = ['creating', 'starting', 'stopping'].includes(status);
    
    return (
      <Icon 
        className={`w-4 h-4 ${colorClass} ${isTransitional ? 'animate-spin' : ''}`} 
      />
    );
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -10 }}
      transition={{ duration: 0.2 }}
      className="space-y-6"
    >
      {/* Worktrees Section */}
      <div>
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-2xl font-semibold text-slate-800 dark:text-slate-200">Active Worktrees</h2>
            <p className="text-slate-600 dark:text-slate-400 mt-1">
              Manage your development environments and containers
            </p>
          </div>
          
          <div className="flex gap-3">
            <Button
              variant="outline"
              onClick={() => refetchWorktrees()}
              className="border-slate-300 dark:border-slate-600"
              disabled={worktreesLoading}
            >
              <RotateCw className={`w-4 h-4 mr-2 ${worktreesLoading ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
            <Button
              onClick={() => setIsCreatingWorktree(true)}
              className="bg-gradient-to-r from-cyan-500 to-purple-500 text-white hover:from-cyan-600 hover:to-purple-600"
            >
              <Plus className="w-4 h-4 mr-2" />
              New Worktree
            </Button>
          </div>
        </div>

        {worktreesLoading ? (
          <div className="flex items-center justify-center p-16">
            <Loader2 className="w-8 h-8 animate-spin text-cyan-500" />
          </div>
        ) : worktreesError ? (
          <Card className="p-8 text-center border-red-200 dark:border-red-900/50 bg-red-50 dark:bg-red-900/20">
            <AlertCircle className="w-12 h-12 mx-auto mb-4 text-red-500" />
            <h3 className="text-xl font-semibold mb-2 text-red-700 dark:text-red-300">Error Loading Worktrees</h3>
            <p className="text-red-600 dark:text-red-400 mb-4">
              {worktreesError.message || 'Failed to load worktrees'}
            </p>
            <Button
              onClick={() => refetchWorktrees()}
              variant="outline"
              className="border-red-300 text-red-700 hover:bg-red-100 dark:border-red-700 dark:text-red-300 dark:hover:bg-red-900/30"
            >
              <RotateCw className="w-4 h-4 mr-2" />
              Try Again
            </Button>
          </Card>
        ) : worktrees.length === 0 ? (
          <Card className="p-16 text-center border-2 border-dashed border-slate-200 dark:border-slate-700/50 bg-white/50 dark:bg-slate-800/30 backdrop-blur-sm shadow-none">
            <div className="max-w-md mx-auto">
              <GitBranch className="w-16 h-16 mx-auto mb-4 text-slate-400 dark:text-slate-600" />
              <h3 className="text-xl font-semibold mb-2 text-slate-700 dark:text-slate-300">No Active Worktrees</h3>
              <p className="text-slate-600 dark:text-slate-400 mb-6">
                Create your first worktree to start developing
              </p>
              <Button
                onClick={() => setIsCreatingWorktree(true)}
                className="bg-gradient-to-r from-cyan-500 to-purple-500 text-white hover:from-cyan-600 hover:to-purple-600"
              >
                <Plus className="w-4 h-4 mr-2" />
                Create Worktree
              </Button>
            </div>
          </Card>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
            {worktrees.map((worktree) => (
              <Card
                key={worktree.id}
                className="overflow-hidden border-slate-200 dark:border-slate-700/50 bg-white/70 dark:bg-slate-800/50 backdrop-blur-sm hover:shadow-lg transition-all duration-200"
              >
                <div className="p-6">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-start gap-3">
                      <div className="p-2 rounded-lg bg-gradient-to-br from-cyan-500/10 to-purple-500/10 dark:from-cyan-500/20 dark:to-purple-500/20">
                        <GitBranch className="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
                      </div>
                      <div>
                        <h3 className="font-semibold text-slate-800 dark:text-slate-200">{worktree.name}</h3>
                        <p className="text-sm text-slate-600 dark:text-slate-400">Project {worktree.repository_id}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <StatusIcon status={worktree.status} />
                      <span className={`text-sm font-medium ${getStatusColor(worktree.status)}`}>
                        {worktree.status}
                      </span>
                    </div>
                  </div>

                  <div className="space-y-2 mb-4">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-slate-600 dark:text-slate-400">Branch</span>
                      <span className="font-mono text-slate-700 dark:text-slate-300">{worktree.branch || 'main'}</span>
                    </div>
                    {worktree.container_id && (
                      <div className="flex items-center justify-between text-sm">
                        <span className="text-slate-600 dark:text-slate-400">Container</span>
                        <span className="text-slate-700 dark:text-slate-300">{worktree.container_id}</span>
                      </div>
                    )}
                    {(worktree.updated_at || worktree.created_at) && (
                      <div className="flex items-center justify-between text-sm">
                        <span className="text-slate-600 dark:text-slate-400">Last Activity</span>
                        <div className="flex items-center gap-1 text-slate-700 dark:text-slate-300">
                          <Clock className="w-3 h-3" />
                          {new Date(worktree.updated_at || worktree.created_at || '').toLocaleTimeString()}
                        </div>
                      </div>
                    )}
                  </div>

                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleWorktreeAction(worktree.status === 'running' ? 'stop' : 'start', worktree.id)}
                      disabled={['starting', 'stopping'].includes(worktree.status || '') || isStarting || isStopping}
                      className="flex-1"
                    >
                      {['starting', 'stopping'].includes(worktree.status || '') || isStarting || isStopping ? (
                        <>
                          <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                          {worktree.status === 'stopping' || isStopping ? 'Stopping...' : 'Starting...'}
                        </>
                      ) : worktree.status === 'running' ? (
                        <>
                          <Pause className="w-3 h-3 mr-1" />
                          Stop
                        </>
                      ) : (
                        <>
                          <Play className="w-3 h-3 mr-1" />
                          Start
                        </>
                      )}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        setTerminalWorktreeId(worktree.id);
                        setTerminalWorktreeName(worktree.name);
                      }}
                      disabled={worktree.status !== 'running'}
                      title="Open AI Terminal"
                    >
                      <Terminal className="w-3 h-3 mr-1" />
                      <span className="text-xs hidden sm:inline">AI</span>
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        setLogWorktreeId(worktree.id);
                        setLogWorktreeName(worktree.name);
                      }}
                    >
                      <FileText className="w-3 h-3" />
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleWorktreeAction('delete', worktree.id)}
                      className="text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
                    >
                      <Trash2 className="w-3 h-3" />
                    </Button>
                  </div>
                </div>
              </Card>
            ))}
          </div>
        )}
      </div>

      {/* Services Section */}
      <div className="mt-12 pt-12 border-t border-slate-200 dark:border-slate-700/50">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-2xl font-semibold text-slate-800 dark:text-slate-200">Shared Services</h2>
            <p className="text-slate-600 dark:text-slate-400 mt-1">
              Manage databases, caches, and other services shared across worktrees
            </p>
          </div>
          
          <Button
            variant="outline"
            onClick={() => refetchServices()}
            className="border-slate-300 dark:border-slate-600"
            disabled={servicesLoading}
          >
            <RotateCw className={`w-4 h-4 mr-2 ${servicesLoading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
        </div>

        {servicesLoading ? (
          <Card className="p-16 border-2 border-dashed border-slate-200 dark:border-slate-700/50 bg-white/50 dark:bg-slate-800/30 backdrop-blur-sm shadow-none">
            <div className="flex items-center justify-center">
              <Loader2 className="w-8 h-8 animate-spin text-cyan-500" />
            </div>
          </Card>
        ) : services.length === 0 ? (
          <Card className="p-8 text-center border-2 border-dashed border-slate-200 dark:border-slate-700/50 bg-white/50 dark:bg-slate-800/30 backdrop-blur-sm shadow-none">
            <div className="max-w-md mx-auto">
              <div className="w-12 h-12 mx-auto mb-4 rounded-lg bg-slate-100 dark:bg-slate-800 flex items-center justify-center">
                <Database className="w-6 h-6 text-slate-400 dark:text-slate-600" />
              </div>
              <h3 className="text-lg font-semibold mb-2 text-slate-700 dark:text-slate-300">No Services Running</h3>
              <p className="text-slate-600 dark:text-slate-400 text-sm">
                Shared services like databases and caches will appear here when started
              </p>
            </div>
          </Card>
        ) : (
          <ServiceStatus
            services={services}
            onStartService={(name) => {
              const service = services.find(s => s.name === name);
              if (service?.id) handleStartService(service.id);
            }}
            onStopService={(name) => {
              const service = services.find(s => s.name === name);
              if (service?.id) handleStopService(service.id);
            }}
            onRestartService={(name) => {
              const service = services.find(s => s.name === name);
              if (service?.id) handleRestartService(service.id);
            }}
            loading={servicesLoading}
            error={servicesError?.message || null}
          />
        )}
      </div>

      {/* Modals */}
      <TerminalModal
        isOpen={!!terminalWorktreeId}
        onClose={() => {
          setTerminalWorktreeId(null);
          setTerminalWorktreeName(null);
        }}
        worktreeId={terminalWorktreeId || ''}
        title={`Terminal - ${terminalWorktreeName || 'Worktree'}`}
      />

      <LogModal
        isOpen={!!logWorktreeId}
        onClose={() => {
          setLogWorktreeId(null);
          setLogWorktreeName(null);
        }}
        worktreeId={logWorktreeId || ''}
        title={`Logs - ${logWorktreeName || 'Worktree'}`}
      />
    </motion.div>
  );
}