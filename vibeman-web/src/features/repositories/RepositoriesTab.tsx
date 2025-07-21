import React, { useState } from 'react';
import {
  Database,
  Plus,
  ExternalLink,
  Calendar,
  RotateCw,
  GitBranch,
  Loader2,
  AlertCircle,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { CreateRepositoryModal } from '@/components/CreateRepositoryModal';
import { useAppState } from '@/shared/context/AppStateContext';
import { useProjects } from '@/hooks/api/useProjects';

export function RepositoriesTab() {
  const { setSelectedProject } = useAppState();
  const { 
    projects, 
    isLoading, 
    error, 
    refetch, 
    createProject, 
    isCreating 
  } = useProjects();
  const [isCreateRepoModalOpen, setIsCreateRepoModalOpen] = useState(false);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-semibold text-slate-800 dark:text-slate-200">Repositories</h2>
          <p className="text-slate-600 dark:text-slate-400 mt-1">
            Manage your Git repositories and create new worktrees
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
          <Button
            onClick={() => setIsCreateRepoModalOpen(true)}
            className="bg-gradient-to-r from-cyan-500 to-purple-500 text-white hover:from-cyan-600 hover:to-purple-600"
          >
            <Plus className="w-4 h-4 mr-2" />
            Add Repository
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center p-16">
          <Loader2 className="w-8 h-8 animate-spin text-cyan-500" />
        </div>
      ) : error ? (
        <Card className="p-8 text-center border-red-200 dark:border-red-900/50 bg-red-50 dark:bg-red-900/20">
          <AlertCircle className="w-12 h-12 mx-auto mb-4 text-red-500" />
          <h3 className="text-xl font-semibold mb-2 text-red-700 dark:text-red-300">Error Loading Repositories</h3>
          <p className="text-red-600 dark:text-red-400 mb-4">
            {error.message || 'Failed to load repositories'}
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
      ) : projects.length === 0 ? (
        <Card className="p-16 text-center border-2 border-dashed border-slate-200 dark:border-slate-700/50 bg-white/50 dark:bg-slate-800/30 backdrop-blur-sm shadow-none">
          <div className="max-w-md mx-auto">
            <Database className="w-16 h-16 mx-auto mb-4 text-slate-400 dark:text-slate-600" />
            <h3 className="text-xl font-semibold mb-2 text-slate-700 dark:text-slate-300">No Repositories</h3>
            <p className="text-slate-600 dark:text-slate-400 mb-6">
              Add your first repository to start creating worktrees
            </p>
            <Button
              onClick={() => setIsCreateRepoModalOpen(true)}
              className="bg-gradient-to-r from-cyan-500 to-purple-500 text-white hover:from-cyan-600 hover:to-purple-600"
            >
              <Plus className="w-4 h-4 mr-2" />
              Add Repository
            </Button>
          </div>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {projects.map((project) => (
            <Card
              key={project.id}
              className="overflow-hidden border-slate-200 dark:border-slate-700/50 bg-white/70 dark:bg-slate-800/50 backdrop-blur-sm hover:shadow-lg transition-all duration-200 cursor-pointer"
              onClick={() => setSelectedProject(project)}
            >
              <div className="p-6">
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-start gap-3">
                    <div className="p-2 rounded-lg bg-gradient-to-br from-cyan-500/10 to-purple-500/10 dark:from-cyan-500/20 dark:to-purple-500/20">
                      <GitBranch className="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <h3 className="font-semibold text-slate-800 dark:text-slate-200 truncate">{project.name}</h3>
                      <p className="text-sm text-slate-600 dark:text-slate-400 truncate">
                        {project.git_url || project.repository_url}
                      </p>
                    </div>
                  </div>
                  <a
                    href={project.repository_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <ExternalLink className="w-4 h-4" />
                  </a>
                </div>

                {project.description && (
                  <p className="text-sm text-slate-600 dark:text-slate-400 mb-4 line-clamp-2">
                    {project.description}
                  </p>
                )}

                <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-500">
                  <Calendar className="w-3 h-3" />
                  <span>Added {new Date(project.created_at).toLocaleDateString()}</span>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      <CreateRepositoryModal
        open={isCreateRepoModalOpen}
        onOpenChange={setIsCreateRepoModalOpen}
        onSuccess={() => {
          setIsCreateRepoModalOpen(false);
          refetch();
        }}
        createProject={createProject}
        isCreating={isCreating}
      />
    </div>
  );
}