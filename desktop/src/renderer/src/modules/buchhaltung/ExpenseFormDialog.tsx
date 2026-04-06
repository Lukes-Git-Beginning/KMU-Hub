import { useState } from 'react'
import { useTranslation } from 'react-i18next'
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
  'Büro', 'Beratung', 'Bankgebühren', 'Versicherung', 'Sonstiges',
]

interface ExpenseFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ExpenseFormDialog({ open, onOpenChange }: ExpenseFormDialogProps) {
  const { t } = useTranslation()
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
    toast.success(t('buchhaltung.toast.expenseRecorded'))
    reset()
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) reset(); onOpenChange(v) }}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('buchhaltung.newExpense')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label>{t('buchhaltung.form.description')} *</Label>
            <Input autoFocus placeholder={t('buchhaltung.form.descriptionPlaceholder')} value={description} onChange={(e) => setDescription(e.target.value)} />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('buchhaltung.form.amount')} *</Label>
              <Input type="number" min={0.01} step={0.01} placeholder="0.00" value={amount || ''} onChange={(e) => setAmount(Number(e.target.value))} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('buchhaltung.form.date')}</Label>
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('buchhaltung.table.category')}</Label>
              <Select value={category} onValueChange={setCategory}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {CATEGORIES.map((c) => <SelectItem key={c} value={c}>{t(`buchhaltung.categories.${c}`)}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('buchhaltung.table.supplier')}</Label>
              <Input placeholder={t('buchhaltung.form.supplierPlaceholder')} value={supplier} onChange={(e) => setSupplier(e.target.value)} />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label>{t('buchhaltung.form.project')}</Label>
            <Input placeholder={t('buchhaltung.form.projectPlaceholder')} value={project} onChange={(e) => setProject(e.target.value)} />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => { reset(); onOpenChange(false) }}>{t('common.cancel')}</Button>
          <Button onClick={handleSave} disabled={!description.trim() || amount <= 0}>{t('buchhaltung.form.record')}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
