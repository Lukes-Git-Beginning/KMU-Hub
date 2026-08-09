/**
 * Kontakte module layout — "Kunden-Zentrale" mit Bereichs-Navigation.
 *
 * Die Bereiche liefen bis 2026-08-10 über `NavLink` + `<Outlet/>`, also über den
 * Router. Für den Anpassungs-Editor war das der Grund, warum Kontakte als einziges
 * Modul keine schaltbaren Bereiche hatte: die Sandbox rendert das Modul ohne
 * passende URL, `<Outlet/>` bliebe leer, und einen eigenen Router darf sie nicht
 * aufmachen (`#/editor-window` ist selbst eine Route des globalen Hash-Routers).
 *
 * Seitdem führt **Zustand** den aktiven Bereich, und die Routen bleiben als
 * Einstieg erhalten — beide Richtungen sind verdrahtet:
 *   - URL → Zustand: ein Deep-Link auf `/kontakte/firmen` und der Zurück-Button
 *     des Browsers setzen den Bereich (der Effekt unten).
 *   - Zustand → URL: ein Klick auf einen Bereich schreibt den Pfad, damit Links
 *     teilbar bleiben und der Verlauf stimmt.
 * Im Editor entfällt nur der zweite Teil — dort wird der Bereich rein umgeschaltet,
 * ohne zu navigieren (der Editor blockiert Navigation ohnehin).
 *
 * Die Detail-Seiten (Firma, Deal, Beratungsprotokoll) bleiben echte Routen und
 * rendern weiter über `<Outlet/>`, mitsamt der Bereichs-Leiste darüber.
 */
import { lazy, Suspense, useEffect, useState } from 'react'
import { Outlet, matchPath, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Users, Building2, TrendingUp, Activity, Inbox, BarChart3 } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useCapabilitySet } from '@/hooks/useCapability'
import { RestrictedModeBadge } from '@/components/shared/rbac/RestrictedModeBadge'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import type { ModuleSkeletonVariant } from '@/components/shared'
import {
  EditableText,
  useEditorSurface,
  useModuleAreas,
  useEditorFocusEffect,
  useEditorContextReport,
} from '@/components/customization/EditorSurface'

const KontaktePage = lazy(() => import('@/modules/kontakte/KontaktePage'))
const LeadsInboxPage = lazy(() => import('@/modules/kontakte/leads/LeadsInboxPage'))
const CompaniesListPage = lazy(() => import('@/modules/crm/companies/CompaniesListPage'))
const DealsListPage = lazy(() => import('@/modules/crm/deals/DealsListPage'))
const ActivitiesListPage = lazy(() => import('@/modules/crm/activities/ActivitiesListPage'))
const AuswertungenPage = lazy(() => import('@/modules/kontakte/AuswertungenPage'))

export type KontakteSection =
  | 'kontakte'
  | 'leads'
  | 'firmen'
  | 'pipeline'
  | 'aktivitaeten'
  | 'auswertungen'

interface SectionDef {
  key: KontakteSection
  /** Route, die diesen Bereich anspringt — bleibt als Deep-Link erhalten. */
  path: string
  icon: typeof Users
  labelKey: string
  capKey: string
  /** Skelett-Form beim Nachladen, wie zuvor in der Routen-Tabelle vergeben. */
  variant: ModuleSkeletonVariant
  Component: React.LazyExoticComponent<() => React.JSX.Element | null>
}

const ALL_SECTIONS: SectionDef[] = [
  { key: 'kontakte', path: '/kontakte', icon: Users, labelKey: 'crm.nav.contacts', capKey: 'crm:contact:read', variant: 'kontakte', Component: KontaktePage },
  { key: 'leads', path: '/kontakte/leads', icon: Inbox, labelKey: 'crm.nav.leads', capKey: 'crm:contact:read', variant: 'leads', Component: LeadsInboxPage },
  { key: 'firmen', path: '/kontakte/firmen', icon: Building2, labelKey: 'crm.nav.companies', capKey: 'crm:contact:read', variant: 'kontakte', Component: CompaniesListPage },
  { key: 'pipeline', path: '/kontakte/pipeline', icon: TrendingUp, labelKey: 'crm.nav.deals', capKey: 'crm:deal:read', variant: 'pipeline', Component: DealsListPage },
  { key: 'aktivitaeten', path: '/kontakte/aktivitaeten', icon: Activity, labelKey: 'crm.nav.activities', capKey: 'crm:contact:read', variant: 'aktivitaeten', Component: ActivitiesListPage },
  { key: 'auswertungen', path: '/kontakte/auswertungen', icon: BarChart3, labelKey: 'crm.nav.reports', capKey: 'crm:deal:read', variant: 'dashboard', Component: AuswertungenPage },
]

/** Detail-Seiten — sie bleiben Routen und rendern im Outlet statt eines Bereichs. */
const DETAIL_PATTERNS = [
  '/kontakte/firmen/:id',
  '/kontakte/pipeline/:id',
  '/kontakte/protokoll/:contactId/:protocolId',
]

