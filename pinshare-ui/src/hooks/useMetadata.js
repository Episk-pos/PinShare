import { useQuery } from '@tanstack/react-query'
import { pinshareApi } from '../services/api'

export function useMetadata() {
  return useQuery({
    queryKey: ['metadata'],
    queryFn: () => pinshareApi.get('/files').then(res => res.data),  // API path
    staleTime: 30 * 1000, // 30 seconds - refetch more frequently
    refetchInterval: 30 * 1000, // Auto-refetch every 30 seconds
    retry: 3,
  })
}