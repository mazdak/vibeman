import React, { useCallback, useEffect, useRef, useState } from 'react';

/**
 * Hook to track component mount state
 * Useful for preventing state updates on unmounted components
 */
export function useMounted() {
  const mountedRef = useRef(true);

  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  return mountedRef;
}

/**
 * Hook that provides a safe setState function that only updates if component is mounted
 */
export function useSafeState<T>(
  initialState: T | (() => T)
): [T, (value: T | ((prev: T) => T)) => void] {
  const [state, setState] = useState(initialState);
  const mountedRef = useMounted();

  const safeSetState = useCallback((value: T | ((prev: T) => T)) => {
    if (mountedRef.current) {
      setState(value);
    }
  }, []);

  return [state, safeSetState];
}