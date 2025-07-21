import React, { createContext, useContext, useState, useEffect, useRef, ReactNode } from 'react';
import type { DbRepository } from '@/generated/api/types.gen';

interface AppState {
  // Selected project for filtering worktrees
  selectedProject: DbRepository | null;
  
  // Refs for cleanup
  mountedRef: React.MutableRefObject<boolean>;
  
  // Actions
  setSelectedProject: (project: DbRepository | null) => void;
}

const AppStateContext = createContext<AppState | undefined>(undefined);

export function AppStateProvider({ children }: { children: ReactNode }) {
  const [selectedProject, setSelectedProject] = useState<DbRepository | null>(null);
  const mountedRef = useRef(true);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const value: AppState = {
    selectedProject,
    mountedRef,
    setSelectedProject: (project) => {
      if (mountedRef.current) {
        setSelectedProject(project);
      }
    }
  };

  return (
    <AppStateContext.Provider value={value}>
      {children}
    </AppStateContext.Provider>
  );
}

export function useAppState() {
  const context = useContext(AppStateContext);
  if (!context) {
    throw new Error('useAppState must be used within AppStateProvider');
  }
  return context;
}