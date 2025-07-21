"use client";

import React, { useState, useEffect } from "react";
import { AnimatePresence } from "framer-motion";
import {
  GitBranch,
  Database,
  Settings,
  Moon,
  Sun,
  Monitor,
  AlertCircle,
  Loader2,
} from "lucide-react";
import logo from "@/logo.png";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { AppStateProvider, useAppState } from "@/shared/context/AppStateContext";
import { WorktreesTab } from "@/features/worktrees/WorktreesTab";
import { RepositoriesTab } from "@/features/repositories/RepositoriesTab";
import { SettingsTab } from "@/features/settings/SettingsTab";

type ThemeMode = 'light' | 'dark' | 'system';

function VibemanUIContent() {
  // AppState now only manages selectedProject and mountedRef - loading/error are handled by React Query
  
  // Tab state
  const [activeTab, setActiveTab] = useState<"worktrees" | "repositories" | "settings">("worktrees");
  
  // Theme state with localStorage persistence
  const [themeMode, setThemeMode] = useState<ThemeMode>(() => {
    try {
      const savedTheme = localStorage.getItem('vibeman-theme') as ThemeMode;
      return savedTheme || 'system';
    } catch {
      return 'system';
    }
  });
  const [systemPrefersDark, setSystemPrefersDark] = useState(false);

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

  const cycleTheme = () => {
    const modes: ThemeMode[] = ['light', 'dark', 'system'];
    const currentIndex = modes.indexOf(themeMode);
    const nextIndex = (currentIndex + 1) % modes.length;
    const newTheme = modes[nextIndex];
    setThemeMode(newTheme);
    
    // Persist theme selection to localStorage
    try {
      localStorage.setItem('vibeman-theme', newTheme);
    } catch (error) {
      console.warn('Failed to save theme preference:', error);
    }
  };

  const getThemeIcon = () => {
    if (themeMode === 'system') return Monitor;
    if (themeMode === 'dark') return Moon;
    return Sun;
  };

  const tabs = [
    { id: "worktrees" as const, label: "Worktrees", icon: GitBranch },
    { id: "repositories" as const, label: "Repositories", icon: Database },
    { id: "settings" as const, label: "Settings", icon: Settings },
  ];

  const ThemeIcon = getThemeIcon();

  return (
    <div className={`min-h-screen w-full transition-colors duration-150 ${themeMode === 'dark' || (themeMode === 'system' && systemPrefersDark) ? "bg-slate-900" : "bg-gradient-to-br from-slate-50 to-slate-100"}`}>
      {/* Header */}
      <header className="bg-white/90 dark:bg-slate-800/90 backdrop-blur-xl border-b border-slate-200/50 dark:border-slate-700/50 sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            {/* Logo and Title */}
            <div className="flex items-center gap-2">
              <div className="w-16 h-16">
                <img
                  src={logo}
                  alt="Vibeman Logo"
                  className="w-full h-full object-contain"
                />
              </div>
              <div className="flex flex-col justify-center mt-3">
                <h1 className="text-2xl font-bold bg-gradient-to-r from-cyan-500 to-purple-500 bg-clip-text text-transparent tracking-wider leading-tight">
                  VIBEMAN
                </h1>
                <p className="text-sm text-slate-600 dark:text-slate-400 font-medium tracking-wide -mt-1">
                  Manage your Vibe
                </p>
              </div>
            </div>
            
            {/* Header Actions */}
            <div className="flex items-center gap-4">
              <Button
                variant="outline"
                size="sm"
                onClick={cycleTheme}
                className="border-slate-300 dark:border-slate-600"
              >
                <ThemeIcon className="w-4 h-4" />
              </Button>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto p-6">
        {/* Navigation Tabs */}
        <div className="flex gap-1 p-1 bg-slate-200/80 dark:bg-slate-800/50 rounded-lg backdrop-blur-sm mb-6">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            const isActive = activeTab === tab.id;
            
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`
                  flex items-center gap-2 px-4 py-2 rounded-md font-medium transition-all duration-200
                  ${isActive 
                    ? 'bg-white dark:bg-slate-700 text-cyan-600 dark:text-cyan-400 shadow-sm' 
                    : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100'
                  }
                `}
              >
                <Icon className="w-4 h-4" />
                {tab.label}
              </button>
            );
          })}
        </div>

        {/* Tab Content - loading/error states are now handled individually by each tab using React Query */}
        <AnimatePresence mode="wait">
          {activeTab === "worktrees" && <WorktreesTab key="worktrees" />}
          {activeTab === "repositories" && <RepositoriesTab key="repositories" />}
          {activeTab === "settings" && <SettingsTab key="settings" />}
        </AnimatePresence>
      </div>
    </div>
  );
}

export default function VibemanUI() {
  return (
    <AppStateProvider>
      <VibemanUIContent />
    </AppStateProvider>
  );
}