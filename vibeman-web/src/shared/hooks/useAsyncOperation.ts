import { useCallback, useState } from 'react';
import { useMounted } from './useMounted';

export interface AsyncOperationState {
  isLoading: boolean;
  error: string | null;
  data: any;
}

export interface UseAsyncOperationOptions<T> {
  onSuccess?: (data: T) => void;
  onError?: (error: string) => void;
  initialData?: T;
}

export interface UseAsyncOperationReturn<T> {
  state: AsyncOperationState;
  execute: (...args: any[]) => Promise<T | undefined>;
  reset: () => void;
  setData: (data: T) => void;
  setError: (error: string | null) => void;
}

/**
 * Hook for managing async operations with loading, error, and success states
 */
export function useAsyncOperation<T = any>(
  asyncFunction: (...args: any[]) => Promise<T>,
  options: UseAsyncOperationOptions<T> = {}
): UseAsyncOperationReturn<T> {
  const { onSuccess, onError, initialData } = options;
  const mountedRef = useMounted();

  const [state, setState] = useState<AsyncOperationState>({
    isLoading: false,
    error: null,
    data: initialData
  });

  const execute = useCallback(async (...args: any[]): Promise<T | undefined> => {
    if (!mountedRef.current) return;

    setState(prev => ({
      ...prev,
      isLoading: true,
      error: null
    }));

    try {
      const result = await asyncFunction(...args);
      
      if (mountedRef.current) {
        setState(prev => ({
          ...prev,
          isLoading: false,
          data: result,
          error: null
        }));
        onSuccess?.(result);
      }
      
      return result;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'An error occurred';
      
      if (mountedRef.current) {
        setState(prev => ({
          ...prev,
          isLoading: false,
          error: errorMessage
        }));
      }
      
      onError?.(errorMessage);
      return undefined;
    }
  }, [asyncFunction, onSuccess, onError, mountedRef]);

  const reset = useCallback(() => {
    if (mountedRef.current) {
      setState({
        isLoading: false,
        error: null,
        data: initialData
      });
    }
  }, [initialData, mountedRef]);

  const setData = useCallback((data: T) => {
    if (mountedRef.current) {
      setState(prev => ({
        ...prev,
        data,
        error: null
      }));
    }
  }, [mountedRef]);

  const setError = useCallback((error: string | null) => {
    if (mountedRef.current) {
      setState(prev => ({
        ...prev,
        error,
        isLoading: false
      }));
    }
  }, [mountedRef]);

  return {
    state,
    execute,
    reset,
    setData,
    setError
  };
}