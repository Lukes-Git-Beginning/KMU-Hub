/**
 * IntakeFormFill — renders an editor-built intake form and dispatches the
 * submission to its bound target via the shared intake engine.
 *
 * Reused by every non-agent channel (internal self-service, external public
 * page). The requester name/email roles are NOT rendered when a `requester` is
 * supplied (internal self-service auto-fills them from the profile); the engine
 * still receives them via the build context. Themed with app tokens (internal
 * surfaces); the external public page can wrap it with its own chrome.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Loader2 } from 'lucide-react'
import { useFormSchema } from '@/api/hooks/useFormulare'
import type { FormField } from '@/api/formulare-types'
import { useIntakeSubmit } from './useIntakeSubmit'
import { getIntakeTarget } from './registry'
import type { IntakeBuildContext } from './types'

const PAGE_BREAK = '__page_break__'
const REQUESTER_ROLES = new Set(['requester_name', 'requester_email'])

export interface IntakeFormFillProps {
  /** Id of the form schema to render (must carry an intakeTargetId). */
  formId: string
  /** Origin channel written onto the created record. */
  channel: string
  /** Resolved requester (internal self-service = the logged-in user). */
  requester?: IntakeBuildContext['requester']
  /** Called with the created record id after a successful submit. */
  onCreated?: (id: string) => void
}

export function IntakeFormFill({ formId, channel, requester, onCreated }: IntakeFormFillProps) {
  const { t } = useTranslation()
  const { data: schema, isLoading } = useFormSchema(formId)
  const dispatch = useIntakeSubmit()

  const [values, setValues] = useState<Record<string, unknown>>({})
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [creating, setCreating] = useState(false)
  const [createdId, setCreatedId] = useState<string | null>(null)
  const [submitError, setSubmitError] = useState<string | null>(null)

  // Fields to render: drop page breaks and (when a requester is known) the
  // requester roles — those are auto-filled from the profile.
  const visibleFields = useMemo(() => {
    const fields = (schema?.fields ?? []) as FormField[]
    return fields.filter(
      (f) =>
        f.label !== PAGE_BREAK && !(requester && f.role && REQUESTER_ROLES.has(f.role)),
    )
  }, [schema, requester])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-10 text-muted-foreground">
        <Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" />
      </div>
    )
  }
  if (!schema || !schema.intakeTargetId || !getIntakeTarget(schema.intakeTargetId)) {
    return <p className="py-6 text-center text-sm text-muted-foreground">{t('intake.fill.unavailable')}</p>
  }

  if (createdId) {
    return (
      <div className="flex flex-col items-center gap-3 py-8 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-success-light">
          <Check className="h-6 w-6 text-success" aria-hidden="true" />
        </div>
        <p className="text-base font-semibold text-foreground">{t('intake.fill.thanks')}</p>
        <p className="text-sm text-muted-foreground">{t('intake.fill.reference', { id: createdId })}</p>
      </div>
    )
  }

  const setVal = (id: string, v: unknown) => {
    setValues((prev) => ({ ...prev, [id]: v }))
    setErrors((prev) => {
      if (!prev[id]) return prev
      const next = { ...prev }
      delete next[id]
      return next
    })
  }

  const submit = async () => {
    const next: Record<string, string> = {}
    for (const field of visibleFields) {
      const raw = values[field.id]
      const missing =
        field.type === 'consent' || field.type === 'checkbox'
          ? raw !== true
          : String(raw ?? '').trim() === ''
      if (field.required && missing) next[field.id] = t('intake.fill.required')
    }
    setErrors(next)
    if (Object.keys(next).length > 0) return

    setSubmitError(null)
    setCreating(true)
    try {
      const res = await dispatch({
        targetId: schema.intakeTargetId as string,
        fields: schema.fields as FormField[],
        answers: values,
        context: { channel, requester },
      })
      setCreatedId(res.id)
      onCreated?.(res.id)
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err))
    } finally {
      setCreating(false)
    }
  }

  const inputCls = (id: string) =>
    `w-full rounded-lg border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring ${
      errors[id] ? 'border-destructive' : 'border-border'
    }`

  return (
    <div className="space-y-4">
      {requester?.name && (
        <p className="rounded-lg bg-info-light px-3 py-2 text-xs text-info">
          {t('intake.fill.reportingAs', { name: requester.name })}
        </p>
      )}
      {visibleFields.map((field) => {
        const val = values[field.id]
        return (
          <div key={field.id} className="space-y-1.5">
            {field.type !== 'consent' && field.type !== 'checkbox' && (
              <label className="text-sm font-medium text-foreground">
                {field.label}
                {field.required && <span className="ml-0.5 text-destructive">*</span>}
              </label>
            )}
            {(field.type === 'text' || field.type === 'email' || field.type === 'number' || field.type === 'date') && (
              <input
                type={field.type === 'text' ? 'text' : field.type}
                value={typeof val === 'string' ? val : ''}
                onChange={(e) => setVal(field.id, e.target.value)}
                placeholder={field.placeholder}
                className={inputCls(field.id)}
              />
            )}
            {field.type === 'textarea' && (
              <textarea
                rows={3}
                value={typeof val === 'string' ? val : ''}
                onChange={(e) => setVal(field.id, e.target.value)}
                placeholder={field.placeholder}
                className={`resize-none ${inputCls(field.id)}`}
              />
            )}
            {field.type === 'select' && (
              <select
                value={typeof val === 'string' ? val : ''}
                onChange={(e) => setVal(field.id, e.target.value)}
                className={inputCls(field.id)}
              >
                <option value="">{t('intake.fill.selectPlaceholder')}</option>
                {field.options?.map((o) => (
                  <option key={o} value={o}>{o}</option>
                ))}
              </select>
            )}
            {field.type === 'radio' && (
              <div className="space-y-1.5">
                {field.options?.map((o) => (
                  <label key={o} className="flex items-center gap-2 text-sm text-foreground">
                    <input type="radio" name={field.id} checked={val === o} onChange={() => setVal(field.id, o)} className="h-4 w-4" />
                    {o}
                  </label>
                ))}
              </div>
            )}
            {(field.type === 'checkbox' || field.type === 'consent') && (
              <label className="flex items-start gap-2 text-sm text-foreground">
                <input
                  type="checkbox"
                  checked={val === true}
                  onChange={(e) => setVal(field.id, e.target.checked)}
                  className={`mt-0.5 h-4 w-4 ${errors[field.id] ? 'ring-2 ring-destructive' : ''}`}
                />
                <span>
                  {field.type === 'consent' ? field.consentText || field.label : field.label}
                  {field.required && <span className="ml-0.5 text-destructive">*</span>}
                </span>
              </label>
            )}
            {errors[field.id] && <p className="text-xs text-destructive">{errors[field.id]}</p>}
          </div>
        )
      })}
      {submitError && (
        <div className="rounded-lg border border-destructive/40 bg-error-light px-3 py-2 text-sm text-error">
          {t('intake.fill.error', { message: submitError })}
        </div>
      )}
      <button
        onClick={submit}
        disabled={creating}
        className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-button-primary-hover disabled:cursor-not-allowed disabled:opacity-60"
      >
        {creating && <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />}
        {creating ? t('intake.fill.submitting') : t('intake.fill.submit')}
      </button>
    </div>
  )
}
