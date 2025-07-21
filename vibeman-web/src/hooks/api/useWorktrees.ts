import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getWorktreesOptions,
  getWorktreesQueryKey,
  postWorktreesMutation,
  postWorktreesByIdStartMutation,
  postWorktreesByIdStopMutation,
} from '@/generated/api/@tanstack/react-query.gen';
import type { DbWorktree } from '@/generated/api/types.gen';
import { transformWorktree, type UIWorktree, client } from '@/lib/api-client';

interface UseWorktreesOptions {
  repositoryId?: string;
}

export function useWorktrees(options?: UseWorktreesOptions) {
  const queryClient = useQueryClient();

  const query = useQuery({
    ...getWorktreesOptions({
      query: options?.repositoryId ? { repository_id: options.repositoryId } : undefined,
    }),
    select: (data) => {
      const worktrees = data?.worktrees || [];
      return worktrees.map(transformWorktree);
    },
  });

  const createMutation = useMutation({
    ...postWorktreesMutation(),
    onSuccess: () => {
      // Invalidate all worktree queries
      queryClient.invalidateQueries({
        queryKey: getWorktreesQueryKey({
          query: options?.repositoryId ? { repository_id: options.repositoryId } : undefined,
        }),
      });
    },
  });

  const startMutation = useMutation({
    ...postWorktreesByIdStartMutation(),
    onSuccess: (_, variables) => {
      // Invalidate worktree queries
      queryClient.invalidateQueries({
        queryKey: getWorktreesQueryKey({
          query: options?.repositoryId ? { repository_id: options.repositoryId } : undefined,
        }),
      });
      
      // Update the specific worktree status
      const queryKey = getWorktreesQueryKey({
        query: options?.repositoryId ? { repository_id: options.repositoryId } : undefined,
      });
      queryClient.setQueryData(queryKey, (old: any) => {
        if (!old?.worktrees) return old;
        
        return {
          ...old,
          worktrees: old.worktrees.map((worktree: DbWorktree) =>
            worktree.id === variables.path.id
              ? { ...worktree, status: 'running' }
              : worktree
          ),
        };
      });
    },
  });

  const stopMutation = useMutation({
    ...postWorktreesByIdStopMutation(),
    onSuccess: (_, variables) => {
      // Invalidate worktree queries
      queryClient.invalidateQueries({
        queryKey: getWorktreesQueryKey({
          query: options?.repositoryId ? { repository_id: options.repositoryId } : undefined,
        }),
      });
      
      // Update the specific worktree status
      const queryKey = getWorktreesQueryKey({
        query: options?.repositoryId ? { repository_id: options.repositoryId } : undefined,
      });
      queryClient.setQueryData(queryKey, (old: any) => {
        if (!old?.worktrees) return old;
        
        return {
          ...old,
          worktrees: old.worktrees.map((worktree: DbWorktree) =>
            worktree.id === variables.path.id
              ? { ...worktree, status: 'stopped' }
              : worktree
          ),
        };
      });
    },
  });

  // Custom delete mutation (no generated endpoint yet)
  const deleteMutation = useMutation({
    mutationFn: async (worktreeId: string) => {
      const response = await client.DELETE('/worktrees/{id}', {
        path: { id: worktreeId },
      });
      if (response.error) {
        throw new Error(response.error.error || 'Failed to delete worktree');
      }
      return response.data;
    },
    onSuccess: () => {
      // Invalidate all worktree queries to refetch data
      queryClient.invalidateQueries({
        queryKey: getWorktreesQueryKey({
          query: options?.repositoryId ? { repository_id: options.repositoryId } : undefined,
        }),
      });
    },
  });

  return {
    worktrees: query.data || [] as UIWorktree[],
    isLoading: query.isLoading,
    error: query.error,
    refetch: query.refetch,
    createWorktree: createMutation.mutate,
    isCreating: createMutation.isPending,
    createError: createMutation.error,
    startWorktree: startMutation.mutate,
    isStarting: startMutation.isPending,
    startError: startMutation.error,
    stopWorktree: stopMutation.mutate,
    isStopping: stopMutation.isPending,
    stopError: stopMutation.error,
    deleteWorktree: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
    deleteError: deleteMutation.error,
  };
}