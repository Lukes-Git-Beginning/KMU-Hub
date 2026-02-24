/**
 * ProfileWidgetSuggestions — recommends dashboard widgets based on
 * the active business profile.
 *
 * Dismissable suggestion cards: "Widget X hinzufuegen".
 */
import { useState } from 'react'
import {
  Truck,
  FileCheck,
  Clock,
  Ticket,
  Rocket,
  MessageSquare,
  Receipt,
  Users,
  X,
  Plus,
} from 'lucide-react'
import { useProfileStore } from '../../stores/profile'
import { moduleHsl, moduleHslBg } from '../../components/layout/sidebar/nav-items'
import type { BusinessProfileId } from '../../config/business-profiles'

interface WidgetSuggestion {
  id: string
  label: string
  description: string
  icon: React.ElementType
  /** nav-items module ID for color lookup */
  moduleId: string
}

const profileSuggestions: Record<string, WidgetSuggestion[]> = {
  handwerk: [
    { id: 'fuhrpark-status', label: 'Fuhrpark-Status', description: 'Fahrzeuge und naechste Termine', icon: Truck, moduleId: 'fuhrpark' },
    { id: 'aktive-rapporte', label: 'Aktive Rapporte', description: 'Offene Rapporte und Freigaben', icon: FileCheck, moduleId: 'rapporte' },
    { id: 'zeiterfassung-quick', label: 'Zeiterfassung', description: 'Timer starten und Stunden', icon: Clock, moduleId: 'zeiterfassung' },
  ],
  it_tech: [
    { id: 'offene-tickets', label: 'Offene Tickets', description: 'Support-Tickets nach Prioritaet', icon: Ticket, moduleId: 'helpdesk' },
    { id: 'sprint-status', label: 'Sprint-Status', description: 'Aktuelle Sprint-Fortschritte', icon: Rocket, moduleId: 'projects' },
    { id: 'nachrichten-quick', label: 'Nachrichten', description: 'Ungelesene Chat-Nachrichten', icon: MessageSquare, moduleId: 'chat' },
  ],
  dienstleistung: [
    { id: 'naechste-termine', label: 'Naechste Termine', description: 'Kommende Termine und Meetings', icon: Clock, moduleId: 'calendar' },
    { id: 'offene-rechnungen', label: 'Offene Rechnungen', description: 'Ueberfaellige und offene Rechnungen', icon: Receipt, moduleId: 'finance' },
    { id: 'kontakte-quick', label: 'Kontakte', description: 'Zuletzt bearbeitete Kontakte', icon: Users, moduleId: 'contacts' },
  ],
}

// Map business profile IDs to suggestion categories
function getSuggestionsForProfile(profileId: BusinessProfileId | null): WidgetSuggestion[] {
  if (!profileId) return []
  if (profileId.includes('handwerk') || profileId.includes('bau')) return profileSuggestions.handwerk
  if (profileId.includes('it') || profileId.includes('tech') || profileId.includes('agentur')) return profileSuggestions.it_tech
  if (profileId.includes('dienstleist') || profileId.includes('beratung') || profileId.includes('kanzlei')) return profileSuggestions.dienstleistung
  // Default: show general suggestions
  return profileSuggestions.dienstleistung
}

export function ProfileWidgetSuggestions() {
  const { businessProfileId } = useProfileStore()
  const [dismissed, setDismissed] = useState<Set<string>>(new Set())
  const [allDismissed, setAllDismissed] = useState(false)

  const suggestions = getSuggestionsForProfile(businessProfileId)
  const visible = suggestions.filter((s) => !dismissed.has(s.id))

  if (allDismissed || visible.length === 0 || !businessProfileId) return null

  return (
    <div className="mb-6">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          Empfohlene Widgets
        </h3>
        <button
          onClick={() => setAllDismissed(true)}
          className="text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          Alle ausblenden
        </button>
      </div>
      <div className="flex flex-wrap gap-3">
        {visible.map((suggestion) => (
          <div
            key={suggestion.id}
            className="group relative flex items-center gap-3 rounded-lg border border-dashed border-border/60 bg-card/50 px-4 py-3 hover:border-border hover:bg-card transition-colors"
          >
            <div
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg"
              style={{ backgroundColor: moduleHslBg(suggestion.moduleId), color: moduleHsl(suggestion.moduleId) }}
            >
              <suggestion.icon className="h-4.5 w-4.5" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">{suggestion.label}</p>
              <p className="text-xs text-muted-foreground">{suggestion.description}</p>
            </div>
            <button className="ml-2 rounded-md bg-primary/10 p-1.5 text-primary hover:bg-primary/20 transition-colors">
              <Plus className="h-3.5 w-3.5" />
            </button>
            <button
              onClick={() => setDismissed((prev) => new Set(prev).add(suggestion.id))}
              className="absolute -right-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-muted text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity hover:bg-muted/80"
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
