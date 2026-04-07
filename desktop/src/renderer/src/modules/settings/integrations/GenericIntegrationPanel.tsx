/**
 * GenericIntegrationPanel — reusable config panel for simple integrations.
 *
 * Renders fields from IntegrationDefinition.fields[] automatically.
 * Groups by section, reads/writes values from Zustand store.
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Loader2, RefreshCw, Plug, Unlink, Zap, Eye, EyeOff } from 'lucide-react'
import { toast } from 'sonner'
import type { IntegrationDefinition, FieldDefinition } from './integration-registry'
import { useIntegrationStore } from '@/stores/integrations'

interface GenericIntegrationPanelProps {
  definition: IntegrationDefinition
  onBack: () => void
}

const STATUS_BADGE: Record<string, { labelKey: string; cls: string }> = {
  connected: { labelKey: 'settings.integrations.panel.status.connected', cls: 'bg-success-light text-success' },
  disconnected: { labelKey: 'settings.integrations.panel.status.disconnected', cls: 'bg-secondary text-muted-foreground' },
  syncing: { labelKey: 'settings.integrations.panel.status.syncing', cls: 'bg-info-light text-info' },
  error: { labelKey: 'settings.integrations.panel.status.error', cls: 'bg-error-light text-destructive' },
}

export function GenericIntegrationPanel({ definition, onBack }: GenericIntegrationPanelProps) {
  const { t } = useTranslation()
  const store = useIntegrationStore()
  const status = store.getStatus(definition.id)
  const storedValues = store.getFieldValues(definition.id)
  const integration = store.integrations[definition.id]
  const fields = useMemo(() => definition.fields ?? [], [definition.fields])

  const [localValues, setLocalValues] = useState<Record<string, string | boolean>>(() => {
    const defaults: Record<string, string | boolean> = {}
    for (const f of fields) {
      defaults[f.id] = storedValues[f.id] ?? f.defaultValue ?? (f.type === 'switch' ? false : '')
    }
    return defaults
  })

  const [visiblePasswords, setVisiblePasswords] = useState<Set<string>>(new Set())
  const [testing, setTesting] = useState(false)

  const Icon = definition.icon
  const badge = STATUS_BADGE[status] ?? STATUS_BADGE.disconnected

  // Group fields by section
  const sections = useMemo(() => {
    const map = new Map<string, FieldDefinition[]>()
    for (const f of fields) {
      const key = f.section ?? ''
      if (!map.has(key)) map.set(key, [])
      map.get(key)!.push(f)
    }
    return map
  }, [fields])

  const updateField = (id: string, value: string | boolean) => {
    setLocalValues((prev) => ({ ...prev, [id]: value }))
  }

  const handleSave = () => {
    store.setFieldValues(definition.id, localValues)
    toast.success('Einstellungen gespeichert')
  }

  const handleConnect = () => {
    store.setFieldValues(definition.id, localValues)
    if (definition.authMethod === 'oauth2') {
      // Mock OAuth2 flow
      store.setStatus(definition.id, 'syncing' as never)
      setTimeout(() => store.connect(definition.id), 1500)
      toast.success(`${definition.name} wird verbunden...`)
    } else {
      store.connect(definition.id)
      toast.success(`${definition.name} verbunden`)
    }
  }

  const handleDisconnect = () => {
    store.disconnect(definition.id)
    toast.success(`${definition.name} getrennt`)
  }

  const handleTest = () => {
    setTesting(true)
    setTimeout(() => {
      setTesting(false)
      toast.success('Verbindung erfolgreich')
    }, 1000)
  }

  const handleSync = () => {
    store.triggerSync(definition.id)
    toast.success('Synchronisation gestartet')
  }

  const togglePasswordVisibility = (fieldId: string) => {
    setVisiblePasswords((prev) => {
      const next = new Set(prev)
      if (next.has(fieldId)) next.delete(fieldId)
      else next.add(fieldId)
      return next
    })
  }

  const renderField = (field: FieldDefinition) => {
    const value = localValues[field.id]

    switch (field.type) {
      case 'text':
      case 'url':
        return (
          <div key={field.id} className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t(field.label)}</label>
            <input
              type={field.type === 'url' ? 'url' : 'text'}
              value={(value as string) ?? ''}
              onChange={(e) => updateField(field.id, e.target.value)}
              placeholder={field.placeholder}
              className={`w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground ${field.monospace ? 'font-mono text-xs' : ''}`}
            />
            {field.helpText && <p className="text-[11px] text-muted-foreground">{t(field.helpText)}</p>}
          </div>
        )

      case 'password': {
        const isVisible = visiblePasswords.has(field.id)
        return (
          <div key={field.id} className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t(field.label)}</label>
            <div className="relative">
              <input
                type={isVisible ? 'text' : 'password'}
                value={(value as string) ?? ''}
                onChange={(e) => updateField(field.id, e.target.value)}
                placeholder={field.placeholder}
                className={`w-full rounded-lg border border-border bg-card px-3 py-2 pr-10 text-sm text-foreground placeholder:text-muted-foreground ${field.monospace ? 'font-mono text-xs' : ''}`}
              />
              <button
                type="button"
                onClick={() => togglePasswordVisibility(field.id)}
                className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground"
              >
                {isVisible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
            {field.helpText && <p className="text-[11px] text-muted-foreground">{t(field.helpText)}</p>}
          </div>
        )
      }

      case 'select':
        return (
          <div key={field.id} className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t(field.label)}</label>
            <select
              value={(value as string) ?? ''}
              onChange={(e) => updateField(field.id, e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground"
            >
              {field.options?.map((opt) => (
                <option key={opt.value} value={opt.value}>{t(opt.label)}</option>
              ))}
            </select>
            {field.helpText && <p className="text-[11px] text-muted-foreground">{t(field.helpText)}</p>}
          </div>
        )

      case 'switch':
        return (
          <div key={field.id} className="flex items-center justify-between py-1">
            <div>
              <span className="text-sm font-medium text-foreground">{t(field.label)}</span>
              {field.helpText && <p className="text-[11px] text-muted-foreground">{t(field.helpText)}</p>}
            </div>
            <button
              onClick={() => updateField(field.id, !value)}
              className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                value ? 'bg-primary' : 'bg-muted'
              }`}
            >
              <span
                className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
                  value ? 'translate-x-4.5' : 'translate-x-0.5'
                }`}
              />
            </button>
          </div>
        )

      case 'readonly':
        return (
          <div key={field.id} className="space-y-1.5">
            <label className="text-sm font-medium text-foreground">{t(field.label)}</label>
            <div className="rounded-lg border border-border bg-muted/50 px-3 py-2 text-sm text-muted-foreground">
              {(value as string) || field.defaultValue || '—'}
            </div>
          </div>
        )

      default:
        return null
    }
  }

  return (
    <div className="max-w-2xl">
      {/* Back button */}
      <button
        onClick={onBack}
        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        {t('settings.integrations.backToIntegrations')}
      </button>

      {/* Header */}
      <div className="flex items-start gap-4 mb-6">
        <div className={`flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-secondary`}>
          <Icon className={`h-6 w-6 ${definition.iconColor}`} />
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-semibold text-foreground">{definition.name}</h2>
            <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-medium ${badge.cls}`}>
              {t(badge.labelKey)}
            </span>
          </div>
          <p className="text-sm text-muted-foreground mt-0.5">{t(definition.description)}</p>
        </div>
      </div>

      {/* Connected info */}
      {status === 'connected' && integration?.connectedAt && (
        <div className="rounded-lg border border-success/30 bg-success-light px-4 py-2.5 mb-6 text-sm text-success">
          {t('settings.integrations.connectedSince', { date: new Date(integration.connectedAt).toLocaleDateString('de-DE') })}
          {integration.lastSync && (
            <span className="ml-3 text-success/70">
              {t('settings.integrations.lastSync', { date: new Date(integration.lastSync).toLocaleString('de-DE') })}
            </span>
          )}
        </div>
      )}

      {/* Fields grouped by section */}
      {Array.from(sections.entries()).map(([sectionName, sectionFields]) => (
        <div key={sectionName} className="mb-6">
          {sectionName && (
            <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">
              {t(sectionName)}
            </h3>
          )}
          <div className="space-y-4">
            {sectionFields.map(renderField)}
          </div>
        </div>
      ))}

      {/* Actions */}
      <div className="flex items-center gap-2 border-t border-border pt-4 mt-6">
        {status === 'connected' ? (
          <>
            <button
              onClick={handleDisconnect}
              className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-muted transition-colors"
            >
              <Unlink className="h-4 w-4" />
              {t('settings.integrations.panel.disconnect')}
            </button>
            <button
              onClick={handleSync}
              disabled={status === 'syncing'}
              className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-muted transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`h-4 w-4 ${status === 'syncing' ? 'animate-spin' : ''}`} />
              {t('settings.integrations.panel.sync')}
            </button>
            <button
              onClick={handleSave}
              className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              {t('settings.integrations.panel.save')}
            </button>
          </>
        ) : (
          <>
            <button
              onClick={handleConnect}
              className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              <Plug className="h-4 w-4" />
              {t('settings.integrations.panel.connect')}
            </button>
            <button
              onClick={handleSave}
              className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-muted transition-colors"
            >
              {t('settings.integrations.panel.save')}
            </button>
          </>
        )}
        <button
          onClick={handleTest}
          disabled={testing}
          className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-muted transition-colors disabled:opacity-50 ml-auto"
        >
          {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />}
          {t('settings.integrations.panel.testConnection')}
        </button>
      </div>
    </div>
  )
}
