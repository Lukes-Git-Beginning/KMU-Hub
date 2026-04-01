// TODO: Wire to backend — no onboarding API exists yet. Currently uses mock data.
import { useState, useMemo } from 'react'
import {
  CheckCircle2,
  Circle,
  ListChecks,
  Plus,
  ChevronDown,
  ChevronRight,
  Users,
  Briefcase,
  Monitor,
  FileText,
  Sparkles,
} from 'lucide-react'
import { toast } from 'sonner'
import { EmptyState } from '@/components/shared'

// ============================================================
// Types
// ============================================================

interface ChecklistItem {
  id: string
  label: string
  completed: boolean
  assignee?: string
  dueInDays?: number
}

interface OnboardingTemplate {
  id: string
  name: string
  icon: typeof ListChecks
  color: string
  items: ChecklistItem[]
}

interface ActiveOnboarding {
  id: string
  employeeName: string
  employeeInitials: string
  startDate: string
  templateId: string
  templateName: string
  items: ChecklistItem[]
}

// ============================================================
// Mock Data
// ============================================================

const TEMPLATES: OnboardingTemplate[] = [
  {
    id: 'tpl-it',
    name: 'IT-Onboarding',
    icon: Monitor,
    color: 'bg-info-light text-info',
    items: [
      { id: 'it-1', label: 'Laptop / Arbeitsgerät bereitstellen', completed: false, assignee: 'IT-Abteilung', dueInDays: -3 },
      { id: 'it-2', label: 'E-Mail-Konto einrichten', completed: false, assignee: 'IT-Abteilung', dueInDays: -2 },
      { id: 'it-3', label: 'Cosmi Zugang erstellen', completed: false, assignee: 'IT-Abteilung', dueInDays: -2 },
      { id: 'it-4', label: 'VPN-Zugang konfigurieren', completed: false, assignee: 'IT-Abteilung', dueInDays: -1 },
      { id: 'it-5', label: 'Software-Lizenzen zuweisen', completed: false, assignee: 'IT-Abteilung', dueInDays: 0 },
      { id: 'it-6', label: 'Drucker einrichten', completed: false, assignee: 'IT-Abteilung', dueInDays: 1 },
      { id: 'it-7', label: 'Passwort-Manager Einweisung', completed: false, assignee: 'IT-Abteilung', dueInDays: 1 },
    ],
  },
  {
    id: 'tpl-hr',
    name: 'HR-Onboarding',
    icon: Users,
    color: 'bg-primary-light text-primary',
    items: [
      { id: 'hr-1', label: 'Personalfragebogen ausgefüllt', completed: false, assignee: 'HR', dueInDays: -5 },
      { id: 'hr-2', label: 'Arbeitsvertrag unterschrieben', completed: false, assignee: 'HR', dueInDays: -3 },
      { id: 'hr-3', label: 'Steuer-ID und SV-Nummer erfasst', completed: false, assignee: 'HR', dueInDays: -1 },
      { id: 'hr-4', label: 'Bankverbindung erfasst', completed: false, assignee: 'HR', dueInDays: 0 },
      { id: 'hr-5', label: 'Betriebsarzt-Termin vereinbart', completed: false, assignee: 'HR', dueInDays: 3 },
      { id: 'hr-6', label: 'Willkommens-Mail verschickt', completed: false, assignee: 'HR', dueInDays: 0 },
      { id: 'hr-7', label: 'Team-Vorstellung geplant', completed: false, assignee: 'Teamleitung', dueInDays: 0 },
      { id: 'hr-8', label: 'Feedback-Gespräch (Woche 1)', completed: false, assignee: 'HR', dueInDays: 5 },
    ],
  },
  {
    id: 'tpl-fach',
    name: 'Fachliches Onboarding',
    icon: Briefcase,
    color: 'bg-warning-light text-warning',
    items: [
      { id: 'f-1', label: 'Einarbeitungsplan erstellt', completed: false, assignee: 'Teamleitung', dueInDays: -2 },
      { id: 'f-2', label: 'Mentor/Buddy zugewiesen', completed: false, assignee: 'Teamleitung', dueInDays: -1 },
      { id: 'f-3', label: 'Projekt-Zugänge eingerichtet', completed: false, assignee: 'Teamleitung', dueInDays: 0 },
      { id: 'f-4', label: 'Dokumentation durcharbeiten', completed: false, dueInDays: 3 },
      { id: 'f-5', label: 'Erste Aufgabe zugewiesen', completed: false, assignee: 'Teamleitung', dueInDays: 2 },
      { id: 'f-6', label: 'Code-Review Prozess erklaert', completed: false, assignee: 'Mentor', dueInDays: 3 },
    ],
  },
  {
    id: 'tpl-compliance',
    name: 'Compliance & Sicherheit',
    icon: FileText,
    color: 'bg-error-light text-error',
    items: [
      { id: 'c-1', label: 'Datenschutzbelehrung (DSGVO)', completed: false, assignee: 'HR', dueInDays: 0 },
      { id: 'c-2', label: 'Arbeitssicherheits-Unterweisung', completed: false, assignee: 'Sicherheitsbeauftragter', dueInDays: 1 },
      { id: 'c-3', label: 'Brandschutz-Einweisung', completed: false, assignee: 'Sicherheitsbeauftragter', dueInDays: 2 },
      { id: 'c-4', label: 'Geheimhaltungsvereinbarung unterschrieben', completed: false, assignee: 'HR', dueInDays: 0 },
      { id: 'c-5', label: 'IT-Sicherheitsrichtlinie gelesen', completed: false, dueInDays: 1 },
    ],
  },
]

