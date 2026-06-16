/**
 * ExpenseFormDialog — Ausgabe erfassen/bearbeiten inkl. SKR-Kontierung +
 * Beleg-Upload (finanzen P2).
 *
 * Swap-ready: schreibt über `useFinanceLedger`-Hooks gegen die MSW-Handler.
 * Beleg = Dateiname (Metadaten) + Blob im Session-Cache für die Vorschau; beim
 * Backend-Anbindung übernimmt der Server Upload + URL.
 */
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Paperclip, X } from 'lucide-react'
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
import type { Expense } from '@/stores/finance'
import { useAddExpense, useUpdateExpense } from '@/api/hooks/useFinanceLedger'
import { useFinanceTenantStore } from '@/stores/financeTenant'
import { EXPENSE_CATEGORIES, SKR_ACCOUNTS, suggestAccount } from './lib/skr-accounts'
import { putReceipt } from './lib/receipt-cache'

interface ExpenseFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Wenn gesetzt: Bearbeiten-Modus (inkl. Kontierung) statt Neu. */
  editExpense?: Expense | null
}

const NO_ACCOUNT = '__none__'

export function ExpenseFormDialog({ open, onOpenChange, editExpense }: ExpenseFormDialogProps) {
  const { t } = useTranslation()
  const framework = useFinanceTenantStore((s) => s.chartFramework)
  const addExpense = useAddExpense()
  const updateExpense = useUpdateExpense()
  const fileInputRef = useRef<HTMLInputElement>(null)

  const isEdit = !!editExpense

  const [description, setDescription] = useState('')
  const [amount, setAmount] = useState(0)
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const [category, setCategory] = useState('Sonstiges')
  const [supplier, setSupplier] = useState('')
  const [project, setProject] = useState('')
  const [account, setAccount] = useState<string>(NO_ACCOUNT)
  const [accountTouched, setAccountTouched] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [receiptName, setReceiptName] = useState<string | undefined>(undefined)

  // Sync form state when (re)opening for create or edit.
  useEffect(() => {
    if (!open) return
    if (editExpense) {
      setDescription(editExpense.description)
      setAmount(editExpense.amount)
      setDate(editExpense.date)
      setCategory(editExpense.category)
      setSupplier(editExpense.supplier)
      setProject(editExpense.project ?? '')
      // Bereits kontiert → Konto übernehmen; sonst aus der Kategorie vorschlagen
      // (das "Kontieren" einer offenen Ausgabe soll direkt einen Vorschlag zeigen).
      if (editExpense.account) {
        setAccount(editExpense.account)
        setAccountTouched(true)
      } else {
        setAccount(suggestAccount(editExpense.category, framework) ?? NO_ACCOUNT)
        setAccountTouched(false)
      }
      setReceiptName(editExpense.receiptName)
    } else {
      setDescription('')
      setAmount(0)
      setDate(new Date().toISOString().split('T')[0])
      setCategory('Sonstiges')
      setSupplier('')
      setProject('')
      setAccount(suggestAccount('Sonstiges', framework) ?? NO_ACCOUNT)
      setAccountTouched(false)
      setReceiptName(undefined)
    }
    setFile(null)
  }, [open, editExpense, framework])

  // Auto-suggest account from category until the user picks one manually.
  const handleCategory = (value: string) => {
    setCategory(value)
    if (!accountTouched) {
      setAccount(suggestAccount(value, framework) ?? NO_ACCOUNT)
    }
  }

  const handleFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0]
    if (f) {
      setFile(f)
      setReceiptName(f.name)
    }
  }

  const removeFile = () => {
    setFile(null)
    setReceiptName(undefined)
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  const valid = description.trim().length > 0 && amount > 0

  const handleSave = async () => {
    if (!valid) return
    const payload = {
      description: description.trim(),
      amount,
      date,
      category,
      supplier: supplier.trim(),
      project: project.trim() || undefined,
      account: account === NO_ACCOUNT ? undefined : account,
      receiptName,
    }
    try {
      if (isEdit && editExpense) {
        await updateExpense.mutateAsync({ id: editExpense.id, data: payload })
        if (file) putReceipt(editExpense.id, file)
        toast.success(t('buchhaltung.toast.expenseUpdated'))
      } else {
        const res = await addExpense.mutateAsync(payload)
        if (file && res.expense?.id) putReceipt(res.expense.id, file)
        toast.success(t('buchhaltung.toast.expenseRecorded'))
      }
      onOpenChange(false)
    } catch {
      toast.error(t('common.error'))
    }
  }

  const accounts = SKR_ACCOUNTS[framework]

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? t('buchhaltung.editExpense') : t('buchhaltung.newExpense')}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label>{t('buchhaltung.form.description')} *</Label>
            <Input
              autoFocus
              placeholder={t('buchhaltung.form.descriptionPlaceholder')}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('buchhaltung.form.amount')} *</Label>
              <Input
                type="number"
                min={0.01}
                step={0.01}
                placeholder="0.00"
                value={amount || ''}
                onChange={(e) => setAmount(Number(e.target.value))}
              />
            </div>
            <div className="space-y-1.5">
              <Label>{t('buchhaltung.form.date')}</Label>
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('buchhaltung.table.category')}</Label>
              <Select value={category} onValueChange={handleCategory}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {EXPENSE_CATEGORIES.map((c) => (
                    <SelectItem key={c.value} value={c.value}>
                      {t(`buchhaltung.categories.${c.value}`, { defaultValue: c.value })}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('buchhaltung.table.supplier')}</Label>
              <Input
                placeholder={t('buchhaltung.form.supplierPlaceholder')}
                value={supplier}
                onChange={(e) => setSupplier(e.target.value)}
              />
            </div>
          </div>

          {/* Kontierung — leichtes SKR-Sachkonto-Mapping für den DATEV-Export */}
          <div className="space-y-1.5">
            <Label>
              {t('buchhaltung.form.account')}
              <span className="ml-1.5 text-[11px] font-normal text-muted-foreground">
                {t('buchhaltung.form.accountHint', { framework })}
              </span>
            </Label>
            <Select
              value={account}
              onValueChange={(v) => {
                setAccount(v)
                setAccountTouched(true)
              }}
            >
              <SelectTrigger>
                <SelectValue placeholder={t('buchhaltung.form.accountPlaceholder')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NO_ACCOUNT}>{t('buchhaltung.form.accountNone')}</SelectItem>
                {accounts.map((a) => (
                  <SelectItem key={a.number} value={a.number}>
                    {a.number} · {a.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label>{t('buchhaltung.form.project')}</Label>
            <Input
              placeholder={t('buchhaltung.form.projectPlaceholder')}
              value={project}
              onChange={(e) => setProject(e.target.value)}
            />
          </div>

          {/* Beleg-Upload */}
          <div className="space-y-1.5">
            <Label>{t('buchhaltung.form.receipt')}</Label>
            {receiptName ? (
              <div className="flex items-center justify-between gap-2 rounded-lg border border-border bg-secondary/40 px-3 py-2">
                <span className="flex min-w-0 items-center gap-2 text-sm text-foreground">
                  <Paperclip className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                  <span className="truncate">{receiptName}</span>
                </span>
                <button
                  type="button"
                  onClick={removeFile}
                  aria-label={t('buchhaltung.form.receiptRemove')}
                  className="shrink-0 rounded-md p-1 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                >
                  <X className="h-4 w-4" aria-hidden="true" />
                </button>
              </div>
            ) : (
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                className="flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-border px-3 py-2.5 text-sm text-muted-foreground transition-colors hover:border-primary hover:text-foreground"
              >
                <Paperclip className="h-4 w-4" aria-hidden="true" />
                {t('buchhaltung.form.receiptUpload')}
              </button>
            )}
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*,application/pdf"
              className="hidden"
              onChange={handleFile}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSave} disabled={!valid || addExpense.isPending || updateExpense.isPending}>
            {isEdit ? t('common.save') : t('buchhaltung.form.record')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
