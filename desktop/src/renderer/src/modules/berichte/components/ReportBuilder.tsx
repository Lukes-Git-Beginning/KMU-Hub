import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Calendar, Download, FileBarChart, FilePlus, FileText, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import type { ReportDefinition, ReportFormat } from '@/api/berichte-types'
import { useExportReport } from '@/api/hooks/useBerichte'

interface ReportBuilderProps {
  definitions: ReportDefinition[]
  isLoading?: boolean
}

const FORMAT_OPTIONS: { value: ReportFormat; icon: typeof FileText }[] = [
  { value: 'pdf', icon: FileText },
  { value: 'xlsx', icon: FileBarChart },
  { value: 'csv', icon: FileText },
]

export function ReportBuilder({ definitions, isLoading }: ReportBuilderProps) {
  const { t } = useTranslation()
  const exportMutation = useExportReport()

  const systemDefs = useMemo(
    () => definitions.filter((d) => d.kind === 'system' && d.is_published),
    [definitions],
  )

  const [selectedDefId, setSelectedDefId] = useState<string>('')
  const [dateFrom, setDateFrom] = useState<string>('')
  const [dateTo, setDateTo] = useState<string>('')
  const [format, setFormat] = useState<ReportFormat>('pdf')

  const selectedDef = systemDefs.find((d) => d.id === selectedDefId) ?? null

  const handleGenerate = () => {
    if (!selectedDef) {
      toast.error(t('berichte.erstellen.errorBericht', { defaultValue: 'Bitte einen Bericht auswählen.' }))
      return
    }
    const params: Record<string, unknown> = {}
    if (dateFrom) params.date_from = dateFrom
    if (dateTo) params.date_to = dateTo
    exportMutation.mutate(
      { definitionId: selectedDef.id, format, params },
      {
        onSuccess: () =>
          toast.success(
            t('berichte.erstellen.successToast', {
              name: selectedDef.name,
              format: format.toUpperCase(),
            }),
          ),
        onError: (err) => toast.error((err as Error).message),
      },
    )
  }

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
      <div className="lg:col-span-2">
        <div className="rounded-xl border border-border bg-card p-6">
          <div className="mb-6 flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
              <FilePlus className="h-5 w-5 text-primary" />
            </div>
            <div>
              <h3 className="text-sm font-medium text-foreground">{t('berichte.erstellen.title')}</h3>
              <p className="text-xs text-muted-foreground">{t('berichte.erstellen.hint')}</p>
            </div>
          </div>

          <div className="space-y-5">
            {/* System report picker */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                {t('berichte.erstellen.bericht', { defaultValue: 'Bericht' })}
              </label>
              <select
                value={selectedDefId}
                onChange={(e) => setSelectedDefId(e.target.value)}
                disabled={isLoading}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring disabled:opacity-60"
              >
                <option value="">
                  {isLoading
                    ? t('berichte.erstellen.laedt', { defaultValue: 'Lade...' })
                    : t('berichte.erstellen.berichtSelect', { defaultValue: 'Bericht auswählen...' })}
                </option>
                {systemDefs.map((def) => (
                  <option key={def.id} value={def.id}>
                    {def.name} — {def.module}
                  </option>
                ))}
              </select>
              {selectedDef && (
                <p className="mt-1.5 text-xs text-muted-foreground">{selectedDef.description}</p>
              )}
            </div>

            {/* Date range */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                {t('berichte.erstellen.zeitraum')}
              </label>
              <div className="flex items-center gap-2">
                <div className="relative flex-1">
                  <Calendar className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                  <input
                    type="date"
                    value={dateFrom}
                    onChange={(e) => setDateFrom(e.target.value)}
                    className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  />
                </div>
                <span className="text-xs text-muted-foreground">{t('berichte.erstellen.bis')}</span>
                <div className="relative flex-1">
                  <Calendar className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                  <input
                    type="date"
                    value={dateTo}
                    onChange={(e) => setDateTo(e.target.value)}
                    className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                  />
                </div>
              </div>
              <p className="mt-1.5 text-[10px] text-muted-foreground/60">
                {t('berichte.erstellen.zeitraumHint', {
                  defaultValue: 'Leer lassen, um den Standard-Zeitraum des Berichts zu verwenden.',
                })}
              </p>
            </div>

            {/* Format radio */}
            <div>
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                {t('berichte.erstellen.ausgabeformat')}
              </label>
              <div className="flex gap-3">
                {FORMAT_OPTIONS.map(({ value, icon: Icon }) => (
                  <button
                    key={value}
                    onClick={() => setFormat(value)}
                    type="button"
                    className={`flex items-center gap-2 rounded-lg border px-4 py-2 text-sm transition-colors ${
                      format === value
                        ? 'border-primary bg-primary-light text-primary'
                        : 'border-border text-muted-foreground hover:bg-secondary'
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                    {value.toUpperCase()}
                  </button>
                ))}
              </div>
            </div>

            {/* Generate button */}
            <button
              onClick={handleGenerate}
              disabled={exportMutation.isPending || !selectedDef}
              className="flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover disabled:opacity-60"
            >
              {exportMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Download className="h-4 w-4" />
              )}
              {exportMutation.isPending
                ? t('berichte.erstellen.wirdGeneriert')
                : t('berichte.erstellen.generieren')}
            </button>
          </div>
        </div>
      </div>

      {/* Side panel — system catalog overview */}
      <div className="lg:col-span-1">
        <div className="rounded-lg border border-border bg-card p-4">
          <div className="mb-4 flex items-center gap-2">
            <FileText className="h-4 w-4 text-muted-foreground" />
            <h3 className="text-sm font-medium text-foreground">
              {t('berichte.gespeichert.title')}
            </h3>
          </div>
          <div className="space-y-2">
            {systemDefs.map((def) => (
              <button
                key={def.id}
                type="button"
                onClick={() => setSelectedDefId(def.id)}
                className={`block w-full rounded-lg border p-3 text-left transition-colors ${
                  selectedDefId === def.id
                    ? 'border-primary bg-primary/5'
                    : 'border-border-muted hover:border-primary/30'
                }`}
              >
                <h4 className="mb-1 text-sm font-medium leading-tight text-foreground">
                  {def.name}
                </h4>
                <p className="line-clamp-2 text-xs text-muted-foreground">{def.description}</p>
                <span className="mt-2 inline-block rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
                  {def.module}
                </span>
              </button>
            ))}
            {systemDefs.length === 0 && !isLoading && (
              <p className="text-xs text-muted-foreground/70">
                {t('berichte.gespeichert.empty')}
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
