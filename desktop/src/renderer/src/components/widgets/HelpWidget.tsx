import { useState, useRef, useEffect } from 'react'
import {
  HelpCircle, X, Search, ChevronDown, ChevronUp, Mail, BookOpen, ExternalLink,
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/cn'

interface FAQItem {
  question: string
  answer: string
}

const FAQ_ITEMS: FAQItem[] = [
  { question: 'Wie erstelle ich ein neues Projekt?', answer: 'Gehe zu Projekte in der Sidebar und klicke auf "+ Neues Projekt". Gib den Projektnamen, die Beschreibung und das Team ein.' },
  { question: 'Wie tracke ich meine Arbeitszeit?', answer: 'Nutze den Timer im Header oder gehe zu deinem Profil → Zeiterfassung. Dort kannst du auch manuell Zeiten nachtragen.' },
  { question: 'Wie erstelle ich eine Rechnung?', answer: 'Gehe zu Buchhaltung → Rechnungen und klicke auf "Neue Rechnung". Wähle den Kunden, fuege Positionen hinzu und versende die Rechnung.' },
  { question: 'Wie lade ich Teammitglieder ein?', answer: 'Gehe zu Team in der Sidebar. Als Admin oder Manager kannst du über "Mitglied hinzufügen" neue Personen einladen.' },
  { question: 'Wie funktioniert der Chat?', answer: 'Klicke auf Chat in der Sidebar. Du kannst Direktnachrichten senden oder Gruppenkanale für Projekte erstellen.' },
  { question: 'Wie beantrage ich Urlaub?', answer: 'Gehe zu deinem Profil → Abwesenheiten und klicke auf "Neue Anfrage". Wähle den Zeitraum und den Grund aus.' },
  { question: 'Wie ändere ich mein Passwort?', answer: 'Gehe zu Einstellungen → Sicherheit. Dort kannst du dein Passwort ändern und Zwei-Faktor-Authentifizierung aktivieren.' },
  { question: 'Wie exportiere ich Daten?', answer: 'In den meisten Modulen gibt es einen Export-Button. Zeiterfassung, Kontakte und Rechnungen können als CSV oder PDF exportiert werden.' },
]

const SHORTCUTS = [
  { keys: 'Ctrl+K', description: 'Globale Suche' },
  { keys: 'Ctrl+,', description: 'Einstellungen' },
  { keys: 'Ctrl+N', description: 'Neues Element' },
  { keys: 'Ctrl+Shift+F', description: 'Vollbild umschalten' },
  { keys: 'Esc', description: 'Dialog/Panel schliessen' },
]

type SectionKey = 'faq' | 'shortcuts' | 'contact' | 'docs'

export function HelpWidget() {
  const [isOpen, setIsOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [openSections, setOpenSections] = useState<Set<SectionKey>>(new Set(['faq']))
  const panelRef = useRef<HTMLDivElement>(null)

  // Filter FAQ
  const filteredFAQ = search
    ? FAQ_ITEMS.filter(
        (item) =>
          item.question.toLowerCase().includes(search.toLowerCase()) ||
          item.answer.toLowerCase().includes(search.toLowerCase()),
      )
    : FAQ_ITEMS

  // Close on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (panelRef.current && !panelRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  // Close on Escape
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && isOpen) {
        setIsOpen(false)
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen])

  const toggleSection = (key: SectionKey) => {
    setOpenSections((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  return (
    <div ref={panelRef} className="fixed bottom-20 right-6 z-[80]">
      {/* Panel */}
      {isOpen && (
        <div className="absolute bottom-16 right-0 w-80 max-h-[500px] rounded-xl border border-border bg-card shadow-2xl overflow-hidden animate-in fade-in slide-in-from-bottom-4 duration-200">
          {/* Header */}
          <div className="flex items-center justify-between px-4 py-3 border-b border-border">
            <div className="flex items-center gap-2">
              <HelpCircle className="h-4 w-4 text-primary" />
              <h3 className="font-semibold text-sm text-foreground">Hilfe & Support</h3>
            </div>
            <button onClick={() => setIsOpen(false)} className="p-1 rounded hover:bg-accent transition-colors">
              <X className="h-4 w-4 text-muted-foreground" />
            </button>
          </div>

          {/* Search */}
          <div className="px-4 py-2 border-b border-border">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
              <Input
                placeholder="Hilfe suchen..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="h-8 text-xs pl-8"
              />
            </div>
          </div>

          {/* Scrollable Content */}
          <div className="overflow-y-auto max-h-[360px]">
            {/* FAQ Section */}
            <Section
              title="Häufige Fragen"
              sectionKey="faq"
              isOpen={openSections.has('faq')}
              onToggle={toggleSection}
            >
              {filteredFAQ.length === 0 && (
                <p className="text-xs text-muted-foreground px-4 py-2">Keine Ergebnisse für "{search}"</p>
              )}
              {filteredFAQ.map((item, i) => (
                <FAQAccordion key={i} item={item} />
              ))}
            </Section>

            {/* Shortcuts Section */}
            {!search && (
              <Section
                title="Tastaturkuerzel"
                sectionKey="shortcuts"
                isOpen={openSections.has('shortcuts')}
                onToggle={toggleSection}
              >
                <div className="px-4 py-1 space-y-1.5">
                  {SHORTCUTS.map((s, i) => (
                    <div key={i} className="flex items-center justify-between text-xs">
                      <span className="text-muted-foreground">{s.description}</span>
                      <kbd className="inline-flex items-center rounded border border-border bg-secondary px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground">
                        {s.keys}
                      </kbd>
                    </div>
                  ))}
                </div>
              </Section>
            )}

            {/* Contact Section */}
            {!search && (
              <Section
                title="Kontakt & Support"
                sectionKey="contact"
                isOpen={openSections.has('contact')}
                onToggle={toggleSection}
              >
                <div className="px-4 py-1 space-y-2">
                  <div className="flex items-center gap-2 text-xs">
                    <Mail className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="text-foreground">support@kmuhub.ch</span>
                  </div>
                  <div className="flex items-center gap-2 text-xs">
                    <span className="text-muted-foreground">Tel:</span>
                    <span className="text-foreground">+41 44 123 45 67</span>
                  </div>
                  <button className="flex items-center gap-2 text-xs text-primary hover:text-primary/80 transition-colors mt-1">
                    <ExternalLink className="h-3.5 w-3.5" />
                    Support-Ticket erstellen
                  </button>
                </div>
              </Section>
            )}

            {/* Docs Section */}
            {!search && (
              <Section
                title="Dokumentation"
                sectionKey="docs"
                isOpen={openSections.has('docs')}
                onToggle={toggleSection}
              >
                <div className="px-4 py-1 space-y-2">
                  {[
                    { label: 'Benutzerhandbuch', icon: BookOpen },
                    { label: 'API Dokumentation', icon: BookOpen },
                    { label: 'Changelog', icon: BookOpen },
                  ].map((doc, i) => (
                    <button key={i} className="flex items-center gap-2 text-xs text-foreground hover:text-primary transition-colors w-full">
                      <doc.icon className="h-3.5 w-3.5 text-muted-foreground" />
                      {doc.label}
                      <ExternalLink className="h-3 w-3 text-muted-foreground ml-auto" />
                    </button>
                  ))}
                </div>
              </Section>
            )}
          </div>

          {/* Footer */}
          <div className="px-4 py-2 border-t border-border">
            <p className="text-[10px] text-muted-foreground text-center">KMU Hub v0.1.0 (Beta)</p>
          </div>
        </div>
      )}

      {/* Floating Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          'flex h-12 w-12 items-center justify-center rounded-full shadow-lg transition-all hover:shadow-xl hover:scale-105',
          isOpen
            ? 'bg-muted text-muted-foreground'
            : 'bg-primary text-primary-foreground',
        )}
      >
        {isOpen ? <X className="h-5 w-5" /> : <HelpCircle className="h-5 w-5" />}
      </button>
    </div>
  )
}

function Section({
  title,
  sectionKey,
  isOpen,
  onToggle,
  children,
}: {
  title: string
  sectionKey: SectionKey
  isOpen: boolean
  onToggle: (key: SectionKey) => void
  children: React.ReactNode
}) {
  return (
    <div className="border-b border-border last:border-b-0">
      <button
        onClick={() => onToggle(sectionKey)}
        className="flex items-center justify-between w-full px-4 py-2.5 text-xs font-semibold text-foreground hover:bg-accent/50 transition-colors"
      >
        {title}
        {isOpen ? <ChevronUp className="h-3.5 w-3.5 text-muted-foreground" /> : <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />}
      </button>
      {isOpen && <div className="pb-2">{children}</div>}
    </div>
  )
}

function FAQAccordion({ item }: { item: FAQItem }) {
  const [isOpen, setIsOpen] = useState(false)
  return (
    <div className="px-4">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-start gap-2 w-full text-left py-1.5 text-xs text-foreground hover:text-primary transition-colors"
      >
        <span className="shrink-0 mt-0.5">{isOpen ? '−' : '+'}</span>
        <span>{item.question}</span>
      </button>
      {isOpen && (
        <p className="text-xs text-muted-foreground pl-5 pb-2 leading-relaxed">
          {item.answer}
        </p>
      )}
    </div>
  )
}
