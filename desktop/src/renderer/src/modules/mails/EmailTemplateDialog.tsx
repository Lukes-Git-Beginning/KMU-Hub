import { useState } from 'react'
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
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

// ---------------------------------------------------------------------------
// Template definitions
// ---------------------------------------------------------------------------

export interface EmailTemplate {
  id: string
  name: string
  category: 'vertrieb' | 'kommunikation' | 'finanzen'
  icon: LucideIcon
  subject: string
  /** HTML body with {{placeholder}} variables */
  body: string
  placeholders: string[]
}

const templates: EmailTemplate[] = [
  {
    id: 'angebot',
    name: 'Angebot',
    category: 'vertrieb',
    icon: FileText,
    subject: 'Angebot für {{firma}}',
    placeholders: ['anrede', 'name', 'firma', 'datum'],
    body: `<p>{{anrede}} {{name}},</p>
<p>vielen Dank für Ihr Interesse an unseren Leistungen. Gerne unterbreiten wir Ihnen hiermit unser Angebot.</p>
<p><strong>Leistungsumfang:</strong></p>
<ul>
<li>Leistung 1 — CHF 0.00</li>
<li>Leistung 2 — CHF 0.00</li>
<li>Leistung 3 — CHF 0.00</li>
</ul>
<p>Das Angebot ist gültig bis zum {{datum}}. Bei Fragen stehen wir Ihnen jederzeit zur Verfuegung.</p>`,
  },
  {
    id: 'auftragsbestaetigung',
    name: 'Auftragsbestaetigung',
    category: 'vertrieb',
    icon: CheckCircle2,
    subject: 'Auftragsbestaetigung — {{firma}}',
    placeholders: ['anrede', 'name', 'firma', 'datum'],
    body: `<p>{{anrede}} {{name}},</p>
<p>herzlichen Dank für Ihren Auftrag! Wir bestätigen hiermit den Eingang und die Annahme.</p>
<p><strong>Auftragsdetails:</strong></p>
<ul>
<li><strong>Kunde:</strong> {{firma}}</li>
<li><strong>Datum:</strong> {{datum}}</li>
<li><strong>Auftragsnummer:</strong> AUF-2026-XXX</li>
</ul>
<p>Wir werden Sie ueber den Fortschritt auf dem Laufenden halten. Bei Rueckfragen erreichen Sie uns jederzeit.</p>`,
  },
  {
    id: 'follow-up',
    name: 'Follow-Up',
    category: 'kommunikation',
    icon: MessageSquare,
    subject: 'Nachfassen: Unser Gespräch',
    placeholders: ['anrede', 'name', 'datum'],
    body: `<p>{{anrede}} {{name}},</p>
<p>vielen Dank für unser Gespräch am {{datum}}. Ich moechte kurz die besprochenen Punkte zusammenfassen:</p>
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
    category: 'kommunikation',
    icon: CalendarCheck,
    subject: 'Terminbestaetigung — {{datum}}',
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
    category: 'kommunikation',
    icon: UserPlus,
    subject: 'Willkommen bei {{firma}}!',
    placeholders: ['anrede', 'name', 'firma'],
    body: `<p>{{anrede}} {{name}},</p>
<p>herzlich willkommen bei {{firma}}! Wir freuen uns, Sie als neuen Kunden / Partner begrüßen zu duerfen.</p>
<p><strong>Ihre nächsten Schritte:</strong></p>
<ol>
<li>Zugang einrichten unter <em>[Link]</em></li>
<li>Profil vervollstaendigen</li>
<li>Erste Schritte in der Anleitung lesen</li>
</ol>
<p>Bei Fragen steht Ihnen Ihr persönlicher Ansprechpartner jederzeit zur Verfuegung.</p>`,
  },
  {
    id: 'zahlungserinnerung',
    name: 'Zahlungserinnerung',
    category: 'finanzen',
    icon: CreditCard,
    subject: 'Freundliche Zahlungserinnerung — Rechnung RE-XXXX',
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

const categoryLabels: Record<string, string> = {
  vertrieb: 'Vertrieb',
  kommunikation: 'Kommunikation',
  finanzen: 'Finanzen',
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface EmailTemplateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSelect: (template: { subject: string; body: string }) => void
}

export function EmailTemplateDialog({
  open,
  onOpenChange,
  onSelect,
}: EmailTemplateDialogProps) {
  const [search, setSearch] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const filtered = templates.filter(
    (t) =>
      !search ||
      t.name.toLowerCase().includes(search.toLowerCase()) ||
      t.category.includes(search.toLowerCase()),
  )

  const selected = templates.find((t) => t.id === selectedId)

  const handleInsert = () => {
    if (!selected) return
    onSelect({ subject: selected.subject, body: selected.body })
    onOpenChange(false)
    setSelectedId(null)
    setSearch('')
  }

  // Group by category
  const grouped = filtered.reduce(
    (acc, t) => {
      if (!acc[t.category]) acc[t.category] = []
      acc[t.category].push(t)
      return acc
    },
    {} as Record<string, EmailTemplate[]>,
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>E-Mail-Vorlage wählen</DialogTitle>
        </DialogHeader>

        <div className="flex flex-1 overflow-hidden gap-4 min-h-0">
          {/* Left: template list */}
          <div className="w-64 shrink-0 flex flex-col border-r border-border pr-4 overflow-hidden">
            <div className="relative mb-3">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Vorlagen suchen..."
                className="w-full rounded-md border border-border bg-background pl-8 pr-3 py-1.5 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>
            <div className="flex-1 overflow-y-auto space-y-4">
              {Object.entries(grouped).map(([category, items]) => (
                <div key={category}>
                  <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                    {categoryLabels[category] ?? category}
                  </p>
                  <div className="space-y-0.5">
                    {items.map((t) => {
                      const Icon = t.icon
                      return (
                        <button
                          key={t.id}
                          onClick={() => setSelectedId(t.id)}
                          className={`flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-sm transition-colors ${
                            selectedId === t.id
                              ? 'bg-primary-light text-primary font-medium'
                              : 'text-foreground hover:bg-secondary'
                          }`}
                        >
                          <Icon className="h-4 w-4 shrink-0" />
                          {t.name}
                        </button>
                      )
                    })}
                  </div>
                </div>
              ))}
              {filtered.length === 0 && (
                <p className="text-sm text-muted-foreground text-center py-4">
                  Keine Vorlagen gefunden
                </p>
              )}
            </div>
          </div>

          {/* Right: preview */}
          <div className="flex-1 overflow-y-auto">
            {selected ? (
              <div className="space-y-4">
                <div>
                  <p className="text-xs text-muted-foreground mb-1">Betreff</p>
                  <p className="text-sm font-medium text-foreground">
                    {selected.subject}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground mb-2">
                    Vorschau
                  </p>
                  <div
                    className="rounded-lg border border-border bg-background p-4 prose prose-sm dark:prose-invert max-w-none text-sm"
                    dangerouslySetInnerHTML={{ __html: selected.body }}
                  />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground mb-1.5">
                    Platzhalter
                  </p>
                  <div className="flex flex-wrap gap-1.5">
                    {selected.placeholders.map((p) => (
                      <span
                        key={p}
                        className="rounded-full bg-amber-100 dark:bg-amber-900/30 px-2.5 py-0.5 text-xs font-mono text-amber-700 dark:text-amber-300"
                      >
                        {`{{${p}}}`}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex h-full items-center justify-center text-muted-foreground">
                <div className="text-center">
                  <FileText className="h-10 w-10 mx-auto mb-2 opacity-30" />
                  <p className="text-sm">
                    Wähle eine Vorlage aus der Liste
                  </p>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-2 pt-3 border-t border-border">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleInsert} disabled={!selected}>
            <FileText className="h-4 w-4 mr-1.5" />
            Vorlage einfügen
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
