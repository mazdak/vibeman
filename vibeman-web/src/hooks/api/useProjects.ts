import { useRepositories } from './useRepositories';

// Legacy alias for backward compatibility
// The API actually uses "repositories" but the UI still references "projects" in some places
export function useProjects() {
  const {
    repositories,
    isLoading,
    error,
    refetch,
    createRepository,
    isCreating,
    deleteRepository,
    isDeleting,
  } = useRepositories();

  return {
    projects: repositories,
    isLoading,
    error,
    refetch,
    createProject: createRepository,
    isCreating,
    deleteProject: deleteRepository,
    isDeleting,
  };
}