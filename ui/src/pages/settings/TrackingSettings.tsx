import { useState, useEffect, useMemo } from 'react'
import { useAuth } from '@/hooks/useAuth'
import { fetchAPI } from '@/lib/api'
import { Navigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Activity, Database, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { useLicense } from '@/hooks/useLicenseQuery'
import { SettingsLayout } from './SettingsLayout'

export function TrackingSettings() {
  const { isAdmin } = useAuth()
  const { license } = useLicense()
  const [settings, setSettings] = useState<Record<string, string> | null>(null)
  const [edited, setEdited] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)

  const hasChanges = useMemo(() => Object.keys(edited).length > 0, [edited])

  useEffect(() => {
    if (isAdmin) {
      fetchAPI<Record<string, string>>('/api/settings')
        .then(setSettings)
        .catch(() => toast.error('Failed to load settings'))
    }
  }, [isAdmin])

  if (!isAdmin) {
    return <Navigate to="/settings/domains" replace />
  }

  function getBool(key: string, fallback: boolean): boolean {
    const val = edited[key] ?? settings?.[key]
    if (val === undefined) return fallback
    return val === 'true' || val === '1'
  }

  function getNumber(key: string, fallback: number): string {
    return edited[key] ?? settings?.[key] ?? String(fallback)
  }

  function setBool(key: string, value: boolean) {
    setEdited(prev => ({ ...prev, [key]: value ? 'true' : 'false' }))
  }

  function setNumber(key: string, value: string) {
    setEdited(prev => ({ ...prev, [key]: value }))
  }

  async function handleSave() {
    if (!hasChanges) return
    setSaving(true)
    try {
      await fetchAPI('/api/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(edited),
      })
      setSettings(prev => prev ? { ...prev, ...edited } : edited)
      setEdited({})
      toast.success('Tracking settings saved')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to save settings')
    } finally {
      setSaving(false)
    }
  }

  return (
    <SettingsLayout title="Tracking" description="Configure data collection and privacy settings">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5" />
            Data Collection
          </CardTitle>
          <CardDescription>
            Control what data the tracker collects from visitors.
            Changes take effect within 24 hours (tracker script is cached by browsers).
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Track Performance */}
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Core Web Vitals</Label>
              <p className="text-xs text-muted-foreground">
                Collect LCP, FCP, CLS, INP, TTFB and page load times
              </p>
            </div>
            <Switch
              checked={getBool('track_performance', true)}
              onCheckedChange={v => setBool('track_performance', v)}
            />
          </div>

          {/* Track Errors */}
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>JavaScript Errors</Label>
              <p className="text-xs text-muted-foreground">
                Capture JavaScript errors from tracked pages
              </p>
            </div>
            <Switch
              checked={getBool('track_errors', true)}
              onCheckedChange={v => setBool('track_errors', v)}
            />
          </div>

          {/* Respect DNT */}
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Honor Do Not Track</Label>
              <p className="text-xs text-muted-foreground">
                Respect the DNT browser header. Many browsers enable this by default,
                which may cause significant data loss. Not legally required.
              </p>
            </div>
            <Switch
              checked={getBool('respect_dnt', true)}
              onCheckedChange={v => setBool('respect_dnt', v)}
            />
          </div>

          {/* Session Timeout */}
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label htmlFor="session_timeout">Session Timeout (minutes)</Label>
              <p className="text-xs text-muted-foreground">
                Minutes of inactivity before a visitor session expires (default: 30)
              </p>
            </div>
            <Input
              id="session_timeout"
              type="number"
              min={1}
              max={1440}
              className="w-24"
              value={getNumber('session_timeout_minutes', 30)}
              onChange={e => setNumber('session_timeout_minutes', e.target.value)}
            />
          </div>

          {/* Save */}
          <div className="flex justify-end pt-4 border-t">
            <Button onClick={handleSave} disabled={saving || !hasChanges}>
              {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Save Settings
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            Data Retention
          </CardTitle>
          <CardDescription>
            Configure how long analytics data is kept before automatic deletion.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {(() => {
            const isPro = license.tier === 'pro' || license.tier === 'enterprise'
            const currentValue = getNumber('data_retention_days', 180)

            return (
              <>
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>Delete data older than</Label>
                    <p className="text-xs text-muted-foreground">
                      {isPro
                        ? 'Choose how long to keep analytics data'
                        : 'Community plan: up to 180 days'}
                    </p>
                  </div>
                  <Select
                    value={currentValue}
                    onValueChange={(value) => setNumber('data_retention_days', value)}
                  >
                    <SelectTrigger className="w-[160px]">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="90">90 days</SelectItem>
                      <SelectItem value="180">180 days</SelectItem>
                      {isPro && <SelectItem value="365">1 year</SelectItem>}
                      {isPro && <SelectItem value="-1">Forever</SelectItem>}
                    </SelectContent>
                  </Select>
                </div>

                <div className="flex justify-end pt-4 border-t">
                  <Button onClick={handleSave} disabled={saving || !hasChanges}>
                    {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    Save Settings
                  </Button>
                </div>
              </>
            )
          })()}
        </CardContent>
      </Card>
    </SettingsLayout>
  )
}
