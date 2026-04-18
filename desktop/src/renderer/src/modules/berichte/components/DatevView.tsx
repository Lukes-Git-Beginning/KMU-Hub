import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { BarChart3, Download, Loader2, Table2 } from 'lucide-react'
import { toast } from 'sonner'
import type { ReportColumn, ReportDefinition } from '@/api/berichte-types'
import { useExportReport, useRunReport } from '@/api/hooks/useBerichte'

interface DatevViewProps {
  definitions: ReportDefinition[]
}

type DatevVariant = 'bwa' | 'susa'

interface DatevConfig {
  value: DatevVariant
  kind: string
  labelKey: string
  icon: typeof BarChart3
}

const VARIANTS: DatevConfig[] = [
  { value: 'bwa', kind: 'datev_bwa', labelKey: 'berichte.datev.bwa', icon: BarChart3 },
  { value: 'susa', kind: 'datev_susa', labelKey: 'berichte.datev.susa', icon: Table2 },
]

function findDefinition(defs: ReportDefinition[], kind: string): ReportDefinition | null {
  return (
    defs.find((d) => {
      const cfgKind = (d.query_config as { kind?: unknown } | undefined)?.kind
      return typeof cfgKind === 'string' && cfgKind === kind
    }) ?? null
  )
}

function renderCell(value: unknown, col: ReportColumn): string {
  if (value === null || value === undefined) return '—'
  if (col.type === 'currency' && typeof value === 'number') {
    return `${value.toLocaleString('de-DE')} EUR`
  }
  if (typeof value === 'number') return value.toLocaleString('de-DE')
  return String(value)
}

export function DatevView({ definitions }: DatevViewProps) {
  const { t } = useTranslation()
  const runMutation = useRunReport()
  const exportMutation = useExportReport()

  const bwaDef = useMemo(() => findDefinition(definitions, 'datev_bwa'), [definitions])
  const susaDef = useMemo(() => findDefinition(definitions, 'datev_susa'), [definitions])
  const availableVariants = useMemo(
    () => VARIANTS.filter((v) => (v.value === 'bwa' ? bwaDef : susaDef) !== null),
    [bwaDef, susaDef],
  )

  const activeVariant: DatevVariant = availableVariants[0]?.value ?? 'bwa'
  const activeDef = activeVariant === 'bwa' ? bwaDef : susaDef

  useEffect(() => {
    if (!activeDef) return
    runMutation.mutate({ definitionId: activeDef.id })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeDef?.id])

  const result = runMutation.data?.result
  const columns: ReportColumn[] = result?.columns ?? []
  const rows = result?.rows ?? []

  const handleExport = () => {
    if (!activeDef) return
    exportMutation.mutate(
      { definitionId: activeDef.id, format: 'csv', params: {} },
      {
        onSuccess: () => toast.success(t('berichte.datev.exportToast')),
        onError: (err) => toast.error((err as Error).message),
      },
    )
  }

  if (!bwaDef && !susaDef) {
    return (
      <div className="rounded-lg border border-border bg-card p-6 text-center text-sm text-muted-foreground">
        {t('berichte.datev.unavailable', {
          defaultValue:
            'DATEV-Berichte sind noch nicht konfiguriert. Aktiviere das Backend-Modul und starte das Gateway neu.',
        })}
      </div>
    )
  }

  return (
    <>
      <div className="mb-4 flex items-center justify-between">
        <div className="flex gap-1">
          {availableVariants.map((v) => (
            <button
              key={v.value}
              className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors ${
                activeVariant === v.value
                  ? 'bg-primary/10 font-medium text-primary'
                  : 'text-muted-foreground hover:bg-secondary'
              }`}
            >
              <v.icon className="h-3.5 w-3.5" />
              {t(v.labelKey)}
            </button>
          ))}
        </div>
        <button
          onClick={handleExport}
          disabled={!activeDef || exportMutation.isPending}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover disabled:opacity-60"
        >
          {exportMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Download className="h-4 w-4" />
          )}
          {t('berichte.datev.export')}
        </button>
      </div>

      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <div className="border-b border-border px-4 py-3">
          <h3 className="text-sm font-medium text-foreground">{activeDef?.name ?? '—'}</h3>
          <p className="text-xs text-muted-foreground">{t('berichte.datev.zeitraum')}</p>
        </div>

        {runMutation.isPending ? (
          <div className="flex items-center justify-center gap-2 p-10 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            {t('berichte.datev.laedt', { defaultValue: 'Bericht wird geladen...' })}
          </div>
        ) : rows.length === 0 ? (
          <div className="p-8 text-center text-sm text-muted-foreground">
            {t('berichte.datev.noRows', { defaultValue: 'Keine Daten im aktuellen Zeitraum.' })}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/30">
                  {columns.map((col) => (
                    <th
                      key={col.key}
                      className={`px-4 py-2.5 text-xs font-medium text-muted-foreground ${
                        col.type === 'currency' || col.type === 'number'
                          ? 'text-right'
                          : 'text-left'
                      }`}
                    >
                      {col.label}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {rows.map((row, idx) => (
                  <tr
                    key={idx}
                    className="border-b border-border/40 last:border-0 hover:bg-muted/20"
                  >
                    {columns.map((col) => {
                      const value = row[col.key]
                      const isNegative = typeof value === 'number' && value < 0
                      const alignRight = col.type === 'currency' || col.type === 'number'
                      return (
                        <td
                          key={col.key}
                          className={`px-4 py-2 text-sm ${
                            alignRight ? 'text-right' : 'text-left'
                          } ${isNegative ? 'text-destructive' : 'text-foreground'}`}
                        >
                          {renderCell(value, col)}
                        </td>
                      )
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </>
  )
}
