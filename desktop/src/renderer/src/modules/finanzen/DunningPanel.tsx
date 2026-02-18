import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  AlertTriangle,
  Send,
  Download,
  Settings2,
  ChevronRight,
  RefreshCw,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  useDunnings,
  useDetectDunnings,
  useSendDunning,
  useEscalateDunning,
  useDunningConfig,
  useUpdateDunningConfig,
  useDownloadDunningPDF,
} from '@/api/hooks/useFinance'
import { formatEUR } from '@/stores/finance'
import type { DunningRecord, DunningStatus } from '@/types/finance-types'
import { EmptyState } from '@/components/shared'

const LEVEL_LABELS: Record<number, string> = {
  1: 'Zahlungserinnerung',
  2: '1. Mahnung',
  3: '2. Mahnung / Letzte Mahnung',
}

const LEVEL_COLORS: Record<number, string> = {
  1: 'bg-warning-light text-warning',
  2: 'bg-orange-100 text-orange-600 dark:bg-orange-900/30 dark:text-orange-400',
  3: 'bg-error-light text-error',
}

const STATUS_CONFIG: Record<
  DunningStatus,
  { label: string; colors: string }
> = {
  draft: { label: 'Entwurf', colors: 'bg-secondary text-muted-foreground' },
  sent: { label: 'Gesendet', colors: 'bg-info-light text-info' },
  paid: { label: 'Bezahlt', colors: 'bg-success-light text-success' },
}

