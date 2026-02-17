import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toast } from 'sonner'
import { useFinanceStore } from '@/stores/finance'

const CATEGORIES = [
  'Software', 'Hardware', 'Hosting', 'Reise', 'Verpflegung',
  'Büro', 'Beratung', 'Bankgebuehren', 'Versicherung', 'Sonstiges',
]

interface ExpenseFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ExpenseFormDialog({ open, onOpenChange }: ExpenseFormDialogProps) {
  const { addExpense } = useFinanceStore()

  const [description, setDescription] = useState('')
  const [amount, setAmount] = useState(0)
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const [category, setCategory] = useState('Sonstiges')
  const [supplier, setSupplier] = useState('')
  const [project, setProject] = useState('')

  const reset = () => {
    setDescription('')
    setAmount(0)
    setDate(new Date().toISOString().split('T')[0])
    setCategory('Sonstiges')
    setSupplier('')
    setProject('')
  }

  const handleSave = () => {
    if (!description.trim() || amount <= 0) return
    addExpense({
      description: description.trim(),
      amount,
      date,
      category,
      supplier: supplier.trim(),
      project: project.trim() || undefined,
      receipt: false,
      status: 'pending',
    })
    toast.success('Ausgabe erfasst')
    reset()
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) reset(); onOpenChange(v) }}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Neue Ausgabe</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label>Beschreibung *</Label>
            <Input autoFocus placeholder="Was wurde gekauft?" value={description} onChange={(e) => setDescription(e.target.value)} />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Betrag (CHF) *</Label>
              <Input type="number" min={0.01} step={0.01} placeholder="0.00" value={amount || ''} onChange={(e) => setAmount(Number(e.target.value))} />
            </div>
            <div className="space-y-1.5">
              <Label>Datum</Label>
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Kategorie</Label>
              <Select value={category} onValueChange={setCategory}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {CATEGORIES.map((c) => <SelectItem key={c} value={c}>{c}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Lieferant</Label>
              <Input placeholder="Firma / Shop" value={supplier} onChange={(e) => setSupplier(e.target.value)} />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label>Projekt (optional)</Label>
            <Input placeholder="Zuordnung zu Projekt" value={project} onChange={(e) => setProject(e.target.value)} />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => { reset(); onOpenChange(false) }}>Abbrechen</Button>
          <Button onClick={handleSave} disabled={!description.trim() || amount <= 0}>Erfassen</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
