import { useQuery } from '@tanstack/react-query';
import { getApiStatusOptions } from '@/generated/api/@tanstack/react-query.gen';

export function useSystemStatus() {
  return useQuery(getApiStatusOptions());
}