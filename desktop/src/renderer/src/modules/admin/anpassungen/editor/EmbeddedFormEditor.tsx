/**
 * EmbeddedFormEditor (Modul-Editor, Darien 2026-08-04) — the ticket form is edited
 * INSIDE the module editor, not by jumping out to the Formulare module and not in a
 * separate window. Editing a channel's intake form is part of configuring the
 * module, so it belongs on the same canvas as everything else.
 *
 * Rather than lifting ~1000 lines of builder UI out of FormularePage (a 5k-line
 * page), this mounts that page with a draft already open: FormularePage renders its
 * builder whenever the store holds a draft, so the canvas shows the builder and
 * nothing else. Deliberately rendered OUTSIDE EditorSurfaceProvider — the builder is
 * meant to be operated here, not previewed, so no edit-in-place and no action guard.
 *
 * Closing: the builder's own "Zurück" clears the store draft; that transition is
 * what brings the module preview back (instead of dropping into the Formulare list).
 */
import { useEffect, useRef, Suspense } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft } from 'lucide-react'
import { useFormSchema } from '@/api/hooks/useFormulare'
import { useFormulareStore } from '@/stores/formulare'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import FormularePage, { schemaToDraft } from '@/modules/formulare/FormularePage'

export function EmbeddedFormEditor({
  formId,
  onBack,
}: {
  formId: string
  onBack: () => void
}): React.ReactElement {
  const { t } = useTranslation()
  const { data: schema, isLoading } = useFormSchema(formId)
  const draft = useFormulareStore((s) => s.draft)
  const openDraft = useFormulareStore((s) => s.openDraft)
  const closeDraft = useFormulareStore((s) => s.closeDraft)
  // Guards the "draft went away → leave" effect below: it must not fire in the
  // window between mounting and the schema arriving.
  const opened = useRef(false)

  useEffect(() => {
    if (!schema) return
    openDraft(schemaToDraft(schema))
    opened.current = true
  }, [schema, openDraft])

  // Clear the draft when leaving, so the Formulare module isn't left mid-edit.
  useEffect(() => () => { closeDraft() }, [closeDraft])

  // The builder's own back button clears the draft — without this FormularePage
  // would fall through to the Formulare overview inside the editor window.
  useEffect(() => {
    if (opened.current && !draft) onBack()
  }, [draft, onBack])

  return (
    <div className="flex h-full min-h-0 flex-col bg-background">
      <div className="flex h-10 shrink-0 items-center gap-2 border-b px-3">
        <button
          type="button"
          onClick={onBack}
          className="flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
          {t('customization.editor.channels.backToModule')}
        </button>
        <span className="truncate text-xs text-muted-foreground">
          {t('customization.editor.channels.editingForm', { form: schema?.title ?? '' })}
        </span>
      </div>
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
        {isLoading || !schema ? (
          <ModuleLoadingFallback />
        ) : (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <FormularePage embedded />
          </Suspense>
        )}
      </div>
    </div>
  )
}
