import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { client } from '@/generated/api/client.gen';
import { 
  getServicesOptions,
  postServicesByIdStartMutation,
  postServicesByIdStopMutation
} from '@/generated/api/@tanstack/react-query.gen';
import type { ServerService } from '@/generated/api/types.gen';

interface UseServicesOptions {
  status?: 'stopped' | 'starting' | 'running' | 'stopping' | 'error';
  type?: 'database' | 'cache' | 'queue' | 'other';
}

export function useServices(options?: UseServicesOptions) {
  const queryClient = useQueryClient();

  const query = useQuery({
    ...getServicesOptions(),
    select: (data) => {
      let services = data?.services || [];
      
      // Apply client-side filtering if options are provided
      if (options?.status) {
        services = services.filter((s: ServerService) => s.status === options.status);
      }
      if (options?.type) {
        services = services.filter((s: ServerService) => s.type === options.type);
      }
      
      return services;
    },
  });

  const startMutation = useMutation({
    ...postServicesByIdStartMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['getServices'] });
    },
  });

  const stopMutation = useMutation({
    ...postServicesByIdStopMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['getServices'] });
    },
  });

  const restartMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await client.POST('/services/{id}/restart', {
        path: { id },
      });
      if (response.error) {
        throw new Error(response.error.error || 'Failed to restart service');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['getServices'] });
    },
  });

  return {
    services: query.data || [],
    isLoading: query.isLoading,
    error: query.error,
    refetch: query.refetch,
    startService: (id: string) => startMutation.mutate({ path: { id } }),
    stopService: (id: string) => stopMutation.mutate({ path: { id } }),
    restartService: (id: string) => restartMutation.mutate(id),
    isStarting: startMutation.isPending,
    isStopping: stopMutation.isPending,
    isRestarting: restartMutation.isPending,
  };
}