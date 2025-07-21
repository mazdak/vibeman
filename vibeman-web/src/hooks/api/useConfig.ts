import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getConfigOptions, getConfigQueryKey } from '@/generated/api/@tanstack/react-query.gen';
import type { ServerConfigResponse } from '@/generated/api/types.gen';

export function useConfig() {
  const queryClient = useQueryClient();

  const query = useQuery({
    ...getConfigOptions(),
  });

  // Since updateConfig isn't in the generated SDK yet, we'll handle it manually
  const updateMutation = useMutation({
    mutationFn: async (config: ServerConfigResponse) => {
      const token = localStorage.getItem('auth_token');
      const response = await fetch('/api/config', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          ...(token && { Authorization: `Bearer ${token}` }),
        },
        body: JSON.stringify(config),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return response.json();
    },
    onSuccess: (updatedConfig) => {
      // Update the cache with the new config
      queryClient.setQueryData(getConfigQueryKey(), updatedConfig);
      
      // Invalidate to ensure fresh data
      queryClient.invalidateQueries({
        queryKey: getConfigQueryKey(),
      });
    },
  });

  return {
    config: query.data,
    isLoading: query.isLoading,
    error: query.error,
    refetch: query.refetch,
    updateConfig: updateMutation.mutate,
    isUpdating: updateMutation.isPending,
  };
}