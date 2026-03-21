import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { fetchAPI } from '@/lib/api'
import { useDateRangeStore } from '../stores/useDateRangeStore'
import { useDomainStore } from '../stores/useDomainStore'
import { useFraudSummary, useSourceQuality, useAdFraudCampaigns, useCreateCampaign, useDeleteCampaign, useAvailableEventNames } from '../hooks/useAnalyticsQueries'
import { useDomainSettings, useUpdateDomainSettings } from '../hooks/useDomainSettings'
import { FeatureGate } from '../components/FeatureGate'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { DateRangePicker } from '../components/ui/date-range-picker'
import { Skeleton } from '../components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../components/ui/select'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '../components/ui/chart'
import { ShieldAlert, DollarSign, TrendingDown, Plus, Trash2, Download, Target, Crosshair, Server } from 'lucide-react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Cell } from 'recharts'
import { toast } from 'sonner'
import { formatNumber } from '@/lib/utils'
import type { FraudSignal } from '@/lib/types'

const qualityChartConfig = {
  quality_score: { label: 'Quality Score', color: 'var(--chart-2)' },
} satisfies ChartConfig

function formatCurrency(num: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(num)
}

function getQualityColor(score: number): string {
  if (score >= 80) return 'hsl(142, 71%, 45%)'
  if (score >= 60) return 'hsl(84, 81%, 44%)'
  if (score >= 40) return 'hsl(38, 92%, 50%)'
  if (score >= 20) return 'hsl(25, 95%, 53%)'
  return 'hsl(0, 84%, 60%)'
}

function getSeverityColor(severity: FraudSignal['severity']): string {
  switch (severity) {
    case 'high': return 'text-red-500 bg-red-500/10'
    case 'medium': return 'text-yellow-500 bg-yellow-500/10'
    case 'low': return 'text-blue-500 bg-blue-500/10'
  }
}

function StatCardSkeleton() {
  return (
    <Card>
      <CardContent className="p-6">
        <Skeleton className="h-4 w-24 mb-2" />
        <Skeleton className="h-8 w-16" />
      </CardContent>
    </Card>
  )
}

