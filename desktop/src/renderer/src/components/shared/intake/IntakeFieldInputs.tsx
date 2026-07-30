/**
 * IntakeFieldInputs — renders a list of intake form fields as controlled inputs.
 *
 * Extracted from IntakeFormFill so the same field rendering is reused by the
 * agent new-ticket dialog (which embeds the bound agent form's fields alongside
 * its own professional tools) and by the self-service / external fill surfaces.
 * Pure presentation: the caller owns values, errors and the change handler.
 */
import { useTranslation } from 'react-i18next'
import type { FormField } from '@/api/formulare-types'

export interface IntakeFieldInputsProps {
  fields: FormField[]
  values: Record<string, unknown>
  errors?: Record<string, string>
  onChange: (id: string, v: unknown) => void
}

export function IntakeFieldInputs({ fields, values, errors = {}, onChange }: IntakeFieldInputsProps) {
  const { t } = useTranslation()

  const inputCls = (id: string) =>
    `w-full rounded-lg border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring ${
      errors[id] ? 'border-destructive' : 'border-border'
    }`

  return (
    <>
      {fields.map((field) => {
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
                onChange={(e) => onChange(field.id, e.target.value)}
                placeholder={field.placeholder}
                className={inputCls(field.id)}
              />
            )}
            {field.type === 'textarea' && (
              <textarea
                rows={3}
                value={typeof val === 'string' ? val : ''}
                onChange={(e) => onChange(field.id, e.target.value)}
                placeholder={field.placeholder}
                className={`resize-none ${inputCls(field.id)}`}
              />
            )}
            {field.type === 'select' && (
              <select
                value={typeof val === 'string' ? val : ''}
                onChange={(e) => onChange(field.id, e.target.value)}
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
                    <input type="radio" name={field.id} checked={val === o} onChange={() => onChange(field.id, o)} className="h-4 w-4" />
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
                  onChange={(e) => onChange(field.id, e.target.checked)}
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
    </>
  )
}
