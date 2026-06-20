import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Copy,
  Download,
  FileSpreadsheet,
  FileText,
  LayoutDashboard,
  LayoutGrid,
  List as ListIcon,
  Search,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import type { ReportDefinition } from '@/api/berichte-types'
import { isBuilderQuery } from '@/api/berichte-types'
import {
  useCreateDefinition,
  useDefinitions,
  useDeleteDefinition,
  useExportReport,
  useUpdateDefinition,
} from '@/api/hooks/useBerichte'
import { ConfirmDialog, EmptyState, ItemActions, SortMenu, type SortDirection } from '@/components/shared'
import { formatDate } from '@/lib/format'
import { resolveSource } from '../../report-sources/registry'
import { VIZ_GALLERY } from './builder-utils'

interface MyReportsLibraryProps {
  onOpen: (def: ReportDefinition) => void
}

function vizIconFor(def: ReportDefinition) {
  if (isBuilderQuery(def.query_config)) {
    return VIZ_GALLERY.find((v) => v.type === def.query_config.viz)?.icon ?? FileText
  }
  return FileText
}

function sourceLabelFor(def: ReportDefinition): string {
  if (isBuilderQuery(def.query_config)) {
    return resolveSource(def.query_config.sourceId)?.label ?? def.module
  }
  return def.module
}

export function MyReportsLibrary({ onOpen }: MyReportsLibraryProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useDefinitions({ kind: 'custom' })
  const createMutation = useCreateDefinition()
  const deleteMutation = useDeleteDefinition()
  const updateMutation = useUpdateDefinition()
  const exportMutation = useExportReport()

  const [search, setSearch] = useState('')
  const [moduleFilter, setModuleFilter] = useState('all')
  const [sortField, setSortField] = useState('updated_at')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')
  const [layout, setLayout] = useState<'grid' | 'list'>('grid')
  const [pendingDelete, setPendingDelete] = useState<ReportDefinition | null>(null)

  const defs = useMemo(() => data?.definitions ?? [], [data])
  const modules = useMemo(() => [...new Set(defs.map((d) => d.module))], [defs])

  const filtered = useMemo(() => {
    let r = defs
    if (search) r = r.filter((d) => d.name.toLowerCase().includes(search.toLowerCase()))
    if (moduleFilter !== 'all') r = r.filter((d) => d.module === moduleFilter)
    return [...r].sort((a, b) => {
      const av = sortField === 'name' ? a.name : a.updated_at
      const bv = sortField === 'name' ? b.name : b.updated_at
      const cmp = String(av).localeCompare(String(bv))
      return sortDir === 'desc' ? -cmp : cmp
    })
  }, [defs, search, moduleFilter, sortField, sortDir])

  function duplicate(def: ReportDefinition) {
    createMutation.mutate(
      {
        name: `${def.name} (Kopie)`,
        description: def.description,
        module: def.module,
        query_config: def.query_config,
        default_format: def.default_format,
        is_published: false,
      },
      {
        onSuccess: () => toast.success(t('berichte.library.duplicated', { defaultValue: 'Bericht dupliziert.' })),
      },
    )
  }

  function exportAs(def: ReportDefinition, format: 'pdf' | 'csv') {
    exportMutation.mutate({ definitionId: def.id, format })
  }

  function togglePin(def: ReportDefinition) {
    updateMutation.mutate(
      { id: def.id, is_published: !def.is_published },
      {
        onSuccess: () =>
          toast.success(
            def.is_published
              ? t('berichte.library.unpinned', { defaultValue: 'Vom Dashboard entfernt.' })
              : t('berichte.library.pinned', { defaultValue: 'Zum Dashboard hinzugefügt.' }),
          ),
      },
    )
  }

  function confirmDelete() {
    if (!pendingDelete) return
    deleteMutation.mutate(pendingDelete.id, {
      onSuccess: () => toast.success(t('berichte.library.deleted', { defaultValue: 'Bericht gelöscht.' })),
    })
    setPendingDelete(null)
  }

  function actionsFor(def: ReportDefinition) {
    return [
      { label: t('berichte.library.duplicate', { defaultValue: 'Duplizieren' }), icon: Copy, onClick: () => duplicate(def) },
      { label: t('berichte.library.exportPdf', { defaultValue: 'Als PDF' }), icon: FileText, onClick: () => exportAs(def, 'pdf') },
      { label: t('berichte.library.exportCsv', { defaultValue: 'Als CSV' }), icon: FileSpreadsheet, onClick: () => exportAs(def, 'csv') },
      {
        label: def.is_published
          ? t('berichte.library.unpin', { defaultValue: 'Vom Dashboard entfernen' })
          : t('berichte.library.pin', { defaultValue: 'Zum Dashboard' }),
        icon: LayoutDashboard,
        onClick: () => togglePin(def),
      },
      {
        label: t('berichte.library.delete', { defaultValue: 'Löschen' }),
        icon: Trash2,
        onClick: () => setPendingDelete(def),
        variant: 'destructive' as const,
      },
    ]
  }

  function Card({ def }: { def: ReportDefinition }) {
    const Icon = vizIconFor(def)
    return (
      <div
        role="button"
        tabIndex={0}
        onClick={() => onOpen(def)}
        onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && onOpen(def)}
        className="group flex cursor-pointer items-start gap-3 rounded-xl border border-border bg-card p-4 text-left transition-colors hover:border-primary/40"
      >
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-light text-primary">
          <Icon className="h-4.5 w-4.5" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-2">
            <h4 className="truncate text-sm font-medium text-foreground">{def.name}</h4>
            <div onClick={(e) => e.stopPropagation()}>
              <ItemActions items={actionsFor(def)} />
            </div>
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-1.5">
            <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground">
              {sourceLabelFor(def)}
            </span>
            {def.is_published && (
              <span className="inline-flex items-center gap-1 rounded-full bg-primary-light px-2 py-0.5 text-[10px] text-primary">
                <LayoutDashboard className="h-2.5 w-2.5" />
                {t('berichte.library.onDashboard', { defaultValue: 'Im Dashboard' })}
              </span>
            )}
          </div>
          <p className="mt-1.5 text-[11px] text-muted-foreground/70">
            {t('berichte.library.edited', { defaultValue: 'Bearbeitet' })} {formatDate(def.updated_at)}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-[200px] flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t('berichte.library.search', { defaultValue: 'Berichte durchsuchen…' })}
            className="w-full rounded-lg border border-border bg-card py-2 pl-9 pr-3 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
        </div>
        <SortMenu
          options={[
            { value: 'updated_at', label: t('berichte.library.sortEdited', { defaultValue: 'Bearbeitet' }) },
            { value: 'name', label: t('berichte.library.sortName', { defaultValue: 'Name' }) },
          ]}
          field={sortField}
          direction={sortDir}
          onChange={(f, d) => {
            setSortField(f)
            setSortDir(d)
          }}
        />
        <div className="flex overflow-hidden rounded-lg border border-border">
          <button
            type="button"
            onClick={() => setLayout('grid')}
            className={`p-2 ${layout === 'grid' ? 'bg-primary-light text-primary' : 'text-muted-foreground hover:bg-secondary'}`}
            aria-label={t('berichte.library.grid', { defaultValue: 'Kacheln' })}
          >
            <LayoutGrid className="h-4 w-4" />
          </button>
          <button
            type="button"
            onClick={() => setLayout('list')}
            className={`p-2 ${layout === 'list' ? 'bg-primary-light text-primary' : 'text-muted-foreground hover:bg-secondary'}`}
            aria-label={t('berichte.library.list', { defaultValue: 'Liste' })}
          >
            <ListIcon className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Module filter chips */}
      {modules.length > 1 && (
        <div className="flex flex-wrap gap-1.5">
          <button
            type="button"
            onClick={() => setModuleFilter('all')}
            className={`rounded-full border px-2.5 py-1 text-xs transition-colors ${
              moduleFilter === 'all' ? 'border-primary bg-primary-light text-primary' : 'border-border text-muted-foreground hover:bg-secondary'
            }`}
          >
            {t('berichte.library.allModules', { defaultValue: 'Alle' })}
          </button>
          {modules.map((m) => (
            <button
              key={m}
              type="button"
              onClick={() => setModuleFilter(m)}
              className={`rounded-full border px-2.5 py-1 text-xs capitalize transition-colors ${
                moduleFilter === m ? 'border-primary bg-primary-light text-primary' : 'border-border text-muted-foreground hover:bg-secondary'
              }`}
            >
              {m}
            </button>
          ))}
        </div>
      )}

      {/* Grid / list */}
      {isLoading ? (
        <p className="py-10 text-center text-sm text-muted-foreground">
          {t('berichte.library.loading', { defaultValue: 'Lade Berichte…' })}
        </p>
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={FileText}
          title={t('berichte.library.emptyTitle', { defaultValue: 'Noch keine eigenen Berichte' })}
          description={t('berichte.library.emptyHint', {
            defaultValue: 'Erstelle im Tab „Neuer Bericht" deinen ersten Bericht und speichere ihn.',
          })}
        />
      ) : (
        <div className={layout === 'grid' ? 'grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3' : 'flex flex-col gap-2'}>
          {filtered.map((def) => (
            <Card key={def.id} def={def} />
          ))}
        </div>
      )}

      <ConfirmDialog
        open={!!pendingDelete}
        onOpenChange={(o) => !o && setPendingDelete(null)}
        title={t('berichte.library.deleteTitle', { defaultValue: 'Bericht löschen?' })}
        description={t('berichte.library.deleteDesc', {
          defaultValue: 'Der Bericht „{name}" wird dauerhaft entfernt.',
          name: pendingDelete?.name ?? '',
        })}
        confirmLabel={t('berichte.library.delete', { defaultValue: 'Löschen' })}
        onConfirm={confirmDelete}
        variant="destructive"
      />
    </div>
  )
}
