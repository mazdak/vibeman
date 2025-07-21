"use client";

import React, { useState, useEffect, useRef } from "react";
import {
  GitBranch,
  Container,
  Play,
  Square,
  Terminal,
  Settings,
  Plus,
  Trash2,
  Eye,
  Sun,
  Moon,
  Monitor,
  Activity,
  Folder,
  Code,
  Database,
  Clock,
  CheckCircle,
  AlertCircle,
  XCircle,
  RefreshCw,
  ExternalLink,
  Info,
  FileText,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import logo from "../logo.png";
import { useWorktrees } from "../hooks/api/useWorktrees";
import { useRepositories } from "../hooks/api/useRepositories";
import { useSystemStatus } from "../hooks/api/useSystemStatus";
import { useServices } from "../hooks/api/useServices";
import { useConfig } from "../hooks/api/useConfig";
import type { UIWorktree } from "../lib/api-client";
import type { DbRepository, ServerConfigResponse } from "../lib/api-client";

// Service types from generated API
import type { ServerService } from "../generated/api/types.gen";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./ui/select";
import { CreateRepositoryModal } from "./CreateRepositoryModal";
import { CreateWorktreeModal } from "./CreateWorktreeModal";
import { RemoveWorktreeDialog } from "./RemoveWorktreeDialog";
import { RemoveRepositoryDialog } from "./RemoveRepositoryDialog";
import { TerminalModal } from "./TerminalModal";
import { LogModal } from "./logs/LogModal";
import { ServiceStatus } from "./ServiceStatus";
import { SystemStatusIndicator } from "./SystemStatusIndicator";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Checkbox } from "./ui/checkbox";
import { ToastContainer, useToast } from "./ui/toast";
// Extend Worktree type with repository info
type WorktreeWithRepository = UIWorktree & {
  repository?: string; // Will be populated from repository data
};
interface VibemanManagementUIProps {
  onLogout?: () => void;
}
const VibemanManagementUI: React.FC<VibemanManagementUIProps> = ({
  onLogout,
}) => {
  // React Query hooks
  const { 
    repositories, 
    isLoading: repositoriesLoading, 
    error: repositoriesError,
    createRepository,
    isCreating: isCreatingRepository,
    deleteRepository,
    isDeleting: isDeletingRepository
  } = useRepositories();
  
  const { 
    worktrees, 
    isLoading: worktreesLoading, 
    error: worktreesError,
    createWorktree,
    isCreating: isCreateWorktreeModalOpen,
    startWorktree,
    isStarting: isStartingWorktree,
    stopWorktree,
    isStopping: isStoppingWorktree,
    deleteWorktree,
    isDeleting: isDeletingWorktree
  } = useWorktrees();

  const {
    services,
    isLoading: servicesLoading,
    error: servicesError,
    startService,
    stopService,
    restartService,
    isStarting: isStartingService,
    isStopping: isStoppingService,
    isRestarting: isRestartingService,
  } = useServices();

  const {
    config,
    isLoading: configLoading,
    error: configError,
    updateConfig,
    isUpdating: configSaving,
  } = useConfig();

  // UI state
  const [isCreateRepoModalOpen, setIsCreateRepoModalOpen] = useState(false);
  const [isCreateWorktreeModalOpen, setIsCreateWorktreeModalOpen] = useState(false);
  const [themeMode, setThemeMode] = useState<'light' | 'dark' | 'system'>('system');
  const [systemPrefersDark, setSystemPrefersDark] = useState(false);
  const [activeTab, setActiveTab] = useState<
    "worktrees" | "repositories" | "settings"
  >("worktrees");
  const [selectedWorktree, setSelectedWorktree] = useState<string | null>(null);
  const [terminalWorktreeId, setTerminalWorktreeId] = useState<string | null>(null);
  const [terminalWorktreeName, setTerminalWorktreeName] = useState<string | null>(null);
  const [logWorktreeId, setLogWorktreeId] = useState<string | null>(null);
  const [logWorktreeName, setLogWorktreeName] = useState<string | null>(null);
  const [removeWorktreeId, setRemoveWorktreeId] = useState<string | null>(null);
  const [removeWorktreeData, setRemoveWorktreeData] = useState<{name: string; branch: string} | null>(null);
  const [removeRepositoryId, setRemoveRepositoryId] = useState<string | null>(null);
  const [removeRepositoryData, setRemoveRepositoryData] = useState<{name: string; path: string} | null>(null);
  
  const { toasts, toast, dismiss } = useToast();

  // Transform worktrees with repository info
  const worktreesWithRepository: WorktreeWithRepository[] = worktrees.map(wt => {
    const repository = repositories.find(r => r.id === wt.repository_id);
    return {
      ...wt,
      repository: repository?.name || 'Unknown'
    };
  });

  // Compute loading states
  const loading = repositoriesLoading || worktreesLoading;
  const error = repositoriesError || worktreesError;

  // Track mounted state for async operations
  const mountedRef = useRef(true);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  // Data is automatically loaded by React Query hooks

  // Services and config are automatically loaded by React Query hooks

  // Data fetching is handled by React Query hooks

  const handleStartService = async (name: string) => {
    try {
      startService(name);
    } catch (err) {
      console.error(`Failed to start service ${name}:`, err);
      toast({
        title: "Error",
        description: `Failed to start service ${name}`,
        type: "error",
      });
    }
  };

  const handleStopService = async (name: string) => {
    try {
      stopService(name);
    } catch (err) {
      console.error(`Failed to stop service ${name}:`, err);
      toast({
        title: "Error",
        description: `Failed to stop service ${name}`,
        type: "error",
      });
    }
  };

  const handleRestartService = async (name: string) => {
    try {
      restartService(name);
    } catch (err) {
      console.error(`Failed to restart service ${name}:`, err);
      toast({
        title: "Error",
        description: `Failed to restart service ${name}`,
        type: "error",
      });
    }
  };

  const saveConfig = () => {
    if (!config) return;
    
    updateConfig(config);
    toast({
      title: 'Settings saved',
      description: 'Your configuration has been updated successfully.',
      type: 'success'
    });
  };

  const resetConfig = () => {
    const defaultConfig: ServerConfigResponse = {
      storage: {
        repositories_path: "~/vibeman/repos",
        worktrees_path: "~/vibeman/worktrees",
      },
      git: {
        default_branch_prefix: "feature/",
        auto_fetch: true,
      },
      container: {
        default_runtime: "docker",
        auto_start: true,
      },
    };
    updateConfig(defaultConfig);
  };

  // Detect system theme preference
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    setSystemPrefersDark(mediaQuery.matches);
    
    const handleChange = (e: MediaQueryListEvent) => {
      setSystemPrefersDark(e.matches);
    };
    
    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  // Apply theme based on mode
  useEffect(() => {
    const isDark = themeMode === 'dark' || (themeMode === 'system' && systemPrefersDark);
    document.documentElement.classList.toggle('dark', isDark);
  }, [themeMode, systemPrefersDark]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      mountedRef.current = false;
      // Clear any pending timeouts
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      }
    };
  }, []);

  const cycleTheme = () => {
    const modes: Array<'light' | 'dark' | 'system'> = ['light', 'dark', 'system'];
    const currentIndex = modes.indexOf(themeMode);
    const nextIndex = (currentIndex + 1) % modes.length;
    setThemeMode(modes[nextIndex]);
  };

  const getThemeIcon = () => {
    if (themeMode === 'system') return <Monitor className="w-5 h-5 text-slate-600" />;
    if (themeMode === 'dark') return <Moon className="w-5 h-5 text-slate-600" />;
    return <Sun className="w-5 h-5 text-amber-500" />;
  };
  const getStatusIcon = (status: string) => {
    switch (status) {
      case "running":
      case "active":
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case "building":
        return <RefreshCw className="w-4 h-4 text-blue-500 animate-spin" />;
      case "error":
      case "failed":
        return <XCircle className="w-4 h-4 text-red-500" />;
      default:
        return <AlertCircle className="w-4 h-4 text-gray-500" />;
    }
  };
  const getStatusColor = (status: string) => {
    switch (status) {
      case "running":
      case "active":
        return "bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400";
      case "building":
        return "bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400";
      case "error":
      case "failed":
        return "bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400";
      default:
        return "bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400";
    }
  };
  const handleWorktreeAction = async (
    id: string,
    action: "start" | "stop" | "delete",
  ) => {
    try {

      // Make API call
      switch (action) {
        case "start":
          startWorktree(id);
          toast({
            title: 'Worktree started',
            description: 'Container is now running.',
            type: 'success'
          });
          break;
        case "stop":
          stopWorktree(id);
          toast({
            title: 'Worktree stopped',
            description: 'Container has been stopped.',
            type: 'success'
          });
          break;
        case "delete":
          deleteWorktree(id);
          toast({
            title: 'Worktree removed',
            description: 'The worktree has been deleted.',
            type: 'success'
          });
          break;
      }
      
      // Refresh data to get latest state
      if (action !== "delete") {
        // Clear any existing timeout
        if (timeoutRef.current) {
          clearTimeout(timeoutRef.current);
        }
        
        timeoutRef.current = setTimeout(() => {
          if (mountedRef.current) {
            // Data will be refetched automatically by React Query
          }
          timeoutRef.current = null;
        }, 1000);
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Action failed';
      toast({
        title: 'Error',
        description: errorMessage,
        type: 'error'
      });
      
      // Show specific error toast for delete action
      if (action === "delete") {
        toast({
          title: 'Failed to remove worktree',
          description: errorMessage,
          type: 'error'
        });
      }
      
      // Reload to revert optimistic update
      // Data is automatically refreshed by React Query
    }
  };
  return (
    <div
      className={`min-h-screen w-full transition-colors duration-150 ${themeMode === 'dark' || (themeMode === 'system' && systemPrefersDark) ? "bg-slate-900" : "bg-gradient-to-br from-slate-50 to-slate-100"}`}
      style={{ minHeight: '100vh' }}
    >
      {/* Header */}
      <header className="bg-white/90 dark:bg-slate-800/90 backdrop-blur-xl border-b border-slate-200/50 dark:border-slate-700/50 sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 py-4">
          <div className="flex items-center justify-between">
            {/* Logo and Title */}
            <div className="flex items-center gap-2">
              <div className="w-12 h-12 sm:w-16 sm:h-16">
                <img
                  src={logo}
                  alt="Vibeman Logo"
                  className="w-full h-full object-contain"
                />
              </div>
              <div className="flex flex-col justify-center mt-3">
                <h1 className="text-xl sm:text-2xl font-bold bg-gradient-to-r from-cyan-500 to-purple-500 bg-clip-text text-transparent tracking-wider leading-tight">
                  VIBEMAN
                </h1>
                <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-400 font-medium tracking-wide -mt-1">
                  Manage your Vibe
                </p>
              </div>
            </div>

            {/* Header Actions */}
            <div className="flex items-center gap-3">
              <SystemStatusIndicator />
              <motion.button
                onClick={cycleTheme}
                className="p-2 rounded-full bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm border border-slate-200 dark:border-slate-700 shadow-lg hover:shadow-xl transition-all duration-150"
                whileHover={{
                  scale: 1.05,
                }}
                whileTap={{
                  scale: 0.95,
                }}
                aria-label="Toggle theme"
              >
                {getThemeIcon()}
              </motion.button>
            </div>
          </div>

          {/* Navigation Tabs */}
          <nav className="flex gap-1 mt-6 overflow-x-auto scrollbar-hide">
            {[
              {
                id: "worktrees",
                label: "Worktrees",
                icon: GitBranch,
              },
              {
                id: "repositories",
                label: "Repositories",
                icon: Database,
              },
              {
                id: "settings",
                label: "Settings",
                icon: Settings,
              },
            ].map(({ id, label, icon: Icon }) => (
              <button
                key={id}
                onClick={() => setActiveTab(id as any)}
                className={`flex items-center gap-2 px-3 sm:px-4 py-2 rounded-lg font-medium transition-all duration-100 whitespace-nowrap ${activeTab === id ? "bg-gradient-to-r from-cyan-500 to-purple-500 text-white shadow-lg" : "text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100 hover:bg-slate-100 dark:hover:bg-slate-700/50"}`}
              >
                <Icon className="w-4 h-4" />
                <span className="hidden sm:inline">{label}</span>
              </button>
            ))}
          </nav>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 py-6 sm:py-8">
        {/* Error Alert */}
        {error && (
          <div className="mb-6 p-4 bg-red-100 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
            <div className="flex items-center gap-2">
              <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400" />
              <p className="text-red-700 dark:text-red-300">{error}</p>
              <button
                onClick={() => {}}
                className="ml-auto text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-200"
              >
                <XCircle className="w-4 h-4" />
              </button>
            </div>
          </div>
        )}

        {/* Loading State */}
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <RefreshCw className="w-8 h-8 text-cyan-500 animate-spin" />
          </div>
        ) : (
          <AnimatePresence mode="wait">
          {activeTab === "worktrees" && (
            <motion.div
              key="worktrees"
              initial={{
                opacity: 0,
                y: 20,
              }}
              animate={{
                opacity: 1,
                y: 0,
              }}
              exit={{
                opacity: 0,
                y: -20,
              }}
              transition={{
                duration: 0.15,
              }}
              className="space-y-6"
            >
              {/* Worktrees Header */}
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-2xl font-bold text-slate-900 dark:text-slate-100">
                    Active Worktrees
                  </h2>
                  <p className="text-slate-600 dark:text-slate-400 mt-1">
                    Manage your isolated development worktrees
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <motion.button
                    onClick={() => {
                      // Services are automatically refreshed by React Query
                    }}
                    className="flex items-center gap-2 px-3 py-2 bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-lg font-medium transition-colors"
                    whileHover={{
                      scale: 1.02,
                    }}
                    whileTap={{
                      scale: 0.98,
                    }}
                  >
                    <RefreshCw className="w-4 h-4" />
                    Refresh
                  </motion.button>
                  <motion.button
                    onClick={() => setIsCreateWorktreeModalOpen(true)}
                    className="flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-cyan-500 to-purple-500 text-white rounded-lg font-medium shadow-lg hover:shadow-xl transition-all duration-100"
                    whileHover={{
                      scale: 1.02,
                    }}
                    whileTap={{
                      scale: 0.98,
                    }}
                  >
                    <Plus className="w-4 h-4" />
                    New Worktree
                  </motion.button>
                </div>
              </div>

              {/* Worktrees Grid */}
              {worktreesWithRepository.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-16 px-6 bg-white/50 dark:bg-slate-800/50 rounded-2xl border-2 border-dashed border-slate-300 dark:border-slate-600">
                  <GitBranch className="w-12 h-12 text-slate-400 dark:text-slate-500 mb-4" />
                  <h3 className="text-lg font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    No worktrees yet
                  </h3>
                  <p className="text-sm text-slate-600 dark:text-slate-400 text-center mb-6 max-w-md">
                    Create your first worktree to start developing in an isolated workspace with its own container.
                  </p>
                  <motion.button
                    onClick={() => setIsCreateWorktreeModalOpen(true)}
                    className="flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-cyan-500 to-purple-500 text-white rounded-lg font-medium shadow-lg hover:shadow-xl transition-all duration-100"
                    whileHover={{
                      scale: 1.02,
                    }}
                    whileTap={{
                      scale: 0.98,
                    }}
                  >
                    <Plus className="w-4 h-4" />
                    Create First Worktree
                  </motion.button>
                </div>
              ) : (
                <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
                {worktreesWithRepository.map((worktree) => (
                  <motion.div
                    key={worktree.id}
                    layout
                    className="bg-white/90 dark:bg-slate-800/90 backdrop-blur-xl rounded-2xl border border-slate-200/50 dark:border-slate-700/50 p-6 shadow-lg hover:shadow-xl transition-all duration-150"
                  >
                    {/* Worktree Header */}
                    <div className="flex items-start justify-between mb-4">
                      <div className="flex-1">
                        <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-1">
                          {worktree.name}
                        </h3>
                        <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
                          <GitBranch className="w-3 h-3" />
                          {worktree.branch}
                        </div>
                      </div>
                      <div className="flex items-center gap-1">
                        {getStatusIcon(worktree.status)}
                        <span
                          className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(worktree.status)}`}
                        >
                          {worktree.status}
                        </span>
                      </div>
                    </div>

                    {/* Repository Info */}
                    <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400 mb-4">
                      <Database className="w-3 h-3" />
                      {worktree.repository}
                    </div>

                    {/* Path Info */}
                    {worktree.path && (
                      <div className="flex items-start gap-2 text-sm text-slate-600 dark:text-slate-400 mb-4">
                        <Folder className="w-3 h-3 mt-0.5 flex-shrink-0" />
                        <span className="font-mono text-xs break-all" title={worktree.path}>
                          {worktree.path}
                        </span>
                      </div>
                    )}

                    {/* Container Status */}
                    <div className="flex items-center justify-between mb-4 p-3 bg-slate-50 dark:bg-slate-700/50 rounded-lg">
                      <div className="flex items-center gap-2">
                        <Container className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                        <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
                          Container
                        </span>
                      </div>
                      <div className="flex items-center gap-2">
                        {getStatusIcon(worktree.container_status)}
                        <span
                          className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(worktree.container_status)}`}
                        >
                          {worktree.container_status}
                        </span>
                      </div>
                    </div>

                    {/* Last Activity */}
                    <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400 mb-4">
                      <Clock className="w-3 h-3" />
                      Last activity: {worktree.last_activity}
                    </div>

                    {/* Port Info */}
                    {worktree.port && (
                      <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400 mb-4">
                        <ExternalLink className="w-3 h-3" />
                        Port: {worktree.port}
                      </div>
                    )}

                    {/* Actions */}
                    <div className="flex flex-wrap gap-2">
                      {worktree.status === "running" ? (
                        <button
                          onClick={() =>
                            handleWorktreeAction(worktree.id, "stop")
                          }
                          className="flex items-center gap-1 px-3 py-1.5 bg-amber-100 hover:bg-amber-200 dark:bg-amber-900/20 dark:hover:bg-amber-900/30 text-amber-700 dark:text-amber-400 rounded-lg text-sm font-medium transition-colors"
                        >
                          <Square className="w-3 h-3" />
                          <span className="hidden sm:inline">Stop</span>
                        </button>
                      ) : (
                        <button
                          onClick={() =>
                            handleWorktreeAction(worktree.id, "start")
                          }
                          className="flex items-center gap-1 px-3 py-1.5 bg-green-100 hover:bg-green-200 dark:bg-green-900/20 dark:hover:bg-green-900/30 text-green-700 dark:text-green-400 rounded-lg text-sm font-medium transition-colors"
                        >
                          <Play className="w-3 h-3" />
                          <span className="hidden sm:inline">Start</span>
                        </button>
                      )}

                      <button 
                        onClick={() => {
                          setTerminalWorktreeId(worktree.id);
                          setTerminalWorktreeName(worktree.name);
                        }}
                        disabled={worktree.status !== "running"}
                        className={`flex items-center gap-1 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                          worktree.status === "running" 
                            ? "bg-blue-100 hover:bg-blue-200 dark:bg-blue-900/20 dark:hover:bg-blue-900/30 text-blue-700 dark:text-blue-400" 
                            : "bg-gray-100 dark:bg-gray-900/20 text-gray-400 dark:text-gray-600 cursor-not-allowed"
                        }`}
                        title="Terminal"
                      >
                        <Terminal className="w-3 h-3" />
                        <span className="hidden sm:inline">Terminal</span>
                      </button>

                      <button 
                        onClick={() => {
                          setLogWorktreeId(worktree.id);
                          setLogWorktreeName(worktree.name);
                        }}
                        disabled={worktree.status !== "running"}
                        className={`flex items-center gap-1 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                          worktree.status === "running" 
                            ? "bg-purple-100 hover:bg-purple-200 dark:bg-purple-900/20 dark:hover:bg-purple-900/30 text-purple-700 dark:text-purple-400" 
                            : "bg-gray-100 dark:bg-gray-900/20 text-gray-400 dark:text-gray-600 cursor-not-allowed"
                        }`}
                        title="Logs"
                      >
                        <FileText className="w-3 h-3" />
                        <span className="hidden sm:inline">Logs</span>
                      </button>

                      <button
                        onClick={() => {
                          setRemoveWorktreeId(worktree.id);
                          setRemoveWorktreeData({
                            name: worktree.name,
                            branch: worktree.branch
                          });
                        }}
                        className="flex items-center gap-1 px-3 py-1.5 bg-red-100 hover:bg-red-200 dark:bg-red-900/20 dark:hover:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg text-sm font-medium transition-colors"
                        title="Delete"
                      >
                        <Trash2 className="w-3 h-3" />
                        <span className="hidden sm:inline">Delete</span>
                      </button>
                    </div>
                  </motion.div>
                ))}
                </div>
              )}

              {/* Services Section */}
              {services.length > 0 && (
                <div className="mt-8">
                  <div className="mb-4">
                    <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
                      Global Services
                    </h3>
                    <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                      Shared services used by your development worktrees
                    </p>
                  </div>
                  <ServiceStatus
                    services={services}
                    onStartService={handleStartService}
                    onStopService={handleStopService}
                    onRestartService={handleRestartService}
                    loading={servicesLoading}
                    error={servicesError instanceof Error ? servicesError.message : servicesError ? String(servicesError) : null}
                  />
                </div>
              )}
            </motion.div>
          )}

          {activeTab === "repositories" && (
            <motion.div
              key="repositories"
              initial={{
                opacity: 0,
                y: 20,
              }}
              animate={{
                opacity: 1,
                y: 0,
              }}
              exit={{
                opacity: 0,
                y: -20,
              }}
              transition={{
                duration: 0.15,
              }}
              className="space-y-6"
            >
              <div>
                <h2 className="text-2xl font-bold text-slate-900 dark:text-slate-100">
                  Configured Repositories
                </h2>
                <p className="text-slate-600 dark:text-slate-400 mt-1">
                  Manage your Git repositories and TOML configurations
                </p>
              </div>

              {repositories.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-16 px-6 bg-white/50 dark:bg-slate-800/50 rounded-2xl border-2 border-dashed border-slate-300 dark:border-slate-600">
                  <Database className="w-12 h-12 text-slate-400 dark:text-slate-500 mb-4" />
                  <h3 className="text-lg font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    No repositories configured
                  </h3>
                  <p className="text-sm text-slate-600 dark:text-slate-400 text-center mb-6 max-w-md">
                    Add your first Git repository to start creating worktrees and development environments.
                  </p>
                  <motion.button
                    className="flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-cyan-500 to-purple-500 text-white rounded-lg font-medium shadow-lg hover:shadow-xl transition-all duration-100"
                    whileHover={{
                      scale: 1.02,
                    }}
                    whileTap={{
                      scale: 0.98,
                    }}
                    onClick={() => setIsCreateRepoModalOpen(true)}
                  >
                    <Plus className="w-4 h-4" />
                    Add Repository
                  </motion.button>
                </div>
              ) : (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {repositories.map((repository) => {
                  const repositoryWorktreeCount = worktreesWithRepository.filter(w => w.repository_id === repository.id).length;
                  return (
                  <motion.div
                    key={repository.id}
                    className="bg-white/90 dark:bg-slate-800/90 backdrop-blur-xl rounded-2xl border border-slate-200/50 dark:border-slate-700/50 p-6 shadow-lg hover:shadow-xl transition-all duration-150"
                  >
                    <div className="flex items-start justify-between mb-4">
                      <div>
                        <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-2">
                          {repository.name}
                        </h3>
                        <p className="text-sm text-slate-600 dark:text-slate-400 font-mono">
                          {repository.repository_url}
                        </p>
                      </div>
                      <span className="px-3 py-1 bg-cyan-100 dark:bg-cyan-900/20 text-cyan-700 dark:text-cyan-400 rounded-full text-sm font-medium">
                        {repositoryWorktreeCount} worktrees
                      </span>
                    </div>

                    {/* Path Info */}
                    {repository.path && (
                      <div className="flex items-start gap-2 text-sm text-slate-600 dark:text-slate-400 mb-4">
                        <Folder className="w-3 h-3 mt-0.5 flex-shrink-0" />
                        <span className="font-mono text-xs break-all" title={repository.path}>
                          {repository.path}
                        </span>
                      </div>
                    )}

                    <div className="flex flex-wrap gap-2">
                      <button className="flex items-center gap-1 px-3 py-1.5 bg-blue-100 hover:bg-blue-200 dark:bg-blue-900/20 dark:hover:bg-blue-900/30 text-blue-700 dark:text-blue-400 rounded-lg text-sm font-medium transition-colors" title="View Config">
                        <Eye className="w-3 h-3" />
                        <span className="hidden sm:inline">View Config</span>
                      </button>
                      <button className="flex items-center gap-1 px-3 py-1.5 bg-purple-100 hover:bg-purple-200 dark:bg-purple-900/20 dark:hover:bg-purple-900/30 text-purple-700 dark:text-purple-400 rounded-lg text-sm font-medium transition-colors" title="Edit">
                        <Settings className="w-3 h-3" />
                        <span className="hidden sm:inline">Edit</span>
                      </button>
                      <button 
                        onClick={() => {
                          setRemoveRepositoryId(repository.id);
                          setRemoveRepositoryData({
                            name: repository.name,
                            path: repository.path || repository.repository_url
                          });
                        }}
                        className="flex items-center gap-1 px-3 py-1.5 bg-red-100 hover:bg-red-200 dark:bg-red-900/20 dark:hover:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg text-sm font-medium transition-colors"
                        title="Remove"
                      >
                        <Trash2 className="w-3 h-3" />
                        <span className="hidden sm:inline">Remove</span>
                      </button>
                    </div>
                  </motion.div>
                  );
                })}
                </div>
              )}
            </motion.div>
          )}

          {activeTab === "settings" && (
            <motion.div
              key="settings"
              initial={{
                opacity: 0,
                y: 20,
              }}
              animate={{
                opacity: 1,
                y: 0,
              }}
              exit={{
                opacity: 0,
                y: -20,
              }}
              transition={{
                duration: 0.15,
              }}
              className="space-y-6"
            >
              <div>
                <h2 className="text-2xl font-bold text-slate-900 dark:text-slate-100">
                  Settings
                </h2>
                <p className="text-slate-600 dark:text-slate-400 mt-1">
                  Configure your Vibeman environment
                </p>
              </div>

              <div className="bg-white/90 dark:bg-slate-800/90 backdrop-blur-xl rounded-2xl border border-slate-200/50 dark:border-slate-700/50 p-6 shadow-lg">
                <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-4">
                  General Settings
                </h3>
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <label className="font-medium text-slate-700 dark:text-slate-300">
                        Default Container Runtime
                      </label>
                      <p className="text-sm text-slate-600 dark:text-slate-400">
                        Choose between Docker and Apple Container runtime
                      </p>
                    </div>
                    <Select 
                      value={config?.container.default_runtime || "docker"}
                      onValueChange={(value) => config && setConfig({
                        ...config,
                        container: { ...config.container, default_runtime: value }
                      })}
                      disabled={configLoading}
                    >
                      <SelectTrigger className="w-48">
                        <SelectValue placeholder="Select runtime" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="docker">Docker</SelectItem>
                        <SelectItem value="apple">Apple Container</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="flex items-center justify-between">
                    <div>
                      <label className="font-medium text-slate-700 dark:text-slate-300">
                        Auto-start Containers
                      </label>
                      <p className="text-sm text-slate-600 dark:text-slate-400">
                        Automatically start containers when creating worktrees
                      </p>
                    </div>
                    <Checkbox 
                      checked={config?.container.auto_start ?? true}
                      onCheckedChange={(checked) => config && setConfig({
                        ...config,
                        container: { ...config.container, auto_start: checked as boolean }
                      })}
                      disabled={configLoading}
                      className="border-slate-400 data-[state=checked]:bg-gradient-to-r data-[state=checked]:from-cyan-500 data-[state=checked]:to-purple-500 data-[state=checked]:border-0"
                    />
                  </div>
                </div>
              </div>

              <div className="bg-white/90 dark:bg-slate-800/90 backdrop-blur-xl rounded-2xl border border-slate-200/50 dark:border-slate-700/50 p-6 shadow-lg">
                <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-4">
                  Storage Settings
                </h3>
                <div className="space-y-4">
                  <div>
                    <label className="font-medium text-slate-700 dark:text-slate-300 block mb-2">
                      Global Repositories Location
                    </label>
                    <p className="text-sm text-slate-600 dark:text-slate-400 mb-2">
                      Where bare Git repositories are stored
                    </p>
                    <div className="flex gap-2">
                      <Input 
                        type="text"
                        value={config?.storage.repositories_path || "~/vibeman/repos"}
                        onChange={(e) => config && setConfig({
                          ...config,
                          storage: { ...config.storage, repositories_path: e.target.value }
                        })}
                        className="flex-1 bg-slate-50 dark:bg-slate-900/50 border-slate-300 dark:border-slate-600"
                        placeholder="/path/to/repositories"
                        disabled={configLoading}
                      />
                      <Button 
                        variant="outline" 
                        className="border-slate-300 dark:border-slate-600 hover:bg-slate-100 dark:hover:bg-slate-700"
                      >
                        <Folder className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                      </Button>
                    </div>
                  </div>

                  <div>
                    <label className="font-medium text-slate-700 dark:text-slate-300 block mb-2">
                      Global Worktrees Location
                    </label>
                    <p className="text-sm text-slate-600 dark:text-slate-400 mb-2">
                      Default location for Git worktrees (can be overridden per repository)
                    </p>
                    <div className="flex gap-2">
                      <Input 
                        type="text"
                        value={config?.storage.worktrees_path || "~/vibeman/worktrees"}
                        onChange={(e) => config && setConfig({
                          ...config,
                          storage: { ...config.storage, worktrees_path: e.target.value }
                        })}
                        className="flex-1 bg-slate-50 dark:bg-slate-900/50 border-slate-300 dark:border-slate-600"
                        placeholder="/path/to/worktrees"
                        disabled={configLoading}
                      />
                      <Button 
                        variant="outline" 
                        className="border-slate-300 dark:border-slate-600 hover:bg-slate-100 dark:hover:bg-slate-700"
                      >
                        <Folder className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                      </Button>
                    </div>
                  </div>

                  <div className="pt-4 border-t border-slate-200 dark:border-slate-700">
                    <div className="flex items-start gap-2">
                      <Info className="w-4 h-4 text-slate-500 mt-0.5" />
                      <p className="text-sm text-slate-600 dark:text-slate-400">
                        These are default locations. Individual repositories can override these settings in their configuration files.
                      </p>
                    </div>
                  </div>
                </div>
              </div>

              <div className="bg-white/90 dark:bg-slate-800/90 backdrop-blur-xl rounded-2xl border border-slate-200/50 dark:border-slate-700/50 p-6 shadow-lg">
                <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-4">
                  Git Configuration
                </h3>
                <div className="space-y-4">
                  <div>
                    <label className="font-medium text-slate-700 dark:text-slate-300 block mb-2">
                      Default Branch Prefix
                    </label>
                    <p className="text-sm text-slate-600 dark:text-slate-400 mb-2">
                      Prefix for new worktree branches (e.g., feature/, bugfix/)
                    </p>
                    <Input 
                      type="text"
                      value={config?.git.default_branch_prefix || "feature/"}
                      onChange={(e) => config && setConfig({
                        ...config,
                        git: { ...config.git, default_branch_prefix: e.target.value }
                      })}
                      className="w-full bg-slate-50 dark:bg-slate-900/50 border-slate-300 dark:border-slate-600"
                      placeholder="feature/"
                      disabled={configLoading}
                    />
                  </div>

                  <div className="flex items-center justify-between">
                    <div>
                      <label className="font-medium text-slate-700 dark:text-slate-300">
                        Auto-fetch on Worktree Creation
                      </label>
                      <p className="text-sm text-slate-600 dark:text-slate-400">
                        Automatically fetch latest changes when creating worktrees
                      </p>
                    </div>
                    <Checkbox 
                      checked={config?.git.auto_fetch ?? true}
                      onCheckedChange={(checked) => config && setConfig({
                        ...config,
                        git: { ...config.git, auto_fetch: checked as boolean }
                      })}
                      disabled={configLoading}
                      className="border-slate-400 data-[state=checked]:bg-gradient-to-r data-[state=checked]:from-cyan-500 data-[state=checked]:to-purple-500 data-[state=checked]:border-0"
                    />
                  </div>
                </div>
              </div>

              <div className="bg-white/90 dark:bg-slate-800/90 backdrop-blur-xl rounded-2xl border border-slate-200/50 dark:border-slate-700/50 p-6 shadow-lg">
                <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-4">
                  Server Configuration
                </h3>
                <div className="space-y-4">
                  <div>
                    <label className="font-medium text-slate-700 dark:text-slate-300 block mb-2">
                      Server Port
                    </label>
                    <p className="text-sm text-slate-600 dark:text-slate-400 mb-2">
                      Port for the Vibeman API server
                    </p>
                    <Input 
                      type="number"
                      value={config?.server?.port || 8080}
                      onChange={(e) => config && setConfig({
                        ...config,
                        server: { ...config.server, port: parseInt(e.target.value) || 8080 }
                      })}
                      className="w-32 bg-slate-50 dark:bg-slate-900/50 border-slate-300 dark:border-slate-600"
                      placeholder="8080"
                      disabled={configLoading}
                      min="1"
                      max="65535"
                    />
                  </div>

                  <div>
                    <label className="font-medium text-slate-700 dark:text-slate-300 block mb-2">
                      Web UI Port
                    </label>
                    <p className="text-sm text-slate-600 dark:text-slate-400 mb-2">
                      Port for the Vibeman web interface
                    </p>
                    <Input 
                      type="number"
                      value={config?.server?.webui_port || 8081}
                      onChange={(e) => config && setConfig({
                        ...config,
                        server: { ...config.server, webui_port: parseInt(e.target.value) || 8081 }
                      })}
                      className="w-32 bg-slate-50 dark:bg-slate-900/50 border-slate-300 dark:border-slate-600"
                      placeholder="8081"
                      disabled={configLoading}
                      min="1"
                      max="65535"
                    />
                  </div>

                  <div>
                    <label className="font-medium text-slate-700 dark:text-slate-300 block mb-2">
                      Default Services Configuration
                    </label>
                    <p className="text-sm text-slate-600 dark:text-slate-400 mb-2">
                      Path to the global services.toml file
                    </p>
                    <div className="flex gap-2">
                      <Input 
                        type="text"
                        value={config?.server?.services_config_path || "~/vibeman/services.toml"}
                        onChange={(e) => config && setConfig({
                          ...config,
                          server: { ...config.server, services_config_path: e.target.value }
                        })}
                        className="flex-1 bg-slate-50 dark:bg-slate-900/50 border-slate-300 dark:border-slate-600"
                        placeholder="/path/to/services.toml"
                        disabled={configLoading}
                      />
                      <Button 
                        variant="outline" 
                        className="border-slate-300 dark:border-slate-600 hover:bg-slate-100 dark:hover:bg-slate-700"
                      >
                        <Folder className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                      </Button>
                    </div>
                  </div>
                </div>
              </div>

              <div className="bg-white/90 dark:bg-slate-800/90 backdrop-blur-xl rounded-2xl border border-slate-200/50 dark:border-slate-700/50 p-6 shadow-lg">
                <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-4">
                  Global Services Configuration
                </h3>
                <p className="text-sm text-slate-600 dark:text-slate-400 mb-4">
                  Services defined in the global services.toml file that can be shared across repositories
                </p>
                
                {config?.services ? (
                  <div className="space-y-3">
                    {Object.entries(config.services).map(([name, service]) => (
                      <div key={name} className="flex items-start justify-between p-4 bg-slate-50 dark:bg-slate-900/50 rounded-lg">
                        <div className="flex items-start gap-3">
                          <Database className="w-5 h-5 text-cyan-500 mt-0.5" />
                          <div className="flex-1">
                            <p className="font-medium text-slate-700 dark:text-slate-300">{name}</p>
                            {service.description && (
                              <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">{service.description}</p>
                            )}
                            <div className="mt-2 space-y-1">
                              <p className="text-xs text-slate-500 dark:text-slate-500">
                                <span className="font-medium">Compose file:</span> {service.compose_file}
                              </p>
                              <p className="text-xs text-slate-500 dark:text-slate-500">
                                <span className="font-medium">Service:</span> {service.service}
                              </p>
                            </div>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <span className={`px-2 py-1 text-xs rounded-full ${
                            services.find(s => s.name === name)?.status === 'running'
                              ? 'bg-green-100 text-green-700 dark:bg-green-900/20 dark:text-green-400'
                              : 'bg-gray-100 text-gray-700 dark:bg-gray-900/20 dark:text-gray-400'
                          }`}>
                            {services.find(s => s.name === name)?.status || 'Unknown'}
                          </span>
                        </div>
                      </div>
                    ))}
                    
                    {Object.keys(config.services).length === 0 && (
                      <div className="text-center py-8 text-sm text-slate-500 dark:text-slate-400">
                        No global services configured in services.toml
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="text-center py-8 text-sm text-slate-500 dark:text-slate-400">
                    No global services configured
                  </div>
                )}

                <div className="mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    <Info className="w-3 h-3 inline mr-1" />
                    Global services are configured in the services.toml file specified above. 
                    Edit that file directly to add or modify services.
                  </p>
                </div>
              </div>

              <div className="flex justify-end gap-3 mt-6">
                <Button 
                  variant="outline"
                  onClick={resetConfig}
                  disabled={configLoading || configSaving}
                >
                  Reset to Defaults
                </Button>
                <Button 
                  className="bg-gradient-to-r from-cyan-500 to-purple-500 text-white hover:from-cyan-600 hover:to-purple-600 disabled:opacity-50"
                  onClick={saveConfig}
                  disabled={configLoading || configSaving}
                >
                  {configSaving ? 'Saving...' : 'Save Settings'}
                </Button>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
        )}
      </main>
      
      <CreateRepositoryModal 
        open={isCreateRepoModalOpen}
        onOpenChange={setIsCreateRepoModalOpen}
        onSuccess={() => {
          // Data is automatically refreshed by React Query
          setIsCreateRepoModalOpen(false);
        }}
        createProject={async (data) => {
          try {
            await createRepository({
              name: data.body.name,
              path: data.body.repository_url || data.body.git_url,
              description: data.body.description,
            });
          } catch (error) {
            console.error('Failed to create repository:', error);
            throw error;
          }
        }}
        isCreating={false}
      />
      
      <CreateWorktreeModal
        open={isCreateWorktreeModalOpen}
        onOpenChange={setIsCreateWorktreeModalOpen}
        onSuccess={() => {
          // React Query will automatically refetch data
        }}
        repositories={repositories}
      />
      
      <TerminalModal
        open={terminalWorktreeId !== null}
        onOpenChange={(open) => {
          if (!open) {
            setTerminalWorktreeId(null);
            setTerminalWorktreeName(null);
          }
        }}
        environmentId={terminalWorktreeId || ''}
        worktreeName={terminalWorktreeName || undefined}
      />
      
      <LogModal
        open={logWorktreeId !== null}
        onOpenChange={(open) => {
          if (!open) {
            setLogWorktreeId(null);
            setLogWorktreeName(null);
          }
        }}
        environmentId={logWorktreeId || ''}
        worktreeName={logWorktreeName || undefined}
      />
      
      <RemoveWorktreeDialog
        open={removeWorktreeId !== null}
        onOpenChange={(open) => {
          if (!open) {
            setRemoveWorktreeId(null);
            setRemoveWorktreeData(null);
          }
        }}
        worktreeName={removeWorktreeData?.name || ''}
        branchName={removeWorktreeData?.branch || ''}
        onConfirm={() => {
          if (removeWorktreeId) {
            handleWorktreeAction(removeWorktreeId, "delete");
            setRemoveWorktreeId(null);
            setRemoveWorktreeData(null);
          }
        }}
        isRemoving={isDeletingWorktree}
      />
      
      <RemoveRepositoryDialog
        open={removeRepositoryId !== null}
        onOpenChange={(open) => {
          if (!open) {
            setRemoveRepositoryId(null);
            setRemoveRepositoryData(null);
          }
        }}
        repositoryName={removeRepositoryData?.name || ''}
        repositoryPath={removeRepositoryData?.path || ''}
        activeWorktrees={removeRepositoryId ? worktrees
          .filter(wt => wt.repository_id === removeRepositoryId)
          .map(wt => ({ id: wt.id, name: wt.name, branch: wt.branch })) : []}
        onConfirm={async (removeType) => {
          if (removeRepositoryId) {
            try {
              await deleteRepository(removeRepositoryId);
              setRemoveRepositoryId(null);
              setRemoveRepositoryData(null);
              toast({
                title: 'Repository removed',
                description: `${removeRepositoryData?.name} has been removed.`,
                type: 'success'
              });
            } catch (err) {
              console.error('Failed to remove repository:', err);
              toast({
                title: 'Failed to remove repository',
                description: err instanceof Error ? err.message : 'An error occurred',
                type: 'error'
              });
            }
          }
        }}
        isRemoving={false}
      />
      
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </div>
  );
};
export default VibemanManagementUI;