/**
 * Welcher Bereich gehört zu dieser URL? `null`, wenn keiner (z. B. im Editor-Fenster).
 *
 * Präfix-Vergleich, längster Pfad gewinnt — damit die Detail-Seiten den Bereich
 * behalten, aus dem sie kommen: `/kontakte/firmen/co-003` gehört zu „Unternehmen",
 * nicht zu „Kontakte". Genau das taten vorher die `NavLink`s mit `end: false`; ein
 * exakter Vergleich ließ auf jeder Detail-Seite den ersten Bereich aufleuchten.
 */
function sectionFromPath(pathname: string): KontakteSection | null {
  const clean = pathname.replace(/\/+$/, '') || '/kontakte'
  const match = [...ALL_SECTIONS]
    .sort((a, b) => b.path.length - a.path.length)
    .find((s) => clean === s.path || clean.startsWith(`${s.path}/`))
  return match?.key ?? null
}

export default function KontakteLayout(): React.JSX.Element {
  const { t } = useTranslation()
  const { has, ready } = useCapabilitySet()
  const location = useLocation()
  const navigate = useNavigate()
  const { editing } = useEditorSurface()

  const [section, setSection] = useState<KontakteSection>(
    () => sectionFromPath(location.pathname) ?? 'kontakte',
  )

  // URL → Zustand: Deep-Link und Zurück-Button führen. Im Editor-Fenster passt keine
  // URL auf einen Bereich, dann bleibt der Zustand unberührt.
  useEffect(() => {
    const fromUrl = sectionFromPath(location.pathname)
    if (fromUrl) setSection(fromUrl)
  }, [location.pathname])

  const showDetail = DETAIL_PATTERNS.some((pattern) => matchPath(pattern, location.pathname))

  // Bereiche sind im Editor schaltbar (moduleAreas). Ein fehlender Schlüssel heißt
  // AN — sonst wäre das Modul beim ersten Öffnen leer.
  const areaEnabled = useModuleAreas('kontakte')
  const visibleSections = ALL_SECTIONS.filter(
    (item) => (!ready || has(item.capKey)) && areaEnabled[item.key] !== false,
  )

  // Schaltet jemand den gerade offenen Bereich ab, auf den ersten sichtbaren gehen.
  useEffect(() => {
    if (visibleSections.length > 0 && !visibleSections.some((s) => s.key === section)) {
      setSection(visibleSections[0].key)
    }
  }, [visibleSections, section])

  // Editor-Leiste → Vorschau. Bereiche und Begriffe sind auf der Kontakt-Liste am
  // dichtesten zu beurteilen, Auswertungen naturgemäß im eigenen Bereich.
  useEditorFocusEffect({
    begriffe: () => setSection('kontakte'),
    bereiche: () => setSection('kontakte'),
    wertelisten: () => setSection('pipeline'),
    felder: () => setSection('kontakte'),
    spalten: () => setSection('firmen'),
    statistik: () => setSection('auswertungen'),
  })

  // …und zurück. Nur eindeutige Orte melden: die Kontakt-Liste ist gleichzeitig
  // Heimat von Begriffen, Feldern und Bereichen und meldet deshalb nichts.
  useEditorContextReport(
    section === 'auswertungen' ? 'statistik' : section === 'pipeline' ? 'wertelisten' : null,
  )

  const selectSection = (key: KontakteSection, path: string): void => {
    setSection(key)
    // In der Sandbox nur umschalten — der Editor blockiert Navigation, und die
    // Vorschau soll trotzdem begehbar bleiben.
    if (!editing) navigate(path)
  }

  const active = visibleSections.find((s) => s.key === section) ?? visibleSections[0]

  return (
    <div className="flex h-full flex-col animate-fade-in">
      {/* Bereichs-Leiste — 1:1 Optik wie zuvor */}
      <nav className="flex items-center gap-1 border-b border-border bg-card px-6 py-2">
        {visibleSections.map((item) => (
          <button
            key={item.key}
            type="button"
            onClick={() => selectSection(item.key, item.path)}
            className={cn(
              'flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium transition-all',
              item.key === section
                ? 'bg-primary/10 text-primary tab-accent-active'
                : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
            )}
          >
            <item.icon className="h-4 w-4" />
            <EditableText as="span" dkey={item.labelKey} interactive />
          </button>
        ))}
        <div className="ml-auto">
          <RestrictedModeBadge module="crm" />
        </div>
      </nav>

      {/* Inhalt: Detail-Seiten kommen aus dem Router, Bereiche aus dem Zustand. */}
      <div className="flex-1 overflow-auto">
        {showDetail ? (
          <Outlet />
        ) : active ? (
          <Suspense fallback={<ModuleLoadingFallback variant={active.variant} />}>
            <active.Component />
          </Suspense>
        ) : (
          // Alle Bereiche abgeschaltet oder keine Berechtigung — die Leiste steht
          // noch, damit der Zustand nicht als Fehler wirkt.
          <div className="flex h-full items-center justify-center p-8 text-sm text-muted-foreground">
            {t('crm.nav.noSections')}
          </div>
        )}
      </div>
    </div>
  )
}
