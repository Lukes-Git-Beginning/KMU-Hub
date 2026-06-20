import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Eye, Pencil } from 'lucide-react'
import type { ReportDocument, ReportRow } from '@/api/berichte-types'
import { useReportDocument, useUpdateReportDocument } from '@/api/hooks/useBerichte'
import { Skeleton } from '@/components/shared'
import { ReportStatusBadge } from './ReportStatusBadge'
import { BlockEditor } from './BlockEditor'
import { BLOCK_META, blockSummary, estimatePageCount } from './doc-utils'

interface ReportDocumentEditorProps {
  documentId: string
  onBack: () => void
}

export function ReportDocumentEditor({ documentId, onBack }: ReportDocumentEditorProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useReportDocument(documentId)
  const doc = data?.document

  if (isLoading || !doc) {
    return (
      <div className="flex flex-1 flex-col overflow-hidden">
        <header className="flex items-center gap-3 border-b border-border bg-background px-6 py-3">
          <button
            type="button"
            onClick={onBack}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
            aria-label={t('berichte.docs.back')}
          >
            <ArrowLeft className="h-4 w-4" />
          </button>
          <Skeleton className="h-7 w-56 flex-1 max-w-xs rounded-md" />
        </header>
        <div className="flex-1 overflow-y-auto p-6">
          <div className="mx-auto max-w-3xl space-y-3" aria-busy="true">
            <Skeleton className="h-10 w-full rounded-lg" />
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-14 w-full rounded-lg" />
            ))}
          </div>
        </div>
      </div>
    )
  }

  // Key on doc.id so local edit state resets cleanly when switching documents.
  return <DocumentEditorInner key={doc.id} doc={doc} onBack={onBack} />
}

/** Read mode is the default; released documents are locked (read only). */
function DocumentEditorInner({ doc, onBack }: { doc: ReportDocument; onBack: () => void }) {
  const { t } = useTranslation()
  const updateMutation = useUpdateReportDocument()
  const locked = doc.status === 'released' || doc.status === 'archived'

  const [rows, setRows] = useState<ReportRow[]>(doc.rows)
  const [mode, setMode] = useState<'read' | 'edit'>('edit')
  const firstRun = useRef(true)

  // Debounced auto-save when the block structure changes.
  useEffect(() => {
    if (firstRun.current) {
      firstRun.current = false
      return
    }
    const handle = setTimeout(() => {
      updateMutation.mutate({ id: doc.id, rows })
    }, 800)
    return () => clearTimeout(handle)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, doc.id])

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <header className="flex items-center gap-3 border-b border-border bg-background px-6 py-3">
        <button
          type="button"
          onClick={onBack}
          className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          aria-label={t('berichte.docs.back')}
        >
          <ArrowLeft className="h-4 w-4" />
        </button>

        <input
          defaultValue={doc.title}
          onBlur={(e) => {
            const value = e.target.value.trim()
            if (value && value !== doc.title) updateMutation.mutate({ id: doc.id, title: value })
          }}
          className="min-w-0 flex-1 rounded-lg border border-transparent bg-transparent px-2 py-1 text-base font-semibold text-foreground transition-colors hover:border-border focus:border-border focus:outline-none focus:ring-2 focus:ring-focus-ring"
          aria-label={t('berichte.docs.titleLabel')}
        />

        <ReportStatusBadge status={doc.status} />

        {/* Read / edit toggle (hidden while locked) */}
        {!locked && (
          <div className="flex overflow-hidden rounded-lg border border-border">
            <button
              type="button"
              onClick={() => setMode('read')}
              className={`flex items-center gap-1.5 px-2.5 py-1.5 text-xs ${mode === 'read' ? 'bg-primary-light text-primary' : 'text-muted-foreground hover:bg-secondary'}`}
            >
              <Eye className="h-3.5 w-3.5" />
              {t('berichte.docs.read')}
            </button>
            <button
              type="button"
              onClick={() => setMode('edit')}
              className={`flex items-center gap-1.5 px-2.5 py-1.5 text-xs ${mode === 'edit' ? 'bg-primary-light text-primary' : 'text-muted-foreground hover:bg-secondary'}`}
            >
              <Pencil className="h-3.5 w-3.5" />
              {t('berichte.docs.edit')}
            </button>
          </div>
        )}

        <span className="text-xs text-muted-foreground">
          {t('berichte.docs.pages', { count: estimatePageCount({ ...doc, rows }) })}
        </span>
      </header>

      <div className="flex-1 overflow-y-auto p-6">
        {mode === 'edit' && !locked ? (
          <BlockEditor rows={rows} onChange={setRows} />
        ) : (
          <ReadOutline rows={rows} />
        )}
      </div>
    </div>
  )
}

/** Read mode (B1-3): structural outline. Clean document render lands in R-2. */
function ReadOutline({ rows }: { rows: ReportRow[] }) {
  const { t } = useTranslation()
  return (
    <div className="mx-auto max-w-3xl space-y-3">
      {rows.map((row) => (
        <div key={row.id} className={row.columns.length > 1 ? 'flex gap-2' : undefined}>
          {row.columns.map((col) => (
            <div key={col.id} className="space-y-1.5" style={{ flex: col.width ?? 1 }}>
              {col.blocks.map((block) => {
                const meta = BLOCK_META[block.type]
                const Icon = meta.icon
                const summary = blockSummary(block)
                return (
                  <div
                    key={block.id}
                    className="flex items-start gap-2 rounded-lg border border-border bg-card p-2.5"
                  >
                    <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-secondary text-muted-foreground">
                      <Icon className="h-3.5 w-3.5" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="text-[11px] font-medium text-foreground">
                        {t(meta.labelKey)}
                        {block.type === 'heading' ? ` ${block.level}` : ''}
                      </p>
                      {summary && (
                        <p className="truncate text-[11px] text-muted-foreground">{summary}</p>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          ))}
        </div>
      ))}
    </div>
  )
}
