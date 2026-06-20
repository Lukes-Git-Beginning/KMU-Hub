import { useTranslation } from 'react-i18next'
import { ArrowLeft } from 'lucide-react'
import { useReportDocument, useUpdateReportDocument } from '@/api/hooks/useBerichte'
import { Skeleton } from '@/components/shared'
import { ReportStatusBadge } from './ReportStatusBadge'
import { BLOCK_META, blockSummary, estimatePageCount } from './doc-utils'

interface ReportDocumentEditorProps {
  documentId: string
  onBack: () => void
}

/**
 * R-0 editor shell: header (back, editable title, status, page count) + a
 * read-only structural outline of the report. The visual WYSIWYG block editor
 * (insert / drag-reorder / edit) and the read↔edit toggle land in R-1 / R-2.
 */
export function ReportDocumentEditor({ documentId, onBack }: ReportDocumentEditorProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useReportDocument(documentId)
  const updateMutation = useUpdateReportDocument()
  const doc = data?.document

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Sticky header — always visible while the outline scrolls */}
      <header className="flex items-center gap-3 border-b border-border bg-background px-6 py-3">
        <button
          type="button"
          onClick={onBack}
          className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          aria-label={t('berichte.docs.back')}
        >
          <ArrowLeft className="h-4 w-4" />
        </button>

        {doc ? (
          <input
            key={doc.id}
            defaultValue={doc.title}
            onBlur={(e) => {
              const value = e.target.value.trim()
              if (value && value !== doc.title) {
                updateMutation.mutate({ id: doc.id, title: value })
              }
            }}
            className="min-w-0 flex-1 rounded-lg border border-transparent bg-transparent px-2 py-1 text-base font-semibold text-foreground transition-colors hover:border-border focus:border-border focus:outline-none focus:ring-2 focus:ring-focus-ring"
            aria-label={t('berichte.docs.titleLabel')}
          />
        ) : (
          <Skeleton className="h-7 w-56 flex-1 max-w-xs rounded-md" />
        )}

        {doc && (
          <>
            <ReportStatusBadge status={doc.status} />
            <span className="text-xs text-muted-foreground">
              {t('berichte.docs.pages', { count: estimatePageCount(doc) })}
            </span>
          </>
        )}
      </header>

      {/* Body */}
      <div className="flex-1 overflow-y-auto p-6">
        {isLoading || !doc ? (
          <div className="mx-auto max-w-3xl space-y-3" aria-busy="true">
            <Skeleton className="h-10 w-full rounded-lg" />
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-14 w-full rounded-lg" />
            ))}
          </div>
        ) : (
          <div className="mx-auto max-w-3xl space-y-3">
            <div className="rounded-lg border border-dashed border-border bg-secondary/20 p-3 text-xs text-muted-foreground">
              {t('berichte.docs.editorHint')}
            </div>

            {doc.rows.map((row) => (
              <div
                key={row.id}
                className={row.columns.length > 1 ? 'flex gap-2' : undefined}
              >
                {row.columns.map((col) => (
                  <div
                    key={col.id}
                    className="space-y-1.5"
                    style={{ flex: col.width ?? 1 }}
                  >
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
                              <p className="truncate text-[11px] text-muted-foreground">
                                {summary}
                              </p>
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
        )}
      </div>
    </div>
  )
}
