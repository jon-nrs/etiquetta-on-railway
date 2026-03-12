import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { fetchAPI } from '../lib/api'
import type { PrivacyAudit, VisitorLookupResult, ErasureResult, AuditLogResponse } from '../lib/types'

export function usePrivacyAudit() {
  return useQuery({
    queryKey: ['privacy', 'audit'],
    queryFn: () => fetchAPI<PrivacyAudit>('/api/privacy/audit'),
    staleTime: 60_000,
  })
}

export function useVisitorLookup(visitorHash: string) {
  return useQuery({
    queryKey: ['privacy', 'lookup', visitorHash],
    queryFn: () => fetchAPI<VisitorLookupResult>(`/api/privacy/erasure/${visitorHash}`),
    enabled: visitorHash.length >= 16,
    staleTime: 30_000,
  })
}

export function useEraseVisitor() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (visitorHash: string) =>
      fetchAPI<ErasureResult>(`/api/privacy/erasure/${visitorHash}`, { method: 'DELETE' }),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['privacy'] })
      toast.success(`Erased ${data.total_deleted} records for visitor`)
    },
    onError: (err) => toast.error('Failed to erase visitor data', { description: err.message }),
  })
}

export function useExportVisitorData() {
  const [isExporting, setIsExporting] = useState(false)

  const exportData = async (visitorHash: string, format: 'json' | 'csv' = 'json') => {
    setIsExporting(true)
    try {
      const res = await fetch(`/api/privacy/export/${visitorHash}?format=${format}`, {
        credentials: 'include',
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: `HTTP ${res.status}` }))
        throw new Error((body as { error: string }).error || `HTTP ${res.status}`)
      }
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `visitor-${visitorHash.slice(0, 12)}-export.${format}`
      a.click()
      URL.revokeObjectURL(url)
      toast.success(`Downloaded visitor data as ${format.toUpperCase()}`)
    } catch (err) {
      toast.error('Failed to export visitor data', {
        description: err instanceof Error ? err.message : 'Unknown error',
      })
    } finally {
      setIsExporting(false)
    }
  }

  return { exportData, isExporting }
}

export function useAuditLog(page: number, perPage: number, action?: string, resourceType?: string) {
  const params = new URLSearchParams({ page: String(page), per_page: String(perPage) })
  if (action) params.set('action', action)
  if (resourceType) params.set('resource_type', resourceType)

  return useQuery({
    queryKey: ['privacy', 'audit-log', page, perPage, action, resourceType],
    queryFn: () => fetchAPI<AuditLogResponse>(`/api/privacy/audit-log?${params}`),
    staleTime: 30_000,
  })
}
