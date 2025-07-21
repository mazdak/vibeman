import { useMutation, useQueryClient } from '@tanstack/react-query';
import { postAuthLoginMutation, postAuthRefreshMutation } from '@/generated/api/@tanstack/react-query.gen';
import { setAuthToken } from '@/lib/api-client';
import { useState, useEffect } from 'react';

export function useAuth() {
  const queryClient = useQueryClient();
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  // Check if authenticated on mount
  useEffect(() => {
    const token = localStorage.getItem('auth_token');
    setIsAuthenticated(!!token);
  }, []);

  const loginMutation = useMutation({
    ...postAuthLoginMutation(),
    onSuccess: (data) => {
      if (data?.token) {
        setAuthToken(data.token);
        setIsAuthenticated(true);
        
        // Clear all queries and refetch
        queryClient.clear();
      }
    },
    onError: () => {
      setIsAuthenticated(false);
    },
  });

  const refreshMutation = useMutation({
    ...postAuthRefreshMutation(),
    onSuccess: (data) => {
      if (data?.token) {
        setAuthToken(data.token);
        setIsAuthenticated(true);
      }
    },
    onError: () => {
      // Token refresh failed, log out
      logout();
    },
  });

  const logout = () => {
    setAuthToken(null);
    setIsAuthenticated(false);
    queryClient.clear();
    // Optionally redirect to login page
  };

  return {
    isAuthenticated,
    login: loginMutation.mutate,
    isLoggingIn: loginMutation.isPending,
    loginError: loginMutation.error,
    refresh: refreshMutation.mutate,
    isRefreshing: refreshMutation.isPending,
    logout,
  };
}