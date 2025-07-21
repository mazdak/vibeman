import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import {
  Settings,
  Container,
  FolderOpen,
  GitBranch,
  Database,
  Plus,
  Save,
  RotateCw,
  Info,
  Loader2,
  AlertCircle,
  Check,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useConfig } from '@/hooks/api/useConfig';
import type { ServerConfigResponse } from '@/generated/api/types.gen';

export function SettingsTab() {
  const { 
    config: originalConfig, 
    isLoading, 
    error, 
    refetch, 
    updateConfig, 
    isUpdating 
  } = useConfig();

  // Local state for form editing
  const [localConfig, setLocalConfig] = useState<ServerConfigResponse | null>(null);
  const [hasChanges, setHasChanges] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);

  // Update local config when original config loads
  useEffect(() => {
    if (originalConfig) {
      setLocalConfig(originalConfig);
      setHasChanges(false);
    }
  }, [originalConfig]);

  // Check if config has changed
  useEffect(() => {
    if (originalConfig && localConfig) {
      const changed = JSON.stringify(originalConfig) !== JSON.stringify(localConfig);
      setHasChanges(changed);
    }
  }, [originalConfig, localConfig]);

  // Clear success message after a delay
  useEffect(() => {
    if (saveSuccess) {
      const timer = setTimeout(() => setSaveSuccess(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [saveSuccess]);

  const handleSave = () => {
    if (localConfig) {
      updateConfig(localConfig, {
        onSuccess: () => {
          setSaveSuccess(true);
          setHasChanges(false);
        }
      });
    }
  };

  const handleReset = () => {
    if (originalConfig) {
      setLocalConfig(originalConfig);
      setHasChanges(false);
    }
  };

  const updateConfigField = (path: string[], value: any) => {
    if (!localConfig) return;
    
    const newConfig = { ...localConfig };
    let current: any = newConfig;
    
    for (let i = 0; i < path.length - 1; i++) {
      if (!current[path[i]]) {
        current[path[i]] = {};
      }
      current = current[path[i]];
    }
    
    current[path[path.length - 1]] = value;
    setLocalConfig(newConfig);
  };

  if (isLoading) {
    return (
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -10 }}
        transition={{ duration: 0.2 }}
      >
        <div className="flex justify-center items-center h-64">
          <Loader2 className="w-8 h-8 animate-spin text-cyan-500" />
        </div>
      </motion.div>
    );
  }

  if (error && !localConfig) {
    return (
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -10 }}
        transition={{ duration: 0.2 }}
      >
        <Card className="p-8 text-center border-red-200 dark:border-red-900/50 bg-red-50 dark:bg-red-900/20">
          <AlertCircle className="w-12 h-12 mx-auto mb-4 text-red-500" />
          <h3 className="text-xl font-semibold mb-2 text-red-700 dark:text-red-300">Error Loading Configuration</h3>
          <p className="text-red-600 dark:text-red-400 mb-4">
            {error.message || 'Failed to load configuration'}
          </p>
          <Button
            onClick={() => refetch()}
            variant="outline"
            className="border-red-300 text-red-700 hover:bg-red-100 dark:border-red-700 dark:text-red-300 dark:hover:bg-red-900/30"
          >
            <RotateCw className="w-4 h-4 mr-2" />
            Try Again
          </Button>
        </Card>
      </motion.div>
    );
  }

  if (!localConfig) return null;

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -10 }}
      transition={{ duration: 0.2 }}
      className="space-y-6"
    >
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-semibold text-slate-800 dark:text-slate-200">Settings</h2>
          <p className="text-slate-600 dark:text-slate-400 mt-1">
            Configure your Vibeman development environment
          </p>
        </div>
        
        <div className="flex gap-3">
          <Button
            variant="outline"
            onClick={() => refetch()}
            className="border-slate-300 dark:border-slate-600"
            disabled={isLoading}
          >
            <RotateCw className={`w-4 h-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
          {hasChanges && (
            <Button
              variant="outline"
              onClick={handleReset}
              className="border-slate-300 dark:border-slate-600"
            >
              Reset Changes
            </Button>
          )}
          <Button
            onClick={handleSave}
            disabled={!hasChanges || isUpdating}
            className="bg-gradient-to-r from-cyan-500 to-purple-500 text-white hover:from-cyan-600 hover:to-purple-600 disabled:opacity-50 disabled:cursor-not-allowed min-w-[130px]"
          >
            {isUpdating ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Saving...
              </>
            ) : saveSuccess ? (
              <>
                <Check className="w-4 h-4 mr-2" />
                Saved!
              </>
            ) : (
              <>
                Save Changes
              </>
            )}
          </Button>
        </div>
      </div>

      {error && (
        <Card className="p-4 border-yellow-200 dark:border-yellow-900/50 bg-yellow-50 dark:bg-yellow-900/20">
          <div className="flex items-start gap-3">
            <Info className="w-5 h-5 text-yellow-600 dark:text-yellow-400 mt-0.5" />
            <div>
              <p className="text-sm text-yellow-700 dark:text-yellow-300">
                Unable to connect to backend. Changes won't be saved.
              </p>
            </div>
          </div>
        </Card>
      )}

      <div className="grid gap-6">
        {/* Storage Settings */}
        <Card className="p-6 border-slate-200 dark:border-slate-700/50 bg-white/70 dark:bg-slate-800/50 backdrop-blur-sm">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-gradient-to-br from-cyan-500/10 to-purple-500/10 dark:from-cyan-500/20 dark:to-purple-500/20">
              <FolderOpen className="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
            </div>
            <h3 className="text-lg font-semibold text-slate-800 dark:text-slate-200">Storage Configuration</h3>
          </div>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="repos-path" className="text-slate-700 dark:text-slate-300">Repositories Path</Label>
              <Input
                id="repos-path"
                type="text"
                value={localConfig.storage?.repositories_path || ''}
                onChange={(e) => updateConfigField(['storage', 'repositories_path'], e.target.value)}
                placeholder="~/vibeman/repos"
                className="mt-1"
              />
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                Where cloned repositories are stored
              </p>
            </div>
            
            <div>
              <Label htmlFor="worktrees-path" className="text-slate-700 dark:text-slate-300">Worktrees Path</Label>
              <Input
                id="worktrees-path"
                type="text"
                value={localConfig.storage?.worktrees_path || ''}
                onChange={(e) => updateConfigField(['storage', 'worktrees_path'], e.target.value)}
                placeholder="~/vibeman/worktrees"
                className="mt-1"
              />
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                Where Git worktrees are created
              </p>
            </div>
          </div>
        </Card>

        {/* Git Settings */}
        <Card className="p-6 border-slate-200 dark:border-slate-700/50 bg-white/70 dark:bg-slate-800/50 backdrop-blur-sm">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-gradient-to-br from-cyan-500/10 to-purple-500/10 dark:from-cyan-500/20 dark:to-purple-500/20">
              <GitBranch className="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
            </div>
            <h3 className="text-lg font-semibold text-slate-800 dark:text-slate-200">Git Configuration</h3>
          </div>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="branch-prefix" className="text-slate-700 dark:text-slate-300">Default Branch Prefix</Label>
              <Input
                id="branch-prefix"
                type="text"
                value={localConfig.git?.default_branch_prefix || ''}
                onChange={(e) => updateConfigField(['git', 'default_branch_prefix'], e.target.value)}
                placeholder="feature/"
                className="mt-1"
              />
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                Prefix for new branches created from worktrees
              </p>
            </div>
            
            <div className="flex items-center space-x-2">
              <Checkbox
                id="auto-fetch"
                checked={localConfig.git?.auto_fetch || false}
                onCheckedChange={(checked) => updateConfigField(['git', 'auto_fetch'], checked)}
              />
              <Label
                htmlFor="auto-fetch"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 text-slate-700 dark:text-slate-300"
              >
                Automatically fetch from remote repositories
              </Label>
            </div>
          </div>
        </Card>

        {/* Container Settings */}
        <Card className="p-6 border-slate-200 dark:border-slate-700/50 bg-white/70 dark:bg-slate-800/50 backdrop-blur-sm">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-gradient-to-br from-cyan-500/10 to-purple-500/10 dark:from-cyan-500/20 dark:to-purple-500/20">
              <Container className="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
            </div>
            <h3 className="text-lg font-semibold text-slate-800 dark:text-slate-200">Container Configuration</h3>
          </div>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="runtime" className="text-slate-700 dark:text-slate-300">Default Runtime</Label>
              <Select
                value={localConfig.container?.default_runtime || 'docker'}
                onValueChange={(value) => updateConfigField(['container', 'default_runtime'], value)}
              >
                <SelectTrigger id="runtime" className="w-full mt-1">
                  <SelectValue placeholder="Select runtime" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="docker">Docker</SelectItem>
                  <SelectItem value="apple">Apple Container</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                Container runtime to use for development environments
              </p>
            </div>
            
            <div className="flex items-center space-x-2">
              <Checkbox
                id="auto-start"
                checked={localConfig.container?.auto_start || false}
                onCheckedChange={(checked) => updateConfigField(['container', 'auto_start'], checked)}
              />
              <Label
                htmlFor="auto-start"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 text-slate-700 dark:text-slate-300"
              >
                Automatically start containers when creating worktrees
              </Label>
            </div>
          </div>
        </Card>

        {/* Shared Services Settings */}
        <Card className="p-6 border-slate-200 dark:border-slate-700/50 bg-white/70 dark:bg-slate-800/50 backdrop-blur-sm">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-gradient-to-br from-cyan-500/10 to-purple-500/10 dark:from-cyan-500/20 dark:to-purple-500/20">
              <Database className="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
            </div>
            <h3 className="text-lg font-semibold text-slate-800 dark:text-slate-200">Shared Services Configuration</h3>
          </div>
          
          <div className="space-y-4">
            <div>
              <Label htmlFor="services-path" className="text-slate-700 dark:text-slate-300">Services Data Path</Label>
              <Input
                id="services-path"
                type="text"
                value={localConfig.services?.data_path || ''}
                onChange={(e) => updateConfigField(['services', 'data_path'], e.target.value)}
                placeholder="~/vibeman/services/data"
                className="mt-1"
              />
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                Where shared service data (databases, caches) is stored
              </p>
            </div>
            
            <div>
              <Label htmlFor="services-port-start" className="text-slate-700 dark:text-slate-300">Service Port Range</Label>
              <div className="flex gap-2 items-center mt-1">
                <Input
                  id="services-port-start"
                  type="number"
                  value={localConfig.services?.port_range_start || 5432}
                  onChange={(e) => updateConfigField(['services', 'port_range_start'], parseInt(e.target.value))}
                  placeholder="5432"
                  className="w-24"
                />
                <span className="text-slate-500 dark:text-slate-400">to</span>
                <Input
                  id="services-port-end"
                  type="number"
                  value={localConfig.services?.port_range_end || 5500}
                  onChange={(e) => updateConfigField(['services', 'port_range_end'], parseInt(e.target.value))}
                  placeholder="5500"
                  className="w-24"
                />
              </div>
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                Port range for allocating service ports
              </p>
            </div>
            
            <div className="flex items-center space-x-2">
              <Checkbox
                id="services-auto-start"
                checked={localConfig.services?.auto_start || false}
                onCheckedChange={(checked) => updateConfigField(['services', 'auto_start'], checked)}
              />
              <Label
                htmlFor="services-auto-start"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 text-slate-700 dark:text-slate-300"
              >
                Automatically start required services when starting worktrees
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="services-persist-data"
                checked={localConfig.services?.persist_data || true}
                onCheckedChange={(checked) => updateConfigField(['services', 'persist_data'], checked)}
              />
              <Label
                htmlFor="services-persist-data"
                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 text-slate-700 dark:text-slate-300"
              >
                Persist service data between restarts
              </Label>
            </div>
          </div>
        </Card>
      </div>
    </motion.div>
  );
}