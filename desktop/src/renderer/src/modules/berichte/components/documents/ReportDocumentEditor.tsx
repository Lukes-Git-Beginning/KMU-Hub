import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Archive, ArrowLeft, CalendarClock, Check, CheckCircle2, Copy, Eye, MoreVertical, Pencil, Printer, Send } from 'lucide-react'
import { toast } from 'sonner'
import type { ReportDocStatus, ReportDocument, ReportRow } from '@/api/berichte-types'
import {
  useCreateReportDocument,
  useReportDocument,
  useUpdateReportDocument,
} from '@/api/hooks/useBerichte'
import { Skeleton } from '@/components/shared'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { formatDate } from '@/lib/format'
import { ReportStatusBadge } from './ReportStatusBadge'
import { DocumentBlockEditor, DocumentReader, type DocRow } from '@/components/shared/document'
import { berichteBlockRegistry, berichteFullPageTypes, berichteQuickInserts } from './berichte-blocks'
import { ScheduleReportModal } from './ScheduleReportModal'
import { ShareActionsMenu } from './ShareActionsMenu'
import { blockCount, estimatePageCount } from './doc-utils'
import './report-print.css'

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
      <div className="flex h-full flex-col overflow-hidden">
        <header className="report-doc-chrome flex shrink-0 items-center gap-3 border-b border-border bg-background px-6 py-3">
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
  const createMutation = useCreateReportDocument()
  const locked = doc.status === 'released' || doc.status === 'archived'

  const [rows, setRows] = useState<ReportRow[]>(doc.rows)
  const [mode, setMode] = useState<'read' | 'edit'>(locked ? 'read' : initialMode)
  const [saveState, setSaveState] = useState<'idle' | 'saving' | 'saved'>('idle')
  const [scheduleOpen, setScheduleOpen] = useState(false)

  const totalBlocks = blockCount({ ...doc, rows })

  /** Move the document through its lifecycle (draft → final → released → archived). */
  function transitionTo(status: ReportDocStatus) {
    setSaveState('saving')
    updateMutation.mutate(
      { id: doc.id, status },
      {
        onSuccess: () => {
          setSaveState('saved')
          if (status === 'released') setMode('read')
          toast.success(t(`berichte.docs.lifecycle.toast.${status}`))
        },
      },
    )
  }

  /** Duplicate the current document as a fresh draft (only path out of archived). */
  function duplicate() {
    createMutation.mutate(
      {
        title: `${doc.title} ${t('berichte.docs.copySuffix')}`,
        module: doc.module,
        rows,
        settings: doc.settings,
      },
      { onSuccess: () => toast.success(t('berichte.docs.duplicated')) },
    )
  }

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
    <div className="flex h-full flex-col overflow-hidden">
      <header className="report-doc-chrome flex items-center gap-3 border-b border-border bg-background px-6 py-3">
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

        {doc.status === 'released' && doc.released_at && (
          <span className="hidden text-xs text-muted-foreground xl:inline">
            {t('berichte.docs.releasedOn', { date: formatDate(doc.released_at) })}
          </span>
        )}

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

        {/* Schedule — only meaningful once released; otherwise a guarded hint. */}
        {mode === 'read' && (
          <button
            type="button"
            onClick={() => doc.status === 'released' && setScheduleOpen(true)}
            disabled={doc.status !== 'released'}
            title={
              doc.status !== 'released'
                ? t('berichte.docs.schedule.guardNotReleased')
                : undefined
            }
            className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:bg-transparent disabled:hover:text-muted-foreground"
          >
            <CalendarClock className="h-3.5 w-3.5" />
            {t('berichte.docs.schedule.button')}
          </button>
        )}

        {/* Print / Save as PDF — available whenever the report is being read. */}
        {mode === 'read' && (
          <button
            type="button"
            onClick={() => window.print()}
            className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <Printer className="h-3.5 w-3.5" />
            {t('berichte.docs.print')}
          </button>
        )}

        {/* Share / distribute (R-5) — attach to task/contact, PDF, share link. */}
        {mode === 'read' && <ShareActionsMenu doc={doc} />}

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

        <StatusActions
          status={doc.status}
          totalBlocks={totalBlocks}
          onTransition={transitionTo}
          onDuplicate={duplicate}
        />
      </header>

      <div className="report-doc-body flex-1 overflow-y-auto p-6">
        {mode === 'edit' && !locked ? (
          <DocumentBlockEditor
            rows={rows as DocRow[]}
            onChange={(r) => setRows(r as ReportRow[])}
            registry={berichteBlockRegistry}
            quickInserts={berichteQuickInserts}
          />
        ) : (
          <DocumentReader
            rows={rows as DocRow[]}
            registry={berichteBlockRegistry}
            mode="print"
            settings={doc.settings}
            fullPageTypes={berichteFullPageTypes}
          />
        )}
      </div>

      <ScheduleReportModal doc={doc} open={scheduleOpen} onClose={() => setScheduleOpen(false)} />
    </div>
  )
}

/** Context-sensitive lifecycle controls in the editor header. */
function StatusActions({
  status,
  totalBlocks,
  onTransition,
  onDuplicate,
}: {
  status: ReportDocStatus
  totalBlocks: number
  onTransition: (status: ReportDocStatus) => void
  onDuplicate: () => void
}) {
  const { t } = useTranslation()
  const [menuOpen, setMenuOpen] = useState(false)
  const primaryCls =
    'flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50'

  if (status === 'draft') {
    return (
      <button
        type="button"
        onClick={() => onTransition('final')}
        disabled={totalBlocks === 0}
        title={totalBlocks === 0 ? t('berichte.docs.lifecycle.guardEmpty') : undefined}
        className={primaryCls}
      >
        <CheckCircle2 className="h-3.5 w-3.5" />
        {t('berichte.docs.lifecycle.markFinal')}
      </button>
    )
  }

  if (status === 'final') {
    return (
      <button type="button" onClick={() => onTransition('released')} className={primaryCls}>
        <Send className="h-3.5 w-3.5" />
        {t('berichte.docs.lifecycle.release')}
      </button>
    )
  }

  // released / archived → secondary actions in a menu.
  return (
    <Popover open={menuOpen} onOpenChange={setMenuOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          aria-label={t('berichte.docs.lifecycle.more')}
        >
          <MoreVertical className="h-4 w-4" />
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-48 p-1" align="end" sideOffset={4}>
        <button
          type="button"
          onClick={() => {
            setMenuOpen(false)
            onDuplicate()
          }}
          className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-xs text-foreground transition-colors hover:bg-secondary"
        >
          <Copy className="h-3.5 w-3.5 text-muted-foreground" />
          {t('berichte.docs.lifecycle.duplicateDraft')}
        </button>
        {status === 'released' && (
          <button
            type="button"
            onClick={() => {
              setMenuOpen(false)
              onTransition('archived')
            }}
            className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-xs text-foreground transition-colors hover:bg-secondary"
          >
            <Archive className="h-3.5 w-3.5 text-muted-foreground" />
            {t('berichte.docs.lifecycle.archive')}
          </button>
        )}
      </PopoverContent>
    </Popover>
  )
}

