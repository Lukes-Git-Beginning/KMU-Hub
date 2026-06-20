import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Check, Eye, Pencil } from 'lucide-react'
import type { ReportDocument, ReportRow } from '@/api/berichte-types'
import { useReportDocument, useUpdateReportDocument } from '@/api/hooks/useBerichte'
import { Skeleton } from '@/components/shared'
import { ReportStatusBadge } from './ReportStatusBadge'
import { BlockEditor } from './BlockEditor'
import { DocumentReader } from './DocumentReader'
import { estimatePageCount } from './doc-utils'

interface ReportDocumentEditorProps {
  documentId: string
  onBack: () => void
  /** Read mode when opened from the library; edit when freshly created. */
  initialMode?: 'read' | 'edit'
}

export function ReportDocumentEditor({
  documentId,
  onBack,
  initialMode = 'read',
}: ReportDocumentEditorProps) {
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
  return <DocumentEditorInner key={doc.id} doc={doc} onBack={onBack} initialMode={initialMode} />
}

/** Read mode is the default; released/archived documents are locked (read only). */
function DocumentEditorInner({
  doc,
  onBack,
  initialMode,
}: {
  doc: ReportDocument
  onBack: () => void
  initialMode: 'read' | 'edit'
}) {
  const { t } = useTranslation()
  const updateMutation = useUpdateReportDocument()
  const locked = doc.status === 'released' || doc.status === 'archived'

  const [rows, setRows] = useState<ReportRow[]>(doc.rows)
  const [mode, setMode] = useState<'read' | 'edit'>(locked ? 'read' : initialMode)
  const [saveState, setSaveState] = useState<'idle' | 'saving' | 'saved'>('idle')

  // Debounced auto-save once the block structure actually changes. Comparing
  // against the initial doc.rows reference skips the mount (StrictMode-safe).
  useEffect(() => {
    if (rows === doc.rows) return
    setSaveState('saving')
    const handle = setTimeout(() => {
      updateMutation.mutate({ id: doc.id, rows }, { onSuccess: () => setSaveState('saved') })
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
            if (value && value !== doc.title) {
              setSaveState('saving')
              updateMutation.mutate(
                { id: doc.id, title: value },
                { onSuccess: () => setSaveState('saved') },
              )
            }
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

        {saveState === 'saving' ? (
          <span className="text-xs text-muted-foreground">{t('berichte.docs.saving')}</span>
        ) : saveState === 'saved' ? (
          <span className="flex items-center gap-1 text-xs text-muted-foreground">
            <Check className="h-3 w-3 text-success" />
            {t('berichte.docs.saved')}
          </span>
        ) : null}

        <span className="text-xs text-muted-foreground">
          {t('berichte.docs.pages', { count: estimatePageCount({ ...doc, rows }) })}
        </span>
      </header>

      <div className="flex-1 overflow-y-auto p-6">
        {mode === 'edit' && !locked ? (
          <BlockEditor rows={rows} onChange={setRows} />
        ) : (
          <DocumentReader rows={rows} settings={doc.settings} />
        )}
      </div>
    </div>
  )
}

