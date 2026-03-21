import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import { fetchAPI } from '../lib/api'

export interface DomainSettings {
  [key: string]: string
}

export function useDomainSettings(domainId: string | null) {
  return useQuery({
    queryKey: ['domain-settings', domainId],
    queryFn: () => fetchAPI<DomainSettings>(`/api/settings/domain/${domainId}`),
    enabled: !!domainId,
    staleTime: 60_000,
  })
}

export function useUpdateDomainSettings(domainId: string | null) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (settings: Record<string, string>) =>
      fetchAPI(`/api/settings/domain/${domainId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(settings),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['domain-settings', domainId] })
      toast.success('Settings saved')
    },
    onError: (err) => toast.error('Failed to save settings', { description: err.message }),
  })
}