export function DunningPanel() {
  const [levelFilter, setLevelFilter] = useState<'all' | 1 | 2 | 3>('all')
  const [statusFilter, setStatusFilter] = useState<'all' | DunningStatus>('all')
  const [showConfig, setShowConfig] = useState(false)

  const { data: dunningsData, isLoading } = useDunnings()
  const detectDunnings = useDetectDunnings()
  const sendDunning = useSendDunning()
  const escalateDunning = useEscalateDunning()
  const downloadPDF = useDownloadDunningPDF()

  const dunnings = dunningsData?.dunnings ?? []

  const filtered = dunnings
    .filter((d) => levelFilter === 'all' || d.level === levelFilter)
    .filter((d) => statusFilter === 'all' || d.status === statusFilter)

  const openCount = dunnings.filter((d) => d.status !== 'paid').length
  const totalAmount = dunnings
    .filter((d) => d.status !== 'paid')
    .reduce((sum, d) => sum + Number(d.fee) + Number(d.interest), 0)

  const handleDetect = () => {
    detectDunnings.mutate(undefined, {
      onSuccess: (data) => {
        const count = data.dunnings?.length ?? 0
        toast.success(
          count > 0
            ? `${count} neue Mahnung(en) erstellt`
            : 'Keine neuen ueberfaelligen Rechnungen gefunden',
        )
      },
      onError: (err) => toast.error(err.message),
    })
  }

  const handleSend = (d: DunningRecord) => {
    sendDunning.mutate(d.id, {
      onSuccess: () => toast.success(`${LEVEL_LABELS[d.level]} gesendet`),
      onError: (err) => toast.error(err.message),
    })
  }

  const handleEscalate = (d: DunningRecord) => {
    escalateDunning.mutate(d.id, {
      onSuccess: () =>
        toast.success(
          `Eskaliert auf ${LEVEL_LABELS[Math.min(d.level + 1, 3) as 1 | 2 | 3]}`,
        ),
      onError: (err) => toast.error(err.message),
    })
  }

  return (
    <div className="space-y-4">
      {/* Header actions */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="rounded-lg border border-border bg-card p-3 min-w-[120px]">
            <p className="text-[10px] text-muted-foreground mb-1">Offene Mahnungen</p>
            <p className="text-lg font-semibold text-foreground">{openCount}</p>
          </div>
          <div className="rounded-lg border border-border bg-card p-3 min-w-[120px]">
            <p className="text-[10px] text-muted-foreground mb-1">Gebuehren gesamt</p>
            <p className="text-lg font-semibold text-foreground">{formatEUR(totalAmount)}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowConfig(true)}
          >
            <Settings2 className="mr-1.5 h-3.5 w-3.5" />
            Konfiguration
          </Button>
          <Button
            size="sm"
            onClick={handleDetect}
            disabled={detectDunnings.isPending}
          >
            <RefreshCw
              className={`mr-1.5 h-3.5 w-3.5 ${detectDunnings.isPending ? 'animate-spin' : ''}`}
            />
            Ueberfaellige Rechnungen pruefen
          </Button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3 flex-wrap">
        <div className="flex items-center gap-1.5">
          <span className="text-xs text-muted-foreground">Mahnstufe:</span>
          {(['all', 1, 2, 3] as const).map((lvl) => (
            <button
              key={String(lvl)}
              onClick={() => setLevelFilter(lvl)}
              className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
                levelFilter === lvl
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-secondary text-muted-foreground hover:text-foreground'
              }`}
            >
              {lvl === 'all' ? 'Alle' : LEVEL_LABELS[lvl]}
            </button>
          ))}
        </div>
        <div className="h-4 w-px bg-border" />
        <div className="flex items-center gap-1.5">
          <span className="text-xs text-muted-foreground">Status:</span>
          {(['all', 'draft', 'sent', 'paid'] as const).map((st) => {
            const labels: Record<string, string> = {
              all: 'Alle',
              draft: 'Entwurf',
              sent: 'Gesendet',
              paid: 'Bezahlt',
            }
            return (
              <button
                key={st}
                onClick={() => setStatusFilter(st)}
                className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
                  statusFilter === st
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-secondary text-muted-foreground hover:text-foreground'
                }`}
              >
                {labels[st]}
              </button>
            )
          })}
        </div>
      </div>

      {/* Dunning list */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
          Lade Mahnungen...
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={AlertTriangle}
          title="Keine Mahnungen"
          description={
            dunnings.length === 0
              ? 'Keine ueberfaelligen Rechnungen gefunden'
              : 'Keine Mahnungen mit diesen Filtern gefunden'
          }
        />
      ) : (
        <div className="rounded-lg border border-border bg-card overflow-hidden">
          <div className="grid grid-cols-[80px_1fr_100px_140px_100px_80px_160px] gap-3 px-4 py-3 text-xs font-medium text-muted-foreground border-b border-border bg-secondary/30">
            <span>Rechnung</span>
            <span>Mahnstufe</span>
            <span>Gebuehr</span>
            <span>Zinsen</span>
            <span>Gesendet</span>
            <span>Status</span>
            <span className="text-right">Aktionen</span>
          </div>
          {filtered.map((d) => {
            const sc = STATUS_CONFIG[d.status]
            return (
              <div
                key={d.id}
                className="grid grid-cols-[80px_1fr_100px_140px_100px_80px_160px] gap-3 items-center px-4 py-3 border-b border-border-muted hover:bg-secondary/30 transition-colors"
              >
                <span className="text-sm font-mono text-primary">{d.invoice_id.slice(0, 8)}</span>
                {/* Level with visual indicator */}
                <div className="flex flex-col gap-1">
                  <span
                    className={`inline-flex items-center self-start rounded-full px-2 py-0.5 text-[10px] font-medium ${LEVEL_COLORS[d.level]}`}
                  >
                    {LEVEL_LABELS[d.level]}
                  </span>
                  <div className="flex items-center gap-0.5">
                    {[1, 2, 3].map((step) => (
                      <div key={step} className="flex items-center">
                        <div
                          className={`h-1.5 w-1.5 rounded-full ${
                            step <= d.level
                              ? step === 3
                                ? 'bg-error'
                                : step === 2
                                  ? 'bg-orange-500'
                                  : 'bg-warning'
                              : 'bg-secondary'
                          }`}
                        />
                        {step < 3 && (
                          <ChevronRight
                            className={`h-2.5 w-2.5 ${
                              step < d.level
                                ? 'text-muted-foreground'
                                : 'text-secondary'
                            }`}
                          />
                        )}
                      </div>
                    ))}
                  </div>
                </div>
                <span className="text-sm font-medium text-foreground">
                  {formatEUR(d.fee)}
                </span>
                <span className="text-sm text-foreground">{formatEUR(d.interest)}</span>
                <span className="text-xs text-muted-foreground">
                  {d.sent_at
                    ? new Date(d.sent_at).toLocaleDateString('de-DE')
                    : '--'}
                </span>
                <span
                  className={`inline-flex items-center self-start rounded-full px-2 py-0.5 text-[10px] font-medium ${sc.colors}`}
                >
                  {sc.label}
                </span>
                <div className="flex items-center justify-end gap-1.5">
                  {d.status === 'draft' && (
                    <button
                      onClick={() => handleSend(d)}
                      disabled={sendDunning.isPending}
                      className="flex items-center gap-1 rounded-md bg-primary px-2 py-1 text-[10px] text-primary-foreground hover:bg-button-primary-hover transition-colors"
                    >
                      <Send className="h-3 w-3" />
                      Senden
                    </button>
                  )}
                  {d.level < 3 && d.status !== 'paid' && (
                    <button
                      onClick={() => handleEscalate(d)}
                      disabled={escalateDunning.isPending}
                      className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-[10px] text-foreground hover:bg-secondary transition-colors"
                    >
                      <AlertTriangle className="h-3 w-3" />
                      Eskalieren
                    </button>
                  )}
                  <button
                    onClick={() => downloadPDF.mutate(d.id)}
                    disabled={downloadPDF.isPending}
                    className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-[10px] text-foreground hover:bg-secondary transition-colors"
                  >
                    <Download className="h-3 w-3" />
                    PDF
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Dunning Config Dialog */}
      <DunningConfigDialog open={showConfig} onOpenChange={setShowConfig} />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Dunning Configuration Dialog
// ---------------------------------------------------------------------------

function DunningConfigDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { data: config } = useDunningConfig()
  const updateConfig = useUpdateDunningConfig()

  const [l1Days, setL1Days] = useState('14')
  const [l2Days, setL2Days] = useState('14')
  const [l3Days, setL3Days] = useState('14')
  const [l1Fee, setL1Fee] = useState('0')
  const [l2Fee, setL2Fee] = useState('5')
  const [l3Fee, setL3Fee] = useState('10')

  // Populate from config when available
  useState(() => {
    if (config) {
      setL1Days(String(config.level1_days_after_due))
      setL2Days(String(config.level2_days_after_level1))
      setL3Days(String(config.level3_days_after_level2))
      setL1Fee(config.level1_fee)
      setL2Fee(config.level2_fee)
      setL3Fee(config.level3_fee)
    }
  })

  const handleSave = () => {
    updateConfig.mutate(
      {
        level1_days_after_due: Number(l1Days),
        level2_days_after_level1: Number(l2Days),
        level3_days_after_level2: Number(l3Days),
        level1_fee: l1Fee,
        level2_fee: l2Fee,
        level3_fee: l3Fee,
      },
      {
        onSuccess: () => {
          toast.success('Mahnkonfiguration gespeichert')
          onOpenChange(false)
        },
        onError: (err) => toast.error(err.message),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Mahnkonfiguration</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Level 1 */}
          <div className="rounded-lg border border-border p-3 space-y-2">
            <p className="text-sm font-medium text-foreground">
              Zahlungserinnerung (Stufe 1)
            </p>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label className="text-xs">Tage nach Faelligkeit</Label>
                <Input
                  type="number"
                  min={1}
                  value={l1Days}
                  onChange={(e) => setL1Days(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Mahngebuehr (EUR)</Label>
                <Input
                  type="number"
                  min={0}
                  step={0.01}
                  value={l1Fee}
                  onChange={(e) => setL1Fee(e.target.value)}
                />
              </div>
            </div>
          </div>

          {/* Level 2 */}
          <div className="rounded-lg border border-border p-3 space-y-2">
            <p className="text-sm font-medium text-foreground">
              1. Mahnung (Stufe 2)
            </p>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label className="text-xs">Tage nach Stufe 1</Label>
                <Input
                  type="number"
                  min={1}
                  value={l2Days}
                  onChange={(e) => setL2Days(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Mahngebuehr (EUR)</Label>
                <Input
                  type="number"
                  min={0}
                  step={0.01}
                  value={l2Fee}
                  onChange={(e) => setL2Fee(e.target.value)}
                />
              </div>
            </div>
          </div>

          {/* Level 3 */}
          <div className="rounded-lg border border-border p-3 space-y-2">
            <p className="text-sm font-medium text-foreground">
              2. Mahnung / Letzte Mahnung (Stufe 3)
            </p>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label className="text-xs">Tage nach Stufe 2</Label>
                <Input
                  type="number"
                  min={1}
                  value={l3Days}
                  onChange={(e) => setL3Days(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Mahngebuehr (EUR)</Label>
                <Input
                  type="number"
                  min={0}
                  step={0.01}
                  value={l3Fee}
                  onChange={(e) => setL3Fee(e.target.value)}
                />
              </div>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-end gap-2 pt-3 border-t border-border">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSave} disabled={updateConfig.isPending}>
            {updateConfig.isPending ? 'Speichert...' : 'Speichern'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
