import { useReplayStats } from '@/hooks/useReplayQueries'
import { useDomainSettings, useUpdateDomainSettings } from '@/hooks/useDomainSettings'
import { useDomainStore } from '@/stores/useDomainStore'
import { FeatureGate } from '@/components/FeatureGate'
import { toast } from 'sonner'
import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, HardDrive, Save, Info } from 'lucide-react'

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

function ScopeIndicator({ scope }: { scope: string | undefined }) {
  if (scope !== 'global') return null
  return (
    <span className="inline-flex items-center gap-1 text-xs text-muted-foreground ml-1">
      <Info className="h-3 w-3" />
      default
    </span>
  )
}

function ReplaySettingsForm() {
  const selectedDomainId = useDomainStore(s => s.selectedDomainId)
  const { data: settings } = useDomainSettings(selectedDomainId)
  const updateSettings = useUpdateDomainSettings(selectedDomainId)
  const { data: stats } = useReplayStats()
  const [edited, setEdited] = useState<Record<string, string>>({})

  const hasChanges = useMemo(() => Object.keys(edited).length > 0, [edited])

  function getBool(key: string, fallback: boolean): boolean {
    const val = edited[key] ?? settings?.[key]
    if (val === undefined) return fallback
    return val === 'true' || val === '1'
  }

  function getNumber(key: string, fallback: number): number {
    const val = edited[key] ?? settings?.[key]
    if (val === undefined) return fallback
    return Number(val) || fallback
  }

  function getScope(key: string): string | undefined {
    return settings?.['scope:' + key]
  }

  function setBool(key: string, value: boolean) {
    setEdited(prev => ({ ...prev, [key]: value ? 'true' : 'false' }))
  }

  function setNum(key: string, value: number) {
    setEdited(prev => ({ ...prev, [key]: String(value) }))
  }

  const handleSave = () => {
    if (!hasChanges) return
    updateSettings.mutate(edited, {
      onSuccess: () => {
        setEdited({})
        toast.success('Replay settings updated')
      },
    })
  }

  return (
    <div className="p-6 space-y-6 max-w-2xl mx-auto overflow-auto h-full">
      <div className="flex items-center gap-3">
        <Link to="/replays" className="p-1.5 rounded-md hover:bg-muted transition-colors">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold">Session Replay Settings</h1>
          <p className="text-muted-foreground">Configure recording behavior and privacy for this property</p>
        </div>
      </div>

      {stats && (
        <div className="rounded-lg border p-4 flex items-center gap-4">
          <HardDrive className="h-5 w-5 text-muted-foreground" />
          <div className="flex-1">
            <div className="text-sm font-medium">
              Storage: {formatBytes(stats.disk_usage_bytes)} / {formatBytes(stats.quota_bytes)}
              <span className="text-xs text-muted-foreground ml-2">(instance-wide)</span>
            </div>
            <div className="mt-1 h-2 bg-muted rounded-full overflow-hidden">
              <div
                className="h-full bg-primary rounded-full transition-all"
                style={{
                  width: `${Math.min(100, (stats.disk_usage_bytes / stats.quota_bytes) * 100)}%`,
                }}
              />
            </div>
          </div>
          <span className="text-sm text-muted-foreground">
            {stats.total_recordings} recordings
          </span>
        </div>
      )}

      <div className="space-y-6">
        {/* Enable/Disable */}
        <div className="flex items-center justify-between rounded-lg border p-4">
          <div>
            <div className="font-medium">
              Enable Session Replay
              <ScopeIndicator scope={getScope('replay_enabled')} />
            </div>
            <div className="text-sm text-muted-foreground">Record user sessions for playback</div>
          </div>
          <button
            onClick={() => setBool('replay_enabled', !getBool('replay_enabled', false))}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
              getBool('replay_enabled', false) ? 'bg-primary' : 'bg-muted'
            }`}
          >
            <span
              className={`inline-block h-4 w-4 rounded-full bg-white transition-transform ${
                getBool('replay_enabled', false) ? 'translate-x-6' : 'translate-x-1'
              }`}
            />
          </button>
        </div>

        {/* Sample Rate */}
        <div className="rounded-lg border p-4 space-y-2">
          <div className="flex items-center justify-between">
            <div>
              <div className="font-medium">
                Sample Rate
                <ScopeIndicator scope={getScope('replay_sample_rate')} />
              </div>
              <div className="text-sm text-muted-foreground">Percentage of sessions to record</div>
            </div>
            <span className="text-lg font-bold">{getNumber('replay_sample_rate', 10)}%</span>
          </div>
          <input
            type="range"
            min={0}
            max={100}
            value={getNumber('replay_sample_rate', 10)}
            onChange={e => setNum('replay_sample_rate', Number(e.target.value))}
            className="w-full"
          />
          <div className="flex justify-between text-xs text-muted-foreground">
            <span>0% (off)</span>
            <span>50%</span>
            <span>100% (all)</span>
          </div>
        </div>

        {/* Privacy */}
        <div className="rounded-lg border p-4 space-y-4">
          <div className="font-medium">Privacy</div>
          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm font-medium">
                Mask all text
                <ScopeIndicator scope={getScope('replay_mask_text')} />
              </div>
              <div className="text-xs text-muted-foreground">Replace visible text with asterisks</div>
            </div>
            <button
              onClick={() => setBool('replay_mask_text', !getBool('replay_mask_text', true))}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                getBool('replay_mask_text', true) ? 'bg-primary' : 'bg-muted'
              }`}
            >
              <span
                className={`inline-block h-4 w-4 rounded-full bg-white transition-transform ${
                  getBool('replay_mask_text', true) ? 'translate-x-6' : 'translate-x-1'
                }`}
              />
            </button>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm font-medium">
                Mask all inputs
                <ScopeIndicator scope={getScope('replay_mask_inputs')} />
              </div>
              <div className="text-xs text-muted-foreground">Hide form field values in recordings</div>
            </div>
            <button
              onClick={() => setBool('replay_mask_inputs', !getBool('replay_mask_inputs', true))}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                getBool('replay_mask_inputs', true) ? 'bg-primary' : 'bg-muted'
              }`}
            >
              <span
                className={`inline-block h-4 w-4 rounded-full bg-white transition-transform ${
                  getBool('replay_mask_inputs', true) ? 'translate-x-6' : 'translate-x-1'
                }`}
              />
            </button>
          </div>
          <p className="text-xs text-muted-foreground border-t pt-3">
            Add <code className="bg-muted px-1 rounded">class="etq-no-record"</code> to any element to exclude it from recordings entirely.
          </p>
        </div>

        {/* Max Duration (per-domain) */}
        <div className="rounded-lg border p-4 space-y-4">
          <div className="font-medium">Limits</div>
          <div>
            <label className="text-sm font-medium">
              Max recording duration
              <ScopeIndicator scope={getScope('replay_max_duration_sec')} />
            </label>
            <select
              value={getNumber('replay_max_duration_sec', 1800)}
              onChange={e => setNum('replay_max_duration_sec', Number(e.target.value))}
              className="mt-1 w-full rounded-md border bg-background px-3 py-2 text-sm"
            >
              <option value={300}>5 minutes</option>
              <option value={600}>10 minutes</option>
              <option value={1800}>30 minutes</option>
              <option value={3600}>1 hour</option>
              <option value={7200}>2 hours</option>
            </select>
          </div>
        </div>

        <button
          onClick={handleSave}
          disabled={updateSettings.isPending || !hasChanges}
          className="inline-flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
        >
          <Save className="h-4 w-4" />
          {updateSettings.isPending ? 'Saving...' : 'Save Settings'}
        </button>
      </div>
    </div>
  )
}

function ReplaySettingsContent() {
  const selectedDomainId = useDomainStore(s => s.selectedDomainId)

  if (!selectedDomainId) {
    return (
      <div className="p-6 max-w-2xl mx-auto">
        <div className="flex items-center gap-3 mb-6">
          <Link to="/replays" className="p-1.5 rounded-md hover:bg-muted transition-colors">
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold">Session Replay Settings</h1>
            <p className="text-muted-foreground">Configure recording behavior and privacy</p>
          </div>
        </div>
        <div className="rounded-lg border p-8 text-center text-muted-foreground">
          Select a property from the sidebar to configure replay settings.
        </div>
      </div>
    )
  }

  return <ReplaySettingsForm />
}

export function ReplaySettings() {
  return (
    <FeatureGate feature="session_replay">
      <ReplaySettingsContent />
    </FeatureGate>
  )
}
