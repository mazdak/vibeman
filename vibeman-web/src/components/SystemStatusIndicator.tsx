import React from 'react';
import { Activity, AlertCircle } from 'lucide-react';
import { useSystemStatus } from '@/hooks/api/useSystemStatus';

export function SystemStatusIndicator() {
  const { data: status, isLoading, error } = useSystemStatus();

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700">
        <Activity className="w-4 h-4 text-slate-400 animate-pulse" />
        <span className="text-xs text-slate-500 dark:text-slate-400">Connecting...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
        <AlertCircle className="w-4 h-4 text-red-500" />
        <span className="text-xs text-red-700 dark:text-red-400">API Error</span>
      </div>
    );
  }

  const isHealthy = status?.status === 'healthy';
  
  return (
    <div className={`flex items-center gap-2 px-3 py-1.5 rounded-full border ${
      isHealthy 
        ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800' 
        : 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800'
    }`}>
      <Activity className={`w-4 h-4 ${
        isHealthy ? 'text-green-500' : 'text-yellow-500'
      }`} />
      <span className={`text-xs ${
        isHealthy 
          ? 'text-green-700 dark:text-green-400' 
          : 'text-yellow-700 dark:text-yellow-400'
      }`}>
        {status?.status || 'Unknown'}
      </span>
      {status && (
        <span className="text-xs text-slate-500 dark:text-slate-400">
          • {status.worktrees || 0} worktrees • {status.containers || 0} containers
        </span>
      )}
    </div>
  );
}