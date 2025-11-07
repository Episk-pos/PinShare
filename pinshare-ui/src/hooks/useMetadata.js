import { useQuery } from '@tanstack/react-query'
import { pinshareApi } from '../services/api'

export function useMetadata() {
  return useQuery({
    queryKey: ['metadata'],
    queryFn: () => pinshareApi.get('/files').then(res => res.data),  // API path
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: 3,
  })
}