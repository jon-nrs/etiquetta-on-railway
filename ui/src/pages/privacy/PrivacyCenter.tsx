import { useState } from 'react'
import { usePrivacyAudit, useVisitorLookup, useEraseVisitor, useExportVisitorData, useAuditLog } from '../../hooks/usePrivacy'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../components/ui/card'
import { Badge } from '../../components/ui/badge'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Skeleton } from '../../components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/tabs'
import { Separator } from '../../components/ui/separator'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../../components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../../components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../../components/ui/dropdown-menu'
import {
  Shield,
  ShieldCheck,
  AlertTriangle,
  Info,
  Search,
  Trash2,
  Cookie,
  Database,
  FileText,
  CheckCircle2,
  Globe,
  Download,
  ChevronLeft,
  ChevronRight,
  ScrollText,
} from 'lucide-react'
import type { PrivacyAuditCheck } from '../../lib/types'

function statusIcon(status: PrivacyAuditCheck['status']) {
  switch (status) {
    case 'pass':
      return <CheckCircle2 className="h-5 w-5 text-green-500 shrink-0" />
    case 'warn':
      return <AlertTriangle className="h-5 w-5 text-amber-500 shrink-0" />
    case 'info':
      return <Info className="h-5 w-5 text-blue-500 shrink-0" />
  }
}

function statusBadge(status: PrivacyAuditCheck['status']) {
  switch (status) {
    case 'pass':
      return <Badge variant="outline" className="bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20">Pass</Badge>
    case 'warn':
      return <Badge variant="outline" className="bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20">Warning</Badge>
    case 'info':
      return <Badge variant="outline" className="bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20">Info</Badge>
  }
}

const ACTION_STYLES: Record<string, string> = {
  create: 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20',
  upload: 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20',
  update: 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20',
  delete: 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20',
  erase: 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20',
  remove: 'bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20',
  export: 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20',
}

function ComplianceAuditTab() {
  const { data: audit, isLoading } = usePrivacyAudit()

  if (isLoading) {
    return (
      <div className="space-y-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-24 w-full" />
        ))}
      </div>
    )
  }

  if (!audit) return null

  const passCount = audit.checks.filter(c => c.status === 'pass').length
  const warnCount = audit.checks.filter(c => c.status === 'warn').length
  const categories = [...new Set(audit.checks.map(c => c.category))]

  return (
    <div className="space-y-6">
      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <ShieldCheck className="h-8 w-8 text-green-500" />
              <div>
                <p className="text-2xl font-bold">{passCount}/{audit.checks.length}</p>
                <p className="text-sm text-muted-foreground">Checks passing</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <Cookie className="h-8 w-8 text-muted-foreground" />
              <div>
                <p className="text-2xl font-bold">{audit.cookie_inventory.filter(c => c.visitor_facing).length}</p>
                <p className="text-sm text-muted-foreground">Visitor-facing cookies</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <Database className="h-8 w-8 text-muted-foreground" />
              <div>
                <p className="text-2xl font-bold">
                  {Object.values(audit.storage_summary).reduce((a, b) => a + b, 0).toLocaleString()}
                </p>
                <p className="text-sm text-muted-foreground">Total records stored</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {warnCount > 0 && (
        <Card className="border-amber-500/30 bg-amber-500/5">
          <CardContent className="pt-6">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-amber-500" />
              <p className="text-sm font-medium">{warnCount} item{warnCount > 1 ? 's' : ''} need attention</p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Compliance checks by category */}
      {categories.map(cat => (
        <Card key={cat}>
          <CardHeader>
            <CardTitle className="text-lg">{cat} Compliance</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {audit.checks.filter(c => c.category === cat).map(check => (
              <div key={check.id} className="flex items-start gap-3 p-3 rounded-lg bg-muted/50">
                {statusIcon(check.status)}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-medium text-sm">{check.name}</span>
                    {statusBadge(check.status)}
                  </div>
                  <p className="text-xs text-muted-foreground">{check.description}</p>
                  <p className="text-xs text-muted-foreground mt-1 italic">{check.detail}</p>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      ))}

      {/* Cookie inventory */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <Cookie className="h-5 w-5" />
            Cookie Inventory
          </CardTitle>
          <CardDescription>Complete list of cookies that may be set</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {audit.cookie_inventory.map(cookie => (
              <div key={cookie.name} className="p-3 rounded-lg bg-muted/50">
                <div className="flex items-center gap-2 mb-1">
                  <code className="text-sm font-mono font-medium">{cookie.name}</code>
                  <Badge variant="outline" className="text-xs">
                    {cookie.type.replace('_', ' ')}
                  </Badge>
                  {cookie.visitor_facing ? (
                    <Badge variant="outline" className="text-xs bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20">
                      visitor-facing
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-xs bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20">
                      admin only
                    </Badge>
                  )}
                </div>
                <p className="text-xs text-muted-foreground">{cookie.description}</p>
                <p className="text-xs text-muted-foreground mt-1">
                  Set by: {cookie.set_by} &middot; Duration: {cookie.duration}
                </p>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Data inventory */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <FileText className="h-5 w-5" />
            Data Inventory
          </CardTitle>
          <CardDescription>What data is collected and why</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b">
                  <th className="text-left py-2 pr-4 font-medium">Field</th>
                  <th className="text-left py-2 pr-4 font-medium">Purpose</th>
                  <th className="text-left py-2 pr-4 font-medium">PII</th>
                  <th className="text-left py-2 font-medium">Note</th>
                </tr>
              </thead>
              <tbody>
                {audit.data_inventory.map(item => (
                  <tr key={item.field} className="border-b last:border-0">
                    <td className="py-2 pr-4 font-mono text-xs">{item.field}</td>
                    <td className="py-2 pr-4 text-muted-foreground">{item.purpose}</td>
                    <td className="py-2 pr-4">
                      <Badge variant="outline" className="text-xs bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20">
                        No
                      </Badge>
                    </td>
                    <td className="py-2 text-xs text-muted-foreground">{item.note}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* Domain consent status */}
      {audit.domain_consents.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Globe className="h-5 w-5" />
              Consent Banner Status
            </CardTitle>
            <CardDescription>Consent configuration per tracked domain</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {audit.domain_consents.map(dc => (
                <div key={dc.domain_id} className="flex items-center justify-between p-3 rounded-lg bg-muted/50">
                  <div>
                    <span className="font-medium text-sm">{dc.domain_name}</span>
                    <span className="text-xs text-muted-foreground ml-2">{dc.domain}</span>
                  </div>
                  {dc.has_consent ? (
                    <Badge variant="outline" className="bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20">
                      v{dc.version} active
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20">
                      Not configured
                    </Badge>
                  )}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Storage breakdown */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <Database className="h-5 w-5" />
            Storage Breakdown
          </CardTitle>
          <CardDescription>
            Records per table
            {audit.data_retention_days > 0
              ? ` (auto-deleted after ${audit.data_retention_days} days)`
              : ' (unlimited retention)'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            {Object.entries(audit.storage_summary).map(([table, count]) => (
              <div key={table} className="text-center p-3 rounded-lg bg-muted/50">
                <p className="text-lg font-bold">{count.toLocaleString()}</p>
                <p className="text-xs text-muted-foreground">{table}</p>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <p className="text-xs text-muted-foreground text-right">
        Audit generated at {new Date(audit.generated_at).toLocaleString()}
      </p>
    </div>
  )
}

function ErasureTab() {
  const [hash, setHash] = useState('')
  const [searchHash, setSearchHash] = useState('')
  const [confirmOpen, setConfirmOpen] = useState(false)
  const { data: lookup, isLoading: lookupLoading, isFetching } = useVisitorLookup(searchHash)
  const eraseMutation = useEraseVisitor()
  const { exportData, isExporting } = useExportVisitorData()

  const handleSearch = () => {
    if (hash.length >= 16) {
      setSearchHash(hash)
    }
  }

  const handleErase = () => {
    eraseMutation.mutate(searchHash, {
      onSuccess: () => {
        setConfirmOpen(false)
        setSearchHash(hash)
      },
    })
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Right to Erasure (GDPR Art. 17)</CardTitle>
          <CardDescription>
            Look up, export, and delete all data associated with a visitor hash.
            The visitor hash is a server-generated identifier — it is not reversible to an IP address or person.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Input
              placeholder="Enter visitor hash (min 16 characters)..."
              value={hash}
              onChange={(e) => setHash(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              className="font-mono"
            />
            <Button onClick={handleSearch} disabled={hash.length < 16 || isFetching}>
              <Search className="h-4 w-4 mr-2" />
              Lookup
            </Button>
          </div>

          <p className="text-xs text-muted-foreground">
            Visitor hashes can be found in the Data Explorer, event exports, or consent records.
          </p>
        </CardContent>
      </Card>

      {lookupLoading && searchHash && (
        <Skeleton className="h-32 w-full" />
      )}

      {lookup && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Visitor Data Found</CardTitle>
            <CardDescription>
              <code className="font-mono text-xs">{lookup.visitor_hash}</code>
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {lookup.total_records === 0 ? (
              <div className="flex items-center gap-2 p-4 rounded-lg bg-muted/50">
                <CheckCircle2 className="h-5 w-5 text-green-500" />
                <p className="text-sm">No data found for this visitor hash. It may have already been erased.</p>
              </div>
            ) : (
              <>
                <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                  {Object.entries(lookup.tables).map(([table, count]) => (
                    <div key={table} className="text-center p-3 rounded-lg bg-muted/50">
                      <p className="text-lg font-bold">{count.toLocaleString()}</p>
                      <p className="text-xs text-muted-foreground">{table}</p>
                    </div>
                  ))}
                </div>

                <Separator />

                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium">Total: {lookup.total_records.toLocaleString()} records</p>
                    <p className="text-xs text-muted-foreground">
                      Download a copy or permanently delete all data for this visitor.
                    </p>
                  </div>
                  <div className="flex gap-2">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="outline" disabled={isExporting}>
                          <Download className="h-4 w-4 mr-2" />
                          {isExporting ? 'Exporting...' : 'Download Data'}
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent>
                        <DropdownMenuItem onClick={() => exportData(searchHash, 'json')}>
                          Download as JSON
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => exportData(searchHash, 'csv')}>
                          Download as CSV
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                    <Button
                      variant="destructive"
                      onClick={() => setConfirmOpen(true)}
                      disabled={eraseMutation.isPending}
                    >
                      <Trash2 className="h-4 w-4 mr-2" />
                      Erase All Data
                    </Button>
                  </div>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      )}

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Data Erasure</DialogTitle>
            <DialogDescription>
              You are about to permanently delete <strong>{lookup?.total_records.toLocaleString()}</strong> records
              for visitor <code className="font-mono text-xs">{searchHash.slice(0, 12)}...</code>.
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleErase} disabled={eraseMutation.isPending}>
              {eraseMutation.isPending ? 'Erasing...' : 'Permanently Erase'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function AuditLogTab() {
  const [page, setPage] = useState(1)
  const [actionFilter, setActionFilter] = useState<string>('')
  const [resourceFilter, setResourceFilter] = useState<string>('')
  const perPage = 50

  const { data, isLoading } = useAuditLog(
    page,
    perPage,
    actionFilter || undefined,
    resourceFilter || undefined,
  )

  const totalPages = data ? Math.ceil(data.total / perPage) : 0

  if (isLoading) {
    return (
      <div className="space-y-4">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <ScrollText className="h-5 w-5" />
            Admin Audit Log
          </CardTitle>
          <CardDescription>
            Track who accessed, modified, or deleted data and when.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Select value={actionFilter} onValueChange={(v) => { setActionFilter(v === 'all' ? '' : v); setPage(1) }}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="All actions" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All actions</SelectItem>
                <SelectItem value="create">Create</SelectItem>
                <SelectItem value="update">Update</SelectItem>
                <SelectItem value="delete">Delete</SelectItem>
                <SelectItem value="erase">Erase</SelectItem>
                <SelectItem value="export">Export</SelectItem>
                <SelectItem value="upload">Upload</SelectItem>
                <SelectItem value="remove">Remove</SelectItem>
              </SelectContent>
            </Select>
            <Select value={resourceFilter} onValueChange={(v) => { setResourceFilter(v === 'all' ? '' : v); setPage(1) }}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="All resources" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All resources</SelectItem>
                <SelectItem value="user">User</SelectItem>
                <SelectItem value="domain">Domain</SelectItem>
                <SelectItem value="settings">Settings</SelectItem>
                <SelectItem value="license">License</SelectItem>
                <SelectItem value="visitor_data">Visitor data</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {data && data.entries.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <ScrollText className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p className="text-sm">No audit log entries found.</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b">
                    <th className="text-left py-2 pr-4 font-medium">Time</th>
                    <th className="text-left py-2 pr-4 font-medium">User</th>
                    <th className="text-left py-2 pr-4 font-medium">Action</th>
                    <th className="text-left py-2 pr-4 font-medium">Resource</th>
                    <th className="text-left py-2 font-medium">Detail</th>
                  </tr>
                </thead>
                <tbody>
                  {data?.entries.map(entry => (
                    <tr key={entry.id} className="border-b last:border-0">
                      <td className="py-2 pr-4 text-xs text-muted-foreground whitespace-nowrap">
                        {new Date(entry.timestamp).toLocaleString()}
                      </td>
                      <td className="py-2 pr-4 text-xs">{entry.user_email}</td>
                      <td className="py-2 pr-4">
                        <Badge variant="outline" className={`text-xs ${ACTION_STYLES[entry.action] ?? ''}`}>
                          {entry.action}
                        </Badge>
                      </td>
                      <td className="py-2 pr-4 text-xs">
                        {entry.resource_type}
                        {entry.resource_id && (
                          <code className="ml-1 text-muted-foreground font-mono">
                            {entry.resource_id.length > 12 ? entry.resource_id.slice(0, 12) + '...' : entry.resource_id}
                          </code>
                        )}
                      </td>
                      <td className="py-2 text-xs text-muted-foreground max-w-xs truncate">{entry.detail}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {totalPages > 1 && (
            <div className="flex items-center justify-between pt-2">
              <p className="text-xs text-muted-foreground">
                Page {page} of {totalPages} ({data?.total} entries)
              </p>
              <div className="flex gap-1">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page <= 1}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export function PrivacyCenter() {
  return (
    <div className="p-6 space-y-6 h-full overflow-y-auto">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <Shield className="h-6 w-6" />
          Privacy Center
        </h1>
        <p className="text-muted-foreground">
          GDPR &amp; ePrivacy compliance audit, cookie inventory, data access &amp; erasure, and admin audit log
        </p>
      </div>

      <Tabs defaultValue="audit">
        <TabsList>
          <TabsTrigger value="audit">Compliance Audit</TabsTrigger>
          <TabsTrigger value="erasure">Data Access &amp; Erasure</TabsTrigger>
          <TabsTrigger value="audit-log">Audit Log</TabsTrigger>
        </TabsList>
        <TabsContent value="audit" className="mt-4">
          <ComplianceAuditTab />
        </TabsContent>
        <TabsContent value="erasure" className="mt-4">
          <ErasureTab />
        </TabsContent>
        <TabsContent value="audit-log" className="mt-4">
          <AuditLogTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}
