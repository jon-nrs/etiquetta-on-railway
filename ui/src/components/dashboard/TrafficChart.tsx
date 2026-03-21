import { useState } from 'react'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, ReferenceLine } from 'recharts'
import { Plus } from 'lucide-react'
import { useTimeseries } from '../../hooks/useAnalyticsQueries'
import { useAnnotations } from '../../hooks/useAnnotations'
import { ANNOTATION_CATEGORIES } from '../../lib/types'
import type { Annotation } from '../../lib/types'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  ChartLegend,
  ChartLegendContent,
  type ChartConfig,
} from '../ui/chart'
import { ChartSkeleton } from './skeletons'
import { AnnotationDialog } from './AnnotationDialog'

const chartConfig = {
  pageviews: { label: 'Pageviews', color: 'var(--chart-1)' },
  visitors: { label: 'Visitors', color: 'var(--chart-2)' },
} satisfies ChartConfig

export function TrafficChart() {
  const { data, isLoading, isPlaceholderData } = useTimeseries()
  const { data: annotations } = useAnnotations()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editAnnotation, setEditAnnotation] = useState<Annotation | null>(null)

  if (isLoading && !data) return <ChartSkeleton />

  const timeseries = data ?? []
  const periods = new Set(timeseries.map((p) => p.period))

  // Match annotations to timeseries periods
  const visibleAnnotations = (annotations ?? []).filter((a) => periods.has(a.date))

  function handleAnnotationClick(ann: Annotation) {
    setEditAnnotation(ann)
    setDialogOpen(true)
  }

  function handleAdd() {
    setEditAnnotation(null)
    setDialogOpen(true)
  }

  return (
    <>
      <Card className={`transition-opacity ${isPlaceholderData ? 'opacity-60' : ''}`}>
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-lg font-semibold">Traffic Overview</CardTitle>
              <CardDescription>Pageviews and unique visitors over time</CardDescription>
            </div>
            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={handleAdd} title="Add annotation">
              <Plus className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {timeseries.length > 0 ? (
            <ChartContainer config={chartConfig} className="h-[300px] w-full">
              <AreaChart data={timeseries} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
                <defs>
                  <linearGradient id="fillPageviews" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--color-pageviews)" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="var(--color-pageviews)" stopOpacity={0.05} />
                  </linearGradient>
                  <linearGradient id="fillVisitors" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--color-visitors)" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="var(--color-visitors)" stopOpacity={0.05} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" vertical={false} />
                <XAxis
                  dataKey="period"
                  tick={{ fontSize: 12 }}
                  tickFormatter={(v) => {
                    if (!v) return ''
                    const parts = v.split(' ')
                    return parts[0]?.slice(5) || v
                  }}
                  axisLine={false}
                  tickLine={false}
                  className="text-muted-foreground"
                />
                <YAxis
                  tick={{ fontSize: 12 }}
                  axisLine={false}
                  tickLine={false}
                  className="text-muted-foreground"
                  width={40}
                />
                <ChartTooltip content={<ChartTooltipContent indicator="dot" />} />
                <ChartLegend content={<ChartLegendContent />} />
                <Area
                  type="monotone"
                  dataKey="pageviews"
                  stroke="var(--color-pageviews)"
                  strokeWidth={2}
                  fill="url(#fillPageviews)"
                />
                <Area
                  type="monotone"
                  dataKey="visitors"
                  stroke="var(--color-visitors)"
                  strokeWidth={2}
                  fill="url(#fillVisitors)"
                />
                {visibleAnnotations.map((ann) => (
                  <ReferenceLine
                    key={ann.id}
                    x={ann.date}
                    stroke={ANNOTATION_CATEGORIES[ann.category]?.color ?? '#6b7280'}
                    strokeDasharray="4 4"
                    strokeWidth={1.5}
                    label={{
                      value: ann.title,
                      position: 'top',
                      fontSize: 10,
                      fill: ANNOTATION_CATEGORIES[ann.category]?.color ?? '#6b7280',
                      cursor: 'pointer',
                      onClick: () => handleAnnotationClick(ann),
                    }}
                  />
                ))}
              </AreaChart>
            </ChartContainer>
          ) : (
            <div className="h-[300px] flex items-center justify-center text-muted-foreground">
              No data for the selected period
            </div>
          )}
        </CardContent>
      </Card>
      <AnnotationDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        annotation={editAnnotation}
      />
    </>
  )
}
