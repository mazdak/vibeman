import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getRepositoriesOptions,
  postRepositoriesMutation,
  deleteRepositoriesByIdMutation,
} from '@/generated/api/@tanstack/react-query.gen';
import type { DbRepository } from '@/generated/api/types.gen';
import { transformRepository } from '@/lib/api-client';

export function useRepositories() {
  const queryClient = useQueryClient();

  const query = useQuery({
    ...getRepositoriesOptions(),
    select: (data) => (data?.repositories || []).map(transformRepository),
  });

  const createMutation = useMutation({
    ...postRepositoriesMutation(),
    onSuccess: () => {
      // Invalidate repositories query
      queryClient.invalidateQueries({
        queryKey: ['getRepositories'],
      });
    },
  });

  const deleteMutation = useMutation({
    ...deleteRepositoriesByIdMutation(),
    onSuccess: () => {
      // Invalidate repositories query
      queryClient.invalidateQueries({
        queryKey: ['getRepositories'],
      });
    },
  });

  return {
    repositories: query.data || [],
    isLoading: query.isLoading,
    error: query.error,
    refetch: query.refetch,
    createRepository: createMutation.mutate,
    isCreating: createMutation.isPending,
    deleteRepository: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
  };
}