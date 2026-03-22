import { useState } from 'react'
import { toast } from 'sonner'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '../ui/dialog'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Label } from '../ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select'
import { useCreateAnnotation, useUpdateAnnotation, useDeleteAnnotation } from '../../hooks/useAnnotations'
import { useDomainStore } from '../../stores/useDomainStore'
import { ANNOTATION_CATEGORIES } from '../../lib/types'
import type { Annotation, AnnotationCategory } from '../../lib/types'
import { Trash2 } from 'lucide-react'

interface AnnotationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  annotation?: Annotation | null
  defaultDate?: string
}

export function AnnotationDialog({ open, onOpenChange, annotation, defaultDate }: AnnotationDialogProps) {
  // Render the inner form only when open, so state resets on each open
  if (!open) return null
  return (
    <Dialog open onOpenChange={onOpenChange}>
      <AnnotationForm annotation={annotation} defaultDate={defaultDate} onClose={() => onOpenChange(false)} />
    </Dialog>
  )
}

function AnnotationForm({ annotation, defaultDate, onClose }: { annotation?: Annotation | null; defaultDate?: string; onClose: () => void }) {
  const { selectedDomainId } = useDomainStore()
  const createMutation = useCreateAnnotation()
  const updateMutation = useUpdateAnnotation()
  const deleteMutation = useDeleteAnnotation()

  const isEdit = !!annotation

  const [date, setDate] = useState(annotation?.date ?? defaultDate ?? new Date().toISOString().slice(0, 10))
  const [title, setTitle] = useState(annotation?.title ?? '')
  const [description, setDescription] = useState(annotation?.description ?? '')
  const [category, setCategory] = useState<AnnotationCategory>(annotation?.category ?? 'other')

  function handleSave() {
    if (!title.trim() || !date) {
      toast.error('Title and date are required')
      return
    }

    if (isEdit && annotation) {
      updateMutation.mutate(
        { id: annotation.id, date, title: title.trim(), description: description.trim(), category },
        {
          onSuccess: () => {
            toast.success('Annotation updated')
            onClose()
          },
          onError: () => toast.error('Failed to update annotation'),
        },
      )
    } else {
      if (!selectedDomainId) return
      createMutation.mutate(
        { domain_id: selectedDomainId, date, title: title.trim(), description: description.trim(), category },
        {
          onSuccess: () => {
            toast.success('Annotation added')
            onClose()
          },
          onError: () => toast.error('Failed to create annotation'),
        },
      )
    }
  }

  function handleDelete() {
    if (!annotation) return
    deleteMutation.mutate(annotation.id, {
      onSuccess: () => {
        toast.success('Annotation deleted')
        onClose()
      },
      onError: () => toast.error('Failed to delete annotation'),
    })
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <DialogContent className="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{isEdit ? 'Edit Annotation' : 'Add Annotation'}</DialogTitle>
      </DialogHeader>
      <div className="grid gap-4 py-2">
        <div className="grid gap-2">
          <Label htmlFor="ann-date">Date</Label>
          <Input id="ann-date" type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="ann-title">Title</Label>
          <Input id="ann-title" placeholder="e.g. Deployed v2.1" value={title} onChange={(e) => setTitle(e.target.value)} />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="ann-desc">Description (optional)</Label>
          <Input id="ann-desc" placeholder="More details..." value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        <div className="grid gap-2">
          <Label>Category</Label>
          <Select value={category} onValueChange={(v) => setCategory(v as AnnotationCategory)}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(Object.entries(ANNOTATION_CATEGORIES) as [AnnotationCategory, { label: string; color: string }][]).map(([key, { label, color }]) => (
                <SelectItem key={key} value={key}>
                  <span className="flex items-center gap-2">
                    <span className="h-2 w-2 rounded-full" style={{ backgroundColor: color }} />
                    {label}
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <DialogFooter className="flex items-center">
        {isEdit && (
          <Button variant="ghost" size="sm" className="mr-auto text-destructive hover:text-destructive" onClick={handleDelete} disabled={deleteMutation.isPending}>
            <Trash2 className="h-4 w-4 mr-1" />
            Delete
          </Button>
        )}
        <Button variant="outline" onClick={onClose}>Cancel</Button>
        <Button onClick={handleSave} disabled={isPending}>
          {isPending ? 'Saving...' : 'Save'}
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}
