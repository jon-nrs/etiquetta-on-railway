import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { fetchAPI } from '../lib/api'
import { useDateRangeStore } from '../stores/useDateRangeStore'
import { useDomainStore } from '../stores/useDomainStore'
import type { Annotation, AnnotationCategory } from '../lib/types'

function useAnnotationParams(category?: AnnotationCategory) {
  const { dateRange } = useDateRangeStore()
  const { selectedDomainId } = useDomainStore()

  const params = new URLSearchParams()
  if (selectedDomainId) params.set('domain_id', selectedDomainId)
  if (dateRange?.from) params.set('start', dateRange.from.toISOString().slice(0, 10))
  if (dateRange?.to) params.set('end', dateRange.to.toISOString().slice(0, 10))
  if (category) params.set('category', category)

  return { qs: params.toString(), enabled: !!selectedDomainId }
}

export function useAnnotations(category?: AnnotationCategory) {
  const { qs, enabled } = useAnnotationParams(category)
  return useQuery({
    queryKey: ['annotations', qs],
    queryFn: () => fetchAPI<Annotation[]>(`/api/annotations?${qs}`),
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useCreateAnnotation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: { domain_id: string; date: string; title: string; description?: string; category?: AnnotationCategory }) =>
      fetchAPI<Annotation>('/api/annotations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['annotations'] }),
  })
}

export function useUpdateAnnotation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: { id: string; date?: string; title?: string; description?: string; category?: AnnotationCategory }) =>
      fetchAPI<void>(`/api/annotations/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['annotations'] }),
  })
}

export function useDeleteAnnotation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      fetchAPI<void>(`/api/annotations/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['annotations'] }),
  })
}
