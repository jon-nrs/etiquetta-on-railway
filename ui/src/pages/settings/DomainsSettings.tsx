import { useState } from 'react'
import { toast } from 'sonner'
import { useDomains, useCreateDomain, useDeleteDomain } from '@/hooks/useDomains'
import { fetchAPI } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Globe, Copy, Trash2, Plus, Check } from 'lucide-react'
import { SettingsLayout } from './SettingsLayout'

export function DomainsSettings() {
  const { data: domains, isLoading } = useDomains()
  const createDomain = useCreateDomain()
  const deleteDomain = useDeleteDomain()
  const [newDomain, setNewDomain] = useState({ name: '', domain: '' })
  const [copiedId, setCopiedId] = useState<string | null>(null)

  async function handleAddDomain(e: React.FormEvent) {
    e.preventDefault()
    if (!newDomain.name || !newDomain.domain) return
    createDomain.mutate(newDomain, {
      onSuccess: () => setNewDomain({ name: '', domain: '' }),
    })
  }

  async function handleDeleteDomain(id: string) {
    if (!confirm('Are you sure you want to delete this domain?')) return
    deleteDomain.mutate(id)
  }

  async function copySnippet(id: string) {
    try {
      const data = await fetchAPI<{ snippet: string }>(`/api/domains/${id}/snippet`)
      await navigator.clipboard.writeText(data.snippet)
      setCopiedId(id)
      toast.success('Snippet copied to clipboard')
      setTimeout(() => setCopiedId(null), 2000)
    } catch {
      toast.error('Failed to copy snippet')
    }
  }

  return (
    <SettingsLayout title="Properties" description="Manage your tracked properties">
      <Card>
        <CardHeader>
          <CardTitle>Add Property</CardTitle>
          <CardDescription>Register a domain to start tracking analytics</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleAddDomain} className="flex gap-4">
            <div className="flex-1">
              <Input
                placeholder="Site name (e.g., My Blog)"
                value={newDomain.name}
                onChange={(e) => setNewDomain({ ...newDomain, name: e.target.value })}
              />
            </div>
            <div className="flex-1">
              <Input
                placeholder="Domain (e.g., blog.example.com)"
                value={newDomain.domain}
                onChange={(e) => setNewDomain({ ...newDomain, domain: e.target.value })}
              />
            </div>
            <Button type="submit" disabled={createDomain.isPending}>
              <Plus className="h-4 w-4 mr-2" />
              Add
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Registered Properties</CardTitle>
          <CardDescription>Click the copy button to get the tracking snippet for each property</CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <p className="text-muted-foreground">Loading...</p>
          ) : !domains || domains.length === 0 ? (
            <p className="text-muted-foreground">No properties registered yet.</p>
          ) : (
            <div className="space-y-3">
              {domains.map((domain) => (
                <div key={domain.id} className="flex items-center justify-between p-4 rounded-lg border border-border">
                  <div className="flex items-center gap-3">
                    <Globe className="h-5 w-5 text-muted-foreground" />
                    <div>
                      <p className="font-medium">{domain.name}</p>
                      <p className="text-sm text-muted-foreground">{domain.domain}</p>
                      <p className="text-xs text-muted-foreground font-mono">{domain.site_id}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" onClick={() => copySnippet(domain.id)}>
                      {copiedId === domain.id ? (
                        <><Check className="h-4 w-4 mr-1" />Copied</>
                      ) : (
                        <><Copy className="h-4 w-4 mr-1" />Copy Snippet</>
                      )}
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => handleDeleteDomain(domain.id)}>
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Tracking Snippet</CardTitle>
          <CardDescription>
            Add this script to your website. Each domain has a unique <code className="text-xs bg-muted px-1 rounded">data-site</code> ID that ensures only your registered domains can send analytics data.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <pre className="p-4 bg-zinc-100 dark:bg-zinc-800 rounded-lg text-sm overflow-x-auto">
            <code>{`<!-- Etiquetta Analytics -->
<script defer data-site="YOUR_SITE_ID" src="${window.location.origin}/s.js"></script>`}</code>
          </pre>
          <p className="text-xs text-muted-foreground mt-3">
            Click "Copy Snippet" on a domain above to get the snippet with the correct site ID.
          </p>
        </CardContent>
      </Card>
    </SettingsLayout>
  )
}
