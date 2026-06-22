import { useState, useMemo } from 'react'
import { sanitizeHtml } from '@/lib/sanitize'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import {
  FileText,
  CheckCircle2,
  MessageSquare,
  CalendarCheck,
  UserPlus,
  CreditCard,
  Search,
  Plus,
  Pencil,
  Trash2,
  ArrowLeft,
  Save,
  User as UserIcon,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import {
  useMailTemplatesStore,
  extractPlaceholders,
  fillPlaceholders,
  type UserTemplate,
} from '@/stores/mailTemplates'

// ---------------------------------------------------------------------------
// Template definitions
// ---------------------------------------------------------------------------

export interface EmailTemplate {
  id: string
  name: string
  nameKey: string
  category: 'vertrieb' | 'kommunikation' | 'finanzen'
  icon: LucideIcon
  subject: string
  subjectKey: string
  /** HTML body with {{placeholder}} variables */
  body: string
  placeholders: string[]
}

const templates: EmailTemplate[] = [
  {
    id: 'angebot',
    name: 'Angebot',
    nameKey: 'mails.templates.name.angebot',
    category: 'vertrieb',
    icon: FileText,
    subject: 'Angebot für {{firma}}',
    subjectKey: 'mails.templates.subject.angebot',
    placeholders: ['anrede', 'name', 'firma', 'datum'],
    body: `<p>{{anrede}} {{name}},</p>
<p>vielen Dank für Ihr Interesse an unseren Leistungen. Gerne unterbreiten wir Ihnen hiermit unser Angebot.</p>
<p><strong>Leistungsumfang:</strong></p>
<ul>
<li>Leistung 1 — CHF 0.00</li>
<li>Leistung 2 — CHF 0.00</li>
<li>Leistung 3 — CHF 0.00</li>
</ul>
<p>Das Angebot ist gültig bis zum {{datum}}. Bei Fragen stehen wir Ihnen jederzeit zur Verfügung.</p>`,
  },
  {
    id: 'auftragsbestaetigung',
    name: 'Auftragsbestaetigung',
    nameKey: 'mails.templates.name.auftragsbestaetigung',
    category: 'vertrieb',
    icon: CheckCircle2,
    subject: 'Auftragsbestaetigung — {{firma}}',
    subjectKey: 'mails.templates.subject.auftragsbestaetigung',
    placeholders: ['anrede', 'name', 'firma', 'datum'],
    body: `<p>{{anrede}} {{name}},</p>
<p>herzlichen Dank für Ihren Auftrag! Wir bestätigen hiermit den Eingang und die Annahme.</p>
<p><strong>Auftragsdetails:</strong></p>
<ul>
<li><strong>Kunde:</strong> {{firma}}</li>
<li><strong>Datum:</strong> {{datum}}</li>
<li><strong>Auftragsnummer:</strong> AUF-2026-XXX</li>
</ul>
<p>Wir werden Sie über den Fortschritt auf dem Laufenden halten. Bei Rückfragen erreichen Sie uns jederzeit.</p>`,
  },
  {
    id: 'follow-up',
    name: 'Follow-Up',
    nameKey: 'mails.templates.name.followUp',
    category: 'kommunikation',
    icon: MessageSquare,
    subject: 'Nachfassen: Unser Gespräch',
    subjectKey: 'mails.templates.subject.followUp',
    placeholders: ['anrede', 'name', 'datum'],
    body: `<p>{{anrede}} {{name}},</p>
<p>vielen Dank für unser Gespräch am {{datum}}. Ich möchte kurz die besprochenen Punkte zusammenfassen:</p>
<ol>
<li>Punkt 1</li>
<li>Punkt 2</li>
<li>Punkt 3</li>
</ol>
<p><strong>Nächste Schritte:</strong></p>
<ul>
<li>Schritt 1 — bis TT.MM.JJJJ</li>
<li>Schritt 2 — bis TT.MM.JJJJ</li>
</ul>
<p>Bitte lassen Sie mich wissen, falls ich etwas übersehen habe oder Anpassungen nötig sind.</p>`,
  },
  {
    id: 'terminbestaetigung',
    name: 'Terminbestaetigung',
    nameKey: 'mails.templates.name.terminbestaetigung',
    category: 'kommunikation',
    icon: CalendarCheck,
    subject: 'Terminbestaetigung — {{datum}}',
    subjectKey: 'mails.templates.subject.terminbestaetigung',
    placeholders: ['anrede', 'name', 'datum'],
    body: `<p>{{anrede}} {{name}},</p>
<p>hiermit bestätigen wir Ihren Termin:</p>
<table>
<tr><td><strong>Datum:</strong></td><td>{{datum}}</td></tr>
<tr><td><strong>Uhrzeit:</strong></td><td>00:00 Uhr</td></tr>
<tr><td><strong>Ort:</strong></td><td>Unser Büro / Online (Link folgt)</td></tr>
</table>
<p>Bitte geben Sie uns Bescheid, falls Sie den Termin nicht wahrnehmen können. Wir freuen uns auf das Gespräch!</p>`,
  },
  {
    id: 'willkommen',
    name: 'Willkommen',
    nameKey: 'mails.templates.name.willkommen',
    category: 'kommunikation',
    icon: UserPlus,
    subject: 'Willkommen bei {{firma}}!',
    subjectKey: 'mails.templates.subject.willkommen',
    placeholders: ['anrede', 'name', 'firma'],
    body: `<p>{{anrede}} {{name}},</p>
<p>herzlich willkommen bei {{firma}}! Wir freuen uns, Sie als neuen Kunden / Partner begrüßen zu dürfen.</p>
<p><strong>Ihre nächsten Schritte:</strong></p>
<ol>
<li>Zugang einrichten unter <em>[Link]</em></li>
<li>Profil vervollständigen</li>
<li>Erste Schritte in der Anleitung lesen</li>
</ol>
<p>Bei Fragen steht Ihnen Ihr persönlicher Ansprechpartner jederzeit zur Verfügung.</p>`,
  },
  {
    id: 'zahlungserinnerung',
    name: 'Zahlungserinnerung',
    nameKey: 'mails.templates.name.zahlungserinnerung',
    category: 'finanzen',
    icon: CreditCard,
    subject: 'Freundliche Zahlungserinnerung — Rechnung RE-XXXX',
    subjectKey: 'mails.templates.subject.zahlungserinnerung',
    placeholders: ['anrede', 'name', 'firma', 'datum'],
    body: `<p>{{anrede}} {{name}},</p>
<p>bei der Prüfung unserer offenen Posten ist uns aufgefallen, dass die nachfolgende Rechnung noch nicht beglichen wurde:</p>
<table>
<tr><td><strong>Rechnungsnr.:</strong></td><td>RE-XXXX</td></tr>
<tr><td><strong>Rechnungsdatum:</strong></td><td>{{datum}}</td></tr>
<tr><td><strong>Betrag:</strong></td><td>CHF 0.00</td></tr>
</table>
<p>Wir bitten Sie, die Zahlung innerhalb der nächsten 10 Tage vorzunehmen. Sollte die Zahlung bereits erfolgt sein, betrachten Sie diese E-Mail bitte als gegenstandslos.</p>`,
  },
]

// ---------------------------------------------------------------------------
// Category labels
// ---------------------------------------------------------------------------

const categoryLabelKeys: Record<string, string> = {
  vertrieb: 'mails.templates.categories.vertrieb',
  kommunikation: 'mails.templates.categories.kommunikation',
  finanzen: 'mails.templates.categories.finanzen',
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface EmailTemplateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSelect: (template: { subject: string; body: string }) => void
}

interface DisplayTemplate {
  id: string
  name: string
  category: string
  subject: string
  body: string
  placeholders: string[]
  builtin: boolean
  icon: LucideIcon
}

type Mode = 'browse' | 'editor' | 'fill'
type EditorForm = { id?: string; name: string; category: UserTemplate['category']; subject: string; body: string }

const CATEGORIES: UserTemplate['category'][] = ['vertrieb', 'kommunikation', 'finanzen']

// ── Right-panel: preview ─────────────────────────────────────────
function TemplatePreview({ template }: { template: DisplayTemplate }) {
  const { t } = useTranslation()
  return (
    <div className="space-y-4">
      <div>
        <p className="text-xs text-muted-foreground mb-1">{t('mails.compose.subject')}</p>
        <p className="text-sm font-medium text-foreground">{template.subject}</p>
      </div>
      <div>
        <p className="text-xs text-muted-foreground mb-2">{t('mails.templates.preview')}</p>
        <div
          className="rounded-lg border border-border bg-background p-4 prose prose-sm dark:prose-invert max-w-none text-sm"
          dangerouslySetInnerHTML={{ __html: sanitizeHtml(template.body) }}
        />
      </div>
      {template.placeholders.length > 0 && (
        <div>
          <p className="text-xs text-muted-foreground mb-1.5">{t('mails.templates.placeholders')}</p>
          <div className="flex flex-wrap gap-1.5">
            {template.placeholders.map((p) => (
              <span key={p} className="rounded-full bg-amber-100 dark:bg-amber-900/30 px-2.5 py-0.5 text-xs font-mono text-amber-700 dark:text-amber-300">
                {`{{${p}}}`}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// ── Right-panel: create / edit a user template ───────────────────
function TemplateEditor({ form, setForm }: { form: EditorForm; setForm: (f: EditorForm) => void }) {
  const { t } = useTranslation()
  const inputCls = 'w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-focus-ring'
  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">{t('mails.templates.fieldName', { defaultValue: 'Name' })}</label>
        <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className={inputCls} />
      </div>
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">{t('mails.templates.fieldCategory', { defaultValue: 'Kategorie' })}</label>
        <select value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value as UserTemplate['category'] })} className={inputCls}>
          {CATEGORIES.map((c) => (
            <option key={c} value={c}>{t(categoryLabelKeys[c])}</option>
          ))}
        </select>
      </div>
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">{t('mails.compose.subject')}</label>
        <input value={form.subject} onChange={(e) => setForm({ ...form, subject: e.target.value })} className={inputCls} />
      </div>
      <div className="space-y-1">
        <label className="text-xs text-muted-foreground">{t('mails.templates.fieldBody', { defaultValue: 'Inhalt (HTML)' })}</label>
        <textarea value={form.body} onChange={(e) => setForm({ ...form, body: e.target.value })} rows={9} className={`${inputCls} resize-none font-mono text-xs`} />
        <p className="text-[10px] text-muted-foreground">{t('mails.templates.placeholderHint', { defaultValue: 'Doppelte geschweifte Klammern markieren Platzhalter — sie werden beim Einfügen abgefragt.' })}</p>
      </div>
    </div>
  )
}

// ── Right-panel: fill placeholders before inserting ──────────────
function PlaceholderFill({
  template,
  values,
  setValues,
}: {
  template: DisplayTemplate
  values: Record<string, string>
  setValues: (v: Record<string, string>) => void
}) {
  const { t } = useTranslation()
  const preview = fillPlaceholders(template.body, values)
  return (
    <div className="space-y-4">
      <div>
        <p className="text-xs text-muted-foreground mb-1.5">{t('mails.templates.fillTitle', { defaultValue: 'Platzhalter ausfüllen' })}</p>
        <div className="space-y-2">
          {template.placeholders.map((p) => (
            <div key={p} className="flex items-center gap-2">
              <span className="w-24 shrink-0 font-mono text-xs text-amber-700 dark:text-amber-300">{`{{${p}}}`}</span>
              <input
                value={values[p] ?? ''}
                onChange={(e) => setValues({ ...values, [p]: e.target.value })}
                placeholder={p}
                className="flex-1 rounded-md border border-border bg-background px-2.5 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
          ))}
        </div>
      </div>
      <div>
        <p className="text-xs text-muted-foreground mb-2">{t('mails.templates.preview')}</p>
        <div className="rounded-lg border border-border bg-background p-4 prose prose-sm dark:prose-invert max-w-none text-sm" dangerouslySetInnerHTML={{ __html: sanitizeHtml(preview) }} />
      </div>
    </div>
  )
}

export function EmailTemplateDialog({ open, onOpenChange, onSelect }: EmailTemplateDialogProps) {
  const { t } = useTranslation()
  const userTemplates = useMailTemplatesStore((s) => s.userTemplates)
  const addTemplate = useMailTemplatesStore((s) => s.addTemplate)
  const updateTemplate = useMailTemplatesStore((s) => s.updateTemplate)
  const removeTemplate = useMailTemplatesStore((s) => s.removeTemplate)

  const [search, setSearch] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [mode, setMode] = useState<Mode>('browse')
  const [form, setForm] = useState<EditorForm>({ name: '', category: 'kommunikation', subject: '', body: '' })
  const [fillValues, setFillValues] = useState<Record<string, string>>({})

  // Merge built-in + user templates into a single display list
  const allTemplates: DisplayTemplate[] = useMemo(() => {
    const builtins: DisplayTemplate[] = templates.map((tpl) => ({
      id: tpl.id,
      name: t(tpl.nameKey),
      category: tpl.category,
      subject: t(tpl.subjectKey),
      body: tpl.body,
      placeholders: tpl.placeholders,
      builtin: true,
      icon: tpl.icon,
    }))
    const users: DisplayTemplate[] = userTemplates.map((tpl) => ({
      id: tpl.id,
      name: tpl.name,
      category: tpl.category,
      subject: tpl.subject,
      body: tpl.body,
      placeholders: extractPlaceholders(tpl.subject, tpl.body),
      builtin: false,
      icon: UserIcon,
    }))
    return [...users, ...builtins]
  }, [t, userTemplates])

  const filtered = allTemplates.filter(
    (tpl) => !search || tpl.name.toLowerCase().includes(search.toLowerCase()) || tpl.category.includes(search.toLowerCase()),
  )
  const selected = allTemplates.find((tpl) => tpl.id === selectedId) ?? null

  const grouped = filtered.reduce((acc, tpl) => {
    ;(acc[tpl.category] ??= []).push(tpl)
    return acc
  }, {} as Record<string, DisplayTemplate[]>)

  const resetAndClose = () => {
    onOpenChange(false)
    setSelectedId(null)
    setSearch('')
    setMode('browse')
    setFillValues({})
  }

  const startInsert = () => {
    if (!selected) return
    if (selected.placeholders.length > 0) {
      setFillValues(Object.fromEntries(selected.placeholders.map((p) => [p, ''])))
      setMode('fill')
    } else {
      onSelect({ subject: selected.subject, body: selected.body })
      resetAndClose()
    }
  }

  const confirmFill = () => {
    if (!selected) return
    onSelect({
      subject: fillPlaceholders(selected.subject, fillValues),
      body: fillPlaceholders(selected.body, fillValues),
    })
    resetAndClose()
  }

  const startNew = () => {
    setForm({ name: '', category: 'kommunikation', subject: '', body: '' })
    setMode('editor')
  }
  const startEdit = (tpl: DisplayTemplate) => {
    setForm({ id: tpl.id, name: tpl.name, category: tpl.category as UserTemplate['category'], subject: tpl.subject, body: tpl.body })
    setMode('editor')
  }
  const saveForm = () => {
    if (!form.name.trim()) return
    if (form.id) {
      updateTemplate(form.id, { name: form.name, category: form.category, subject: form.subject, body: form.body })
    } else {
      const created = addTemplate({ name: form.name, category: form.category, subject: form.subject, body: form.body })
      setSelectedId(created.id)
    }
    setMode('browse')
  }
  const deleteUser = (id: string) => {
    removeTemplate(id)
    if (selectedId === id) setSelectedId(null)
  }

  return (
    <Dialog open={open} onOpenChange={(o) => (o ? onOpenChange(o) : resetAndClose())}>
      <DialogContent className="max-w-3xl max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>{t('mails.templates.title')}</DialogTitle>
        </DialogHeader>

        <div className="flex flex-1 overflow-hidden gap-4 min-h-0">
          {/* Left: template list */}
          <div className="w-64 shrink-0 flex flex-col border-r border-border pr-4 overflow-hidden">
            <button
              onClick={startNew}
              className="mb-3 flex w-full items-center justify-center gap-1.5 rounded-md border border-dashed border-border px-2.5 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
            >
              <Plus className="h-4 w-4" /> {t('mails.templates.new', { defaultValue: 'Neue Vorlage' })}
            </button>
            <div className="relative mb-3">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t('mails.templates.searchPlaceholder')}
                className="w-full rounded-md border border-border bg-background pl-8 pr-3 py-1.5 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="flex-1 overflow-y-auto space-y-4">
              {CATEGORIES.filter((c) => grouped[c]?.length).map((category) => (
                <div key={category}>
                  <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                    {t(categoryLabelKeys[category])}
                  </p>
                  <div className="space-y-0.5">
                    {grouped[category].map((tpl) => {
                      const Icon = tpl.icon
                      return (
                        <div
                          key={tpl.id}
                          className={`group flex items-center gap-1 rounded-md pr-1 transition-colors ${
                            selectedId === tpl.id ? 'bg-primary-light' : 'hover:bg-secondary'
                          }`}
                        >
                          <button
                            onClick={() => { setSelectedId(tpl.id); setMode('browse') }}
                            className={`flex min-w-0 flex-1 items-center gap-2 px-2.5 py-2 text-sm ${
                              selectedId === tpl.id ? 'text-primary font-medium' : 'text-foreground'
                            }`}
                          >
                            <Icon className="h-4 w-4 shrink-0" />
                            <span className="truncate">{tpl.name}</span>
                          </button>
                          {!tpl.builtin && (
                            <div className="flex shrink-0 items-center opacity-0 group-hover:opacity-100 transition-opacity">
                              <button onClick={() => startEdit(tpl)} className="rounded p-1 text-muted-foreground hover:text-foreground" aria-label={t('common.edit', { defaultValue: 'Bearbeiten' })}>
                                <Pencil className="h-3.5 w-3.5" />
                              </button>
                              <button onClick={() => deleteUser(tpl.id)} className="rounded p-1 text-muted-foreground hover:text-error" aria-label={t('common.delete')}>
                                <Trash2 className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          )}
                        </div>
                      )
                    })}
                  </div>
                </div>
              ))}
              {filtered.length === 0 && (
                <p className="text-sm text-muted-foreground text-center py-4">{t('mails.templates.noResults')}</p>
              )}
            </div>
          </div>

          {/* Right: preview / editor / fill */}
          <div className="flex-1 overflow-y-auto">
            {mode === 'editor' ? (
              <TemplateEditor form={form} setForm={setForm} />
            ) : mode === 'fill' && selected ? (
              <PlaceholderFill template={selected} values={fillValues} setValues={setFillValues} />
            ) : selected ? (
              <TemplatePreview template={selected} />
            ) : (
              <div className="flex h-full items-center justify-center text-muted-foreground">
                <div className="text-center">
                  <FileText className="h-10 w-10 mx-auto mb-2 opacity-30" />
                  <p className="text-sm">{t('mails.templates.selectFromList')}</p>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-2 pt-3 border-t border-border">
          {mode === 'editor' ? (
            <>
              <Button variant="outline" onClick={() => setMode('browse')}>
                <ArrowLeft className="h-4 w-4 mr-1.5" />
                {t('common.cancel')}
              </Button>
              <Button onClick={saveForm} disabled={!form.name.trim()}>
                <Save className="h-4 w-4 mr-1.5" />
                {t('common.save', { defaultValue: 'Speichern' })}
              </Button>
            </>
          ) : mode === 'fill' ? (
            <>
              <Button variant="outline" onClick={() => setMode('browse')}>
                <ArrowLeft className="h-4 w-4 mr-1.5" />
                {t('common.back', { defaultValue: 'Zurück' })}
              </Button>
              <Button onClick={confirmFill}>
                <FileText className="h-4 w-4 mr-1.5" />
                {t('mails.compose.insertTemplate')}
              </Button>
            </>
          ) : (
            <>
              <Button variant="outline" onClick={resetAndClose}>{t('common.cancel')}</Button>
              <Button onClick={startInsert} disabled={!selected}>
                <FileText className="h-4 w-4 mr-1.5" />
                {t('mails.compose.insertTemplate')}
              </Button>
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