const ACTIVE_ONBOARDINGS: ActiveOnboarding[] = [
  {
    id: 'ob-1',
    employeeName: 'Tom Brunner',
    employeeInitials: 'TB',
    startDate: '2025-06-01',
    templateId: 'tpl-it',
    templateName: 'IT-Onboarding',
    items: [
      { id: 'it-1', label: 'Laptop / Arbeitsgerät bereitstellen', completed: true, assignee: 'IT-Abteilung' },
      { id: 'it-2', label: 'E-Mail-Konto einrichten', completed: true, assignee: 'IT-Abteilung' },
      { id: 'it-3', label: 'Cosmi Zugang erstellen', completed: true, assignee: 'IT-Abteilung' },
      { id: 'it-4', label: 'VPN-Zugang konfigurieren', completed: true, assignee: 'IT-Abteilung' },
      { id: 'it-5', label: 'Software-Lizenzen zuweisen', completed: true, assignee: 'IT-Abteilung' },
      { id: 'it-6', label: 'Drucker einrichten', completed: true, assignee: 'IT-Abteilung' },
      { id: 'it-7', label: 'Passwort-Manager Einweisung', completed: true, assignee: 'IT-Abteilung' },
    ],
  },
  {
    id: 'ob-2',
    employeeName: 'Tom Brunner',
    employeeInitials: 'TB',
    startDate: '2025-06-01',
    templateId: 'tpl-hr',
    templateName: 'HR-Onboarding',
    items: [
      { id: 'hr-1', label: 'Personalfragebogen ausgefüllt', completed: true, assignee: 'HR' },
      { id: 'hr-2', label: 'Arbeitsvertrag unterschrieben', completed: true, assignee: 'HR' },
      { id: 'hr-3', label: 'Steuer-ID und SV-Nummer erfasst', completed: true, assignee: 'HR' },
      { id: 'hr-4', label: 'Bankverbindung erfasst', completed: true, assignee: 'HR' },
      { id: 'hr-5', label: 'Betriebsarzt-Termin vereinbart', completed: true, assignee: 'HR' },
      { id: 'hr-6', label: 'Willkommens-Mail verschickt', completed: true, assignee: 'HR' },
      { id: 'hr-7', label: 'Team-Vorstellung geplant', completed: true, assignee: 'Teamleitung' },
      { id: 'hr-8', label: 'Feedback-Gespräch (Woche 1)', completed: true, assignee: 'HR' },
    ],
  },
  {
    id: 'ob-3',
    employeeName: 'Claudia Frei',
    employeeInitials: 'CF',
    startDate: '2025-03-01',
    templateId: 'tpl-fach',
    templateName: 'Fachliches Onboarding',
    items: [
      { id: 'f-1', label: 'Einarbeitungsplan erstellt', completed: true, assignee: 'Teamleitung' },
      { id: 'f-2', label: 'Mentor/Buddy zugewiesen', completed: true, assignee: 'Teamleitung' },
      { id: 'f-3', label: 'Projekt-Zugänge eingerichtet', completed: true, assignee: 'Teamleitung' },
      { id: 'f-4', label: 'Dokumentation durcharbeiten', completed: true },
      { id: 'f-5', label: 'Erste Aufgabe zugewiesen', completed: false, assignee: 'Teamleitung' },
      { id: 'f-6', label: 'Code-Review Prozess erklaert', completed: false, assignee: 'Mentor' },
    ],
  },
]

