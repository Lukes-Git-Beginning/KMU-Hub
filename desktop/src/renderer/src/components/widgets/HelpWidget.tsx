import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  HelpCircle, X, Search, ChevronDown, ChevronUp, Mail, BookOpen, ExternalLink,
} from 'lucide-react'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/cn'

interface FAQItem {
  question: string
  answer: string
}

type SectionKey = 'faq' | 'shortcuts' | 'contact' | 'docs'

export function HelpWidget() {
  const { t } = useTranslation()
  const [isOpen, setIsOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [openSections, setOpenSections] = useState<Set<SectionKey>>(new Set(['faq']))
  const panelRef = useRef<HTMLDivElement>(null)

  const FAQ_ITEMS: FAQItem[] = [
    { question: t('widgets.help.faq.createProject.question'), answer: t('widgets.help.faq.createProject.answer') },
    { question: t('widgets.help.faq.trackTime.question'), answer: t('widgets.help.faq.trackTime.answer') },
    { question: t('widgets.help.faq.createInvoice.question'), answer: t('widgets.help.faq.createInvoice.answer') },
    { question: t('widgets.help.faq.inviteTeam.question'), answer: t('widgets.help.faq.inviteTeam.answer') },
    { question: t('widgets.help.faq.chat.question'), answer: t('widgets.help.faq.chat.answer') },
    { question: t('widgets.help.faq.requestLeave.question'), answer: t('widgets.help.faq.requestLeave.answer') },
    { question: t('widgets.help.faq.changePassword.question'), answer: t('widgets.help.faq.changePassword.answer') },
    { question: t('widgets.help.faq.exportData.question'), answer: t('widgets.help.faq.exportData.answer') },
  ]

  const SHORTCUTS = [
    { keys: 'Ctrl+K', description: t('widgets.help.shortcuts.globalSearch') },
    { keys: 'Ctrl+,', description: t('widgets.help.shortcuts.settings') },
    { keys: 'Ctrl+N', description: t('widgets.help.shortcuts.newItem') },
    { keys: 'Ctrl+Shift+F', description: t('widgets.help.shortcuts.toggleFullscreen') },
    { keys: 'Esc', description: t('widgets.help.shortcuts.closeDialog') },
  ]

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
              <h3 className="font-semibold text-sm text-foreground">{t('widgets.help.title')}</h3>
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
                placeholder={t('widgets.help.searchPlaceholder')}
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
              title={t('widgets.help.sections.faq')}
              sectionKey="faq"
              isOpen={openSections.has('faq')}
              onToggle={toggleSection}
            >
              {filteredFAQ.length === 0 && (
                <p className="text-xs text-muted-foreground px-4 py-2">{t('widgets.help.noResults', { query: search })}</p>
              )}
              {filteredFAQ.map((item, i) => (
                <FAQAccordion key={i} item={item} />
              ))}
            </Section>

            {/* Shortcuts Section */}
            {!search && (
              <Section
                title={t('widgets.help.sections.shortcuts')}
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
                title={t('widgets.help.sections.contact')}
                sectionKey="contact"
                isOpen={openSections.has('contact')}
                onToggle={toggleSection}
              >
                <div className="px-4 py-1 space-y-2">
                  <div className="flex items-center gap-2 text-xs">
                    <Mail className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="text-foreground">support@zentria.tech</span>
                  </div>
                  <div className="flex items-center gap-2 text-xs">
                    <span className="text-muted-foreground">Tel:</span>
                    <span className="text-foreground">+49 30 123 45 67</span>
                  </div>
                  <button className="flex items-center gap-2 text-xs text-primary hover:text-primary/80 transition-colors mt-1">
                    <ExternalLink className="h-3.5 w-3.5" />
                    {t('widgets.help.contact.createTicket')}
                  </button>
                </div>
              </Section>
            )}

            {/* Docs Section */}
            {!search && (
              <Section
                title={t('widgets.help.sections.docs')}
                sectionKey="docs"
                isOpen={openSections.has('docs')}
                onToggle={toggleSection}
              >
                <div className="px-4 py-1 space-y-2">
                  {[
                    { label: t('widgets.help.docs.userManual'), icon: BookOpen },
                    { label: t('widgets.help.docs.apiDocs'), icon: BookOpen },
                    { label: t('widgets.help.docs.changelog'), icon: BookOpen },
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
            <p className="text-[10px] text-muted-foreground text-center">Cosmi v0.1.0 (Beta)</p>
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
