import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { startOfMonth, endOfMonth, addMonths, format } from 'date-fns'
import { fetchAPI } from '../lib/api'
import { useDomainStore } from '../stores/useDomainStore'
import { useDomains } from './useDomains'
import type { CalendarHeatmapPoint } from '../lib/types'

export function useCalendarHeatmap(displayedMonth: Date, enabled: boolean) {
  const { selectedDomainId } = useDomainStore()
  const { data: domains } = useDomains()
  const selectedDomain = domains?.find((d) => d.id === selectedDomainId)

  const rangeStart = startOfMonth(displayedMonth)
  const rangeEnd = endOfMonth(addMonths(displayedMonth, 1))

  const params = new URLSearchParams()
  params.set('start', rangeStart.toISOString())
  params.set('end', rangeEnd.toISOString())
  if (selectedDomain) params.set('domain', selectedDomain.domain)

  const qs = params.toString()
  const monthKey = format(displayedMonth, 'yyyy-MM')

  return useQuery({
    queryKey: ['stats', 'calendar-heatmap', monthKey, selectedDomain?.domain ?? ''],
    queryFn: () => fetchAPI<CalendarHeatmapPoint[]>(`/api/stats/calendar-heatmap?${qs}`),
    enabled,
    staleTime: 5 * 60 * 1000,
    placeholderData: keepPreviousData,
  })
}