// ============================================================
// Component
// ============================================================

export function OnboardingChecklist() {
  const [activeTab, setActiveTab] = useState<'aktiv' | 'vorlagen'>('aktiv')
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set(['ob-3']))
  const [onboardings, setOnboardings] = useState(ACTIVE_ONBOARDINGS)

  const toggleExpand = (id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleItem = (obId: string, itemId: string) => {
    setOnboardings((prev) =>
      prev.map((ob) =>
        ob.id === obId
          ? { ...ob, items: ob.items.map((item) => item.id === itemId ? { ...item, completed: !item.completed } : item) }
          : ob,
      ),
    )
  }

  const completedAll = useMemo(() => {
    return onboardings.filter((ob) => ob.items.every((i) => i.completed)).length
  }, [onboardings])

  const inProgress = useMemo(() => {
    return onboardings.filter((ob) => ob.items.some((i) => i.completed) && !ob.items.every((i) => i.completed)).length
  }, [onboardings])

  return (
    <div className="space-y-4">
      {/* KPI Row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">Aktive Onboardings</p>
          <p className="text-lg font-semibold text-foreground">{onboardings.length}</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">In Bearbeitung</p>
          <p className="text-lg font-semibold text-warning">{inProgress}</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">Abgeschlossen</p>
          <p className="text-lg font-semibold text-success">{completedAll}</p>
        </div>
        <div className="rounded-lg border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground mb-1">Vorlagen</p>
          <p className="text-lg font-semibold text-foreground">{TEMPLATES.length}</p>
        </div>
      </div>

      {/* Sub-Tabs */}
      <div className="flex items-center gap-4 border-b border-border">
        {[
          { key: 'aktiv' as const, label: `Aktive Checklisten (${onboardings.length})` },
          { key: 'vorlagen' as const, label: `Vorlagen (${TEMPLATES.length})` },
        ].map((t) => (
          <button
            key={t.key}
            onClick={() => setActiveTab(t.key)}
            className={`border-b-2 px-1 pb-2 text-sm transition-colors ${
              activeTab === t.key ? 'border-primary text-primary font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t.label}
          </button>
        ))}
        <button
          onClick={() => toast.info('Neues Onboarding starten (Mock)')}
          className="ml-auto flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors mb-1"
        >
          <Plus className="h-3.5 w-3.5" />
          Onboarding starten
        </button>
      </div>

      {/* Active Checklists */}
      {activeTab === 'aktiv' && (
        <div className="space-y-3">
          {onboardings.length === 0 ? (
            <EmptyState
              icon={ListChecks}
              title="Keine aktiven Onboardings"
              description="Starte ein neues Onboarding für einen Mitarbeiter"
              action={{ label: 'Onboarding starten', onClick: () => toast.info('Mock') }}
            />
          ) : (
            onboardings.map((ob) => {
              const completedCount = ob.items.filter((i) => i.completed).length
              const totalCount = ob.items.length
              const progress = Math.round((completedCount / totalCount) * 100)
              const isExpanded = expandedIds.has(ob.id)
              const isDone = progress === 100

              return (
                <div key={ob.id} className={`rounded-lg border bg-card overflow-hidden ${isDone ? 'border-success/30' : 'border-border'}`}>
                  <div
                    className="flex items-center gap-3 p-4 cursor-pointer hover:bg-secondary/20 transition-colors"
                    onClick={() => toggleExpand(ob.id)}
                  >
                    {isExpanded ? <ChevronDown className="h-4 w-4 text-muted-foreground flex-shrink-0" /> : <ChevronRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />}
                    <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary-light text-xs font-medium text-primary flex-shrink-0">
                      {ob.employeeInitials}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium text-foreground">{ob.employeeName}</span>
                        <span className="text-xs text-muted-foreground">— {ob.templateName}</span>
                        {isDone && <Sparkles className="h-3.5 w-3.5 text-success" />}
                      </div>
                      <div className="flex items-center gap-3 mt-1">
                        <div className="flex-1 max-w-[200px] h-1.5 rounded-full bg-secondary">
                          <div
                            className={`h-full rounded-full transition-all ${isDone ? 'bg-success' : 'bg-primary'}`}
                            style={{ width: `${progress}%` }}
                          />
                        </div>
                        <span className="text-xs text-muted-foreground tabular-nums">{completedCount}/{totalCount} ({progress}%)</span>
                      </div>
                    </div>
                  </div>

                  {isExpanded && (
                    <div className="border-t border-border divide-y divide-border-muted">
                      {ob.items.map((item) => (
                        <div
                          key={item.id}
                          className={`flex items-center gap-3 px-4 py-2.5 cursor-pointer hover:bg-secondary/20 transition-colors ${
                            item.completed ? 'opacity-60' : ''
                          }`}
                          onClick={() => toggleItem(ob.id, item.id)}
                        >
                          {item.completed ? (
                            <CheckCircle2 className="h-4 w-4 text-success flex-shrink-0" />
                          ) : (
                            <Circle className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                          )}
                          <span className={`text-sm flex-1 ${item.completed ? 'line-through text-muted-foreground' : 'text-foreground'}`}>
                            {item.label}
                          </span>
                          {item.assignee && (
                            <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground flex-shrink-0">
                              {item.assignee}
                            </span>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )
            })
          )}
        </div>
      )}

      {/* Templates */}
      {activeTab === 'vorlagen' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {TEMPLATES.map((tpl) => {
            const TplIcon = tpl.icon
            return (
              <div key={tpl.id} className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-start gap-3 mb-3">
                  <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${tpl.color}`}>
                    <TplIcon className="h-5 w-5" />
                  </div>
                  <div>
                    <h4 className="text-sm font-medium text-foreground">{tpl.name}</h4>
                    <p className="text-xs text-muted-foreground">{tpl.items.length} Aufgaben</p>
                  </div>
                </div>
                <div className="space-y-1.5">
                  {tpl.items.slice(0, 4).map((item) => (
                    <div key={item.id} className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Circle className="h-3 w-3 flex-shrink-0" />
                      <span className="truncate">{item.label}</span>
                    </div>
                  ))}
                  {tpl.items.length > 4 && (
                    <p className="text-xs text-muted-foreground pl-5">+{tpl.items.length - 4} weitere</p>
                  )}
                </div>
                <div className="flex gap-2 mt-3 pt-3 border-t border-border-muted">
                  <button
                    onClick={() => toast.info(`Vorlage "${tpl.name}" bearbeiten (Mock)`)}
                    className="flex-1 rounded-md border border-border py-1.5 text-xs text-foreground hover:bg-secondary transition-colors text-center"
                  >
                    Bearbeiten
                  </button>
                  <button
                    onClick={() => toast.info(`Onboarding mit "${tpl.name}" starten (Mock)`)}
                    className="flex-1 rounded-md bg-primary py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors text-center"
                  >
                    Verwenden
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