function AdFraudContent() {
  const queryClient = useQueryClient()
  const { dateRange, setDateRange } = useDateRangeStore()
  const { selectedDomainId } = useDomainStore()
  const { data: fraudData, isLoading: fraudLoading, isPlaceholderData: fraudStale } = useFraudSummary()
  const { data: sourceQuality, isLoading: qualityLoading, isPlaceholderData: qualityStale } = useSourceQuality()
  const { data: campaigns, isLoading: campaignsLoading } = useAdFraudCampaigns()
  const { data: eventNames } = useAvailableEventNames()
  const { data: domainSettings } = useDomainSettings(selectedDomainId)
  const updateSettings = useUpdateDomainSettings(selectedDomainId)
  const createCampaign = useCreateCampaign()
  const deleteCampaign = useDeleteCampaign()

  const [showAddCampaign, setShowAddCampaign] = useState(false)
  const [newCampaign, setNewCampaign] = useState({ name: '', cpc: '', cpm: '', budget: '' })

  const isLoading = fraudLoading || qualityLoading
  const isStale = fraudStale || qualityStale
  const currentConversionEvent = domainSettings?.conversion_event || ''

  const handleConversionEventChange = (value: string) => {
    const eventValue = value === '__none__' ? '' : value
    updateSettings.mutate(
      { conversion_event: eventValue },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({ queryKey: ['stats', 'fraud'] })
        },
      }
    )
  }

  const handleAddCampaign = () => {
    createCampaign.mutate(
      {
        name: newCampaign.name,
        cpc: parseFloat(newCampaign.cpc) || 0,
        cpm: parseFloat(newCampaign.cpm) || 0,
        budget: parseFloat(newCampaign.budget) || 0,
      },
      {
        onSuccess: () => {
          setShowAddCampaign(false)
          setNewCampaign({ name: '', cpc: '', cpm: '', budget: '' })
          toast.success('Campaign created')
        },
        onError: (err) => {
          toast.error('Failed to create campaign', { description: err.message })
        },
      }
    )
  }

  const handleDeleteCampaign = (id: string) => {
    deleteCampaign.mutate(id, {
      onSuccess: () => toast.success('Campaign deleted'),
      onError: (err) => toast.error('Failed to delete campaign', { description: err.message }),
    })
  }

  const chartData = (sourceQuality || []).slice(0, 10).map(s => ({
    ...s,
    fill: getQualityColor(s.quality_score),
  }))

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full" style={{ opacity: isStale ? 0.6 : 1, transition: 'opacity 150ms' }}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <ShieldAlert className="h-7 w-7" />
            Ad Fraud Detection
          </h1>
          <p className="text-muted-foreground">Monitor click fraud and protect your ad spend</p>
        </div>
        <DateRangePicker dateRange={dateRange} onDateRangeChange={setDateRange} />
      </div>

      {/* Fraud Summary Cards — 6 cards */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <StatCardSkeleton /><StatCardSkeleton /><StatCardSkeleton />
          <StatCardSkeleton /><StatCardSkeleton /><StatCardSkeleton />
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <Card className="transition-all hover:shadow-md hover:border-primary/20">
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Total Clicks</p>
                  <p className="text-2xl font-bold mt-1">{formatNumber(fraudData?.total_clicks || 0)}</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {formatNumber(fraudData?.human_clicks || 0)} human
                  </p>
                </div>
                <div className="h-10 w-10 rounded-xl bg-muted flex items-center justify-center">
                  <TrendingDown className="h-5 w-5 text-muted-foreground" />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="transition-all hover:shadow-md hover:border-primary/20">
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Invalid Traffic</p>
                  <p className="text-2xl font-bold mt-1 text-red-500">
                    {((fraudData?.invalid_rate || 0) * 100).toFixed(1)}%
                  </p>
                </div>
                <div className="h-10 w-10 rounded-xl bg-red-500/10 flex items-center justify-center">
                  <ShieldAlert className="h-5 w-5 text-red-500" />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="transition-all hover:shadow-md hover:border-primary/20">
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Wasted Spend</p>
                  <p className="text-2xl font-bold mt-1 text-orange-500">
                    {formatCurrency(fraudData?.wasted_spend || 0)}
                  </p>
                </div>
                <div className="h-10 w-10 rounded-xl bg-orange-500/10 flex items-center justify-center">
                  <DollarSign className="h-5 w-5 text-orange-500" />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="transition-all hover:shadow-md hover:border-primary/20">
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Real CPC</p>
                  <p className="text-2xl font-bold mt-1">
                    {fraudData?.real_cpc != null ? formatCurrency(fraudData.real_cpc) : (
                      <span className="text-muted-foreground text-lg">N/A</span>
                    )}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">spend / human clicks</p>
                </div>
                <div className="h-10 w-10 rounded-xl bg-emerald-500/10 flex items-center justify-center">
                  <Crosshair className="h-5 w-5 text-emerald-500" />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="transition-all hover:shadow-md hover:border-primary/20">
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Real CPA</p>
                  <p className="text-2xl font-bold mt-1">
                    {fraudData?.real_cpa != null ? formatCurrency(fraudData.real_cpa) : (
                      <span className="text-muted-foreground text-lg">
                        {currentConversionEvent ? 'N/A' : 'Set event'}
                      </span>
                    )}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {fraudData?.conversions != null && fraudData.conversions > 0
                      ? `${formatNumber(fraudData.conversions)} conversions`
                      : 'spend / conversions'}
                  </p>
                </div>
                <div className="h-10 w-10 rounded-xl bg-violet-500/10 flex items-center justify-center">
                  <Target className="h-5 w-5 text-violet-500" />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="transition-all hover:shadow-md hover:border-primary/20">
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Datacenter Traffic</p>
                  <p className="text-2xl font-bold mt-1">{formatNumber(fraudData?.datacenter_clicks || 0)}</p>
                </div>
                <div className="h-10 w-10 rounded-xl bg-yellow-500/10 flex items-center justify-center">
                  <Server className="h-5 w-5 text-yellow-500" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Conversion Event Selector */}
      {selectedDomainId && (
        <div className="flex items-center gap-4">
          <label className="text-sm font-medium text-muted-foreground whitespace-nowrap">
            Conversion Event
          </label>
          <Select
            value={currentConversionEvent || '__none__'}
            onValueChange={handleConversionEventChange}
            disabled={updateSettings.isPending}
          >
            <SelectTrigger className="w-[240px]">
              <SelectValue placeholder="Select an event..." />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">None</SelectItem>
              {(eventNames || []).map((name) => (
                <SelectItem key={name} value={name}>{name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          {currentConversionEvent && (
            <span className="text-xs text-muted-foreground">
              Real CPA = total spend / human &ldquo;{currentConversionEvent}&rdquo; events
            </span>
          )}
        </div>
      )}

      {/* Fraud Signals */}
      {fraudData?.signals && fraudData.signals.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg font-semibold">Detected Fraud Signals</CardTitle>
            <CardDescription>Suspicious patterns found in your campaign traffic</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {fraudData.signals.map((signal, i) => (
                <div key={i} className="flex items-center justify-between p-3 rounded-lg border border-border">
                  <div className="flex items-center gap-3">
                    <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-semibold ${getSeverityColor(signal.severity)}`}>
                      {signal.severity}
                    </span>
                    <span className="text-sm">{signal.description}</span>
                  </div>
                  <span className="text-sm font-semibold tabular-nums">{formatNumber(signal.count)}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Source Quality */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg font-semibold">Traffic Source Quality</CardTitle>
          <CardDescription>Quality scores for your traffic sources (higher is better)</CardDescription>
        </CardHeader>
        <CardContent>
          {qualityLoading ? (
            <Skeleton className="h-[250px] w-full" />
          ) : (sourceQuality || []).length > 0 ? (
            <div className="space-y-6">
              <ChartContainer config={qualityChartConfig} className="h-[250px] w-full">
                <BarChart data={chartData} layout="vertical" margin={{ left: 80 }}>
                  <CartesianGrid strokeDasharray="3 3" className="stroke-muted" horizontal={true} vertical={false} />
                  <XAxis type="number" domain={[0, 100]} tick={{ fontSize: 12 }} axisLine={false} tickLine={false} />
                  <YAxis type="category" dataKey="source" width={80} tick={{ fontSize: 12 }} axisLine={false} tickLine={false} />
                  <ChartTooltip cursor={{ fill: 'hsl(var(--muted))' }} content={<ChartTooltipContent indicator="line" />} />
                  <Bar dataKey="quality_score" radius={[0, 4, 4, 0]}>
                    {chartData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.fill} />
                    ))}
                  </Bar>
                </BarChart>
              </ChartContainer>

              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-border">
                      <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">Source</th>
                      <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">Medium</th>
                      <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">Clicks</th>
                      <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">Invalid</th>
                      <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">Human Rate</th>
                      <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">Quality</th>
                    </tr>
                  </thead>
                  <tbody>
                    {sourceQuality!.map((source, i) => (
                      <tr key={i} className="border-b border-border last:border-0 hover:bg-muted/50 transition-colors">
                        <td className="py-3 px-4 font-medium">{source.source || 'Direct'}</td>
                        <td className="py-3 px-4 text-muted-foreground">{source.medium || '-'}</td>
                        <td className="text-right py-3 px-4 tabular-nums">{formatNumber(source.clicks)}</td>
                        <td className="text-right py-3 px-4 tabular-nums text-red-500">{formatNumber(source.invalid_clicks)}</td>
                        <td className="text-right py-3 px-4 tabular-nums">{((source.human_rate || 0) * 100).toFixed(1)}%</td>
                        <td className="text-right py-3 px-4">
                          <span
                            className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold"
                            style={{
                              backgroundColor: `${getQualityColor(source.quality_score ?? 0)}20`,
                              color: getQualityColor(source.quality_score ?? 0),
                            }}
                          >
                            {source.quality_score ?? 0}%
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground text-center py-8">No traffic source data available yet</p>
          )}
        </CardContent>
      </Card>

      {/* Campaign Manager */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-lg font-semibold">Campaign Manager</CardTitle>
              <CardDescription>Track ad spend and calculate wasted budget</CardDescription>
            </div>
            <Button onClick={() => setShowAddCampaign(true)} size="sm">
              <Plus className="h-4 w-4 mr-2" />
              Add Campaign
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {showAddCampaign && (
            <div className="mb-6 p-4 border border-border rounded-lg space-y-4 bg-muted/30">
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <Input
                  placeholder="Campaign Name"
                  value={newCampaign.name}
                  onChange={(e) => setNewCampaign({ ...newCampaign, name: e.target.value })}
                />
                <Input
                  type="number"
                  placeholder="CPC ($)"
                  value={newCampaign.cpc}
                  onChange={(e) => setNewCampaign({ ...newCampaign, cpc: e.target.value })}
                />
                <Input
                  type="number"
                  placeholder="CPM ($)"
                  value={newCampaign.cpm}
                  onChange={(e) => setNewCampaign({ ...newCampaign, cpm: e.target.value })}
                />
                <Input
                  type="number"
                  placeholder="Budget ($)"
                  value={newCampaign.budget}
                  onChange={(e) => setNewCampaign({ ...newCampaign, budget: e.target.value })}
                />
              </div>
              <div className="flex gap-2">
                <Button onClick={handleAddCampaign} disabled={!newCampaign.name || createCampaign.isPending}>
                  Save Campaign
                </Button>
                <Button variant="outline" onClick={() => setShowAddCampaign(false)}>
                  Cancel
                </Button>
              </div>
            </div>
          )}

          {campaignsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : (campaigns || []).length > 0 ? (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border">
                    <th className="text-left py-3 px-4 text-sm font-medium text-muted-foreground">Campaign</th>
                    <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">CPC</th>
                    <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">CPM</th>
                    <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">Budget</th>
                    <th className="text-right py-3 px-4 text-sm font-medium text-muted-foreground">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {campaigns!.map((campaign) => (
                    <tr key={campaign.id} className="border-b border-border last:border-0 hover:bg-muted/50 transition-colors">
                      <td className="py-3 px-4 font-medium">{campaign.name}</td>
                      <td className="text-right py-3 px-4 tabular-nums">{formatCurrency(campaign.cpc)}</td>
                      <td className="text-right py-3 px-4 tabular-nums">{formatCurrency(campaign.cpm)}</td>
                      <td className="text-right py-3 px-4 tabular-nums">{formatCurrency(campaign.budget)}</td>
                      <td className="text-right py-3 px-4">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={async () => {
                              try {
                                const data = await fetchAPI(`/api/campaigns/${campaign.id}/report`)
                                const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
                                const url = URL.createObjectURL(blob)
                                const a = document.createElement('a')
                                a.href = url
                                a.download = `campaign-${campaign.name || campaign.id}-report.json`
                                a.click()
                                URL.revokeObjectURL(url)
                              } catch {
                                toast.error('Failed to download report')
                              }
                            }}
                          >
                            <Download className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDeleteCampaign(campaign.id)}
                            className="text-destructive hover:text-destructive"
                            disabled={deleteCampaign.isPending}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground text-center py-8">
              No campaigns created yet. Add a campaign to track ad spend.
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export function AdFraud() {
  return (
    <FeatureGate feature="ad_fraud">
      <AdFraudContent />
    </FeatureGate>
  )
}
