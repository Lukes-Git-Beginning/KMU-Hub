/**
 * Widget registry -- definitions and metadata for all dashboard widgets.
 *
 * Each widget is lazy-loaded so the dashboard only bundles the
 * widgets the user actually has active.
 */
import { lazy, type LazyExoticComponent, type ComponentType } from 'react'
import {
  Users,
  TrendingUp,
  MessageSquare,
  Activity,
  Zap,
  Bell,
  type LucideIcon,
} from 'lucide-react'

/** Props every widget component receives. */
export interface WidgetProps {
  widgetId: string
  isEditing: boolean
}

/** Full metadata for a registered widget. */
export interface WidgetDefinition {
  id: string
  /** Display name (German). */
  name: string
  /** Short description for the widget picker. */
  description: string
  /** Lucide icon component. */
  icon: LucideIcon
  /** Default grid size (w x h in grid units). */
  defaultSize: { w: number; h: number }
  /** Minimum grid size. */
  minSize: { w: number; h: number }
  /** Optional maximum grid size. */
  maxSize?: { w: number; h: number }
  /** Lazy-loaded widget component. */
  component: LazyExoticComponent<ComponentType<WidgetProps>>
  /** Roles that see this widget by default. */
  roles: string[]
}

/** All available widget definitions, keyed by ID. */
export const widgetRegistry: Record<string, WidgetDefinition> = {
  'recent-contacts': {
    id: 'recent-contacts',
    name: 'Letzte Kontakte',
    description: 'Die zuletzt aktualisierten Kontakte.',
    icon: Users,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 2, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/RecentContacts')),
    roles: ['admin', 'manager', 'member'],
  },
  'deal-pipeline': {
    id: 'deal-pipeline',
    name: 'Deal Pipeline',
    description: 'Kurzuebersicht der Deals pro Phase.',
    icon: TrendingUp,
    defaultSize: { w: 4, h: 4 },
    minSize: { w: 4, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/DealPipeline')),
    roles: ['admin', 'manager'],
  },
  'unread-messages': {
    id: 'unread-messages',
    name: 'Ungelesene Nachrichten',
    description: 'Kanaele und DMs mit ungelesenen Nachrichten.',
    icon: MessageSquare,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 2, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/UnreadMessages')),
    roles: ['admin', 'manager', 'member'],
  },
  'activity-feed': {
    id: 'activity-feed',
    name: 'Aktivitaeten',
    description: 'Letzte CRM-Aktivitaeten im Ueberblick.',
    icon: Activity,
    defaultSize: { w: 4, h: 4 },
    minSize: { w: 3, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/ActivityFeed')),
    roles: ['admin', 'manager', 'member'],
  },
  'quick-actions': {
    id: 'quick-actions',
    name: 'Schnellaktionen',
    description: 'Schnellzugriff auf haeufige Aktionen.',
    icon: Zap,
    defaultSize: { w: 4, h: 2 },
    minSize: { w: 2, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/QuickActions')),
    roles: ['admin', 'manager', 'member'],
  },
  'notification-summary': {
    id: 'notification-summary',
    name: 'Benachrichtigungen',
    description: 'Aktuelle Benachrichtigungen auf einen Blick.',
    icon: Bell,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 2, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/NotificationSummary')),
    roles: ['admin', 'manager', 'member'],
  },
}

/** Ordered list of all widget definitions for the widget picker. */
export const widgetList: WidgetDefinition[] = Object.values(widgetRegistry)
