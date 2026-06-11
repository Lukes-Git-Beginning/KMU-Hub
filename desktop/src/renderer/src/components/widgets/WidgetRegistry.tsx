/**
 * Widget registry -- definitions and metadata for all dashboard widgets.
 *
 * Each widget is lazy-loaded so the dashboard only bundles the
 * widgets the user actually has active.
 */
import { lazy, type LazyExoticComponent, type ComponentType } from 'react'
import i18next from 'i18next'
import {
  Users,
  TrendingUp,
  MessageSquare,
  Activity,
  Zap,
  Bell,
  Euro,
  CheckCircle2,
  Handshake,
  Calendar,
  UserCheck,
  BarChart3,
  ClipboardList,
  CalendarDays,
  Clock,
  MessageCircle,
  UserX,
  Cake,
  LayoutDashboard,
  Clock4,
  TicketCheck,
  type LucideIcon,
} from 'lucide-react'
import type { ModuleId } from '@/lib/pricing'

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
  /**
   * Module that must be enabled (feature flag `modules.<module>`) for this
   * widget to be available. Undefined = always available (no module gate).
   *
   * NOTE: This is a static data field — do NOT evaluate i18next.t() here at
   * module-load time (Registry is loaded during boot, before i18n is ready,
   * which silently empties all widget names app-wide).
   */
  module?: ModuleId
}

/** All available widget definitions, keyed by ID. */
export const widgetRegistry: Record<string, WidgetDefinition> = {
  'recent-contacts': {
    id: 'recent-contacts',
    name: i18next.t('widgets.registry.recentContacts.name'),
    description: i18next.t('widgets.registry.recentContacts.description'),
    icon: Users,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 2, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/RecentContacts')),
    roles: ['admin', 'manager', 'member'],
    module: 'crm',
  },
  'deal-pipeline': {
    id: 'deal-pipeline',
    name: i18next.t('widgets.registry.dealPipeline.name'),
    description: i18next.t('widgets.registry.dealPipeline.description'),
    icon: TrendingUp,
    defaultSize: { w: 4, h: 4 },
    minSize: { w: 4, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/DealPipeline')),
    roles: ['admin', 'manager'],
    module: 'crm',
  },
  'unread-messages': {
    id: 'unread-messages',
    name: i18next.t('widgets.registry.unreadMessages.name'),
    description: i18next.t('widgets.registry.unreadMessages.description'),
    icon: MessageSquare,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 2, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/UnreadMessages')),
    roles: ['admin', 'manager', 'member'],
    module: 'chat',
  },
  'activity-feed': {
    id: 'activity-feed',
    name: i18next.t('widgets.registry.activityFeed.name'),
    description: i18next.t('widgets.registry.activityFeed.description'),
    icon: Activity,
    defaultSize: { w: 4, h: 4 },
    minSize: { w: 3, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/ActivityFeed')),
    roles: ['admin', 'manager', 'member'],
    // No module gate: activity feed aggregates cross-module events
  },
  'quick-actions': {
    id: 'quick-actions',
    name: i18next.t('widgets.registry.quickActions.name'),
    description: i18next.t('widgets.registry.quickActions.description'),
    icon: Zap,
    defaultSize: { w: 4, h: 2 },
    minSize: { w: 2, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/QuickActions')),
    roles: ['admin', 'manager', 'member'],
    // No module gate: quick actions adapts to available modules at runtime
  },
  'notification-summary': {
    id: 'notification-summary',
    name: i18next.t('widgets.registry.notificationSummary.name'),
    description: i18next.t('widgets.registry.notificationSummary.description'),
    icon: Bell,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 2, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/NotificationSummary')),
    roles: ['admin', 'manager', 'member'],
    // No module gate: notifications are system-wide
  },
  'notification-feed': {
    id: 'notification-feed',
    name: i18next.t('widgets.registry.notificationFeed.name'),
    description: i18next.t('widgets.registry.notificationFeed.description'),
    icon: Bell,
    defaultSize: { w: 4, h: 6 },
    minSize: { w: 3, h: 4 },
    component: lazy(() => import('../../modules/dashboard/widgets/NotificationFeedWidget')),
    roles: ['admin', 'manager', 'member'],
    // No module gate: notifications are system-wide
  },
  'kpi-revenue': {
    id: 'kpi-revenue',
    name: i18next.t('widgets.registry.kpiRevenue.name'),
    description: i18next.t('widgets.registry.kpiRevenue.description'),
    icon: Euro,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 3, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/KpiRevenue')),
    roles: ['admin', 'manager'],
    module: 'finance',
  },
  'kpi-tasks': {
    id: 'kpi-tasks',
    name: i18next.t('widgets.registry.kpiTasks.name'),
    description: i18next.t('widgets.registry.kpiTasks.description'),
    icon: CheckCircle2,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 3, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/KpiTasks')),
    roles: ['admin', 'manager', 'member'],
    module: 'tasks',
  },
  'kpi-deals': {
    id: 'kpi-deals',
    name: i18next.t('widgets.registry.kpiDeals.name'),
    description: i18next.t('widgets.registry.kpiDeals.description'),
    icon: Handshake,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 3, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/KpiDeals')),
    roles: ['admin', 'manager'],
    module: 'crm',
  },
  'calendar-upcoming': {
    id: 'calendar-upcoming',
    name: i18next.t('widgets.registry.calendarUpcoming.name'),
    description: i18next.t('widgets.registry.calendarUpcoming.description'),
    icon: Calendar,
    defaultSize: { w: 4, h: 4 },
    minSize: { w: 3, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/CalendarUpcoming')),
    roles: ['admin', 'manager', 'member'],
    module: 'calendar',
  },
  'team-status': {
    id: 'team-status',
    name: i18next.t('widgets.registry.teamStatus.name'),
    description: i18next.t('widgets.registry.teamStatus.description'),
    icon: UserCheck,
    defaultSize: { w: 4, h: 4 },
    minSize: { w: 3, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/TeamStatus')),
    roles: ['admin', 'manager', 'member'],
    module: 'team',
  },
  'mini-chart': {
    id: 'mini-chart',
    name: i18next.t('widgets.registry.miniChart.name'),
    description: i18next.t('widgets.registry.miniChart.description'),
    icon: BarChart3,
    defaultSize: { w: 6, h: 3 },
    minSize: { w: 4, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/MiniChart')),
    roles: ['admin', 'manager'],
    // No module gate: mini-chart is a generic visualization utility
  },
  'my-tasks': {
    id: 'my-tasks',
    name: i18next.t('widgets.registry.myTasks.name'),
    description: i18next.t('widgets.registry.myTasks.description'),
    icon: ClipboardList,
    defaultSize: { w: 4, h: 4 },
    minSize: { w: 3, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/MyTasks')),
    roles: ['admin', 'manager', 'member'],
    module: 'tasks',
  },
  'my-calendar': {
    id: 'my-calendar',
    name: i18next.t('widgets.registry.myCalendar.name'),
    description: i18next.t('widgets.registry.myCalendar.description'),
    icon: CalendarDays,
    defaultSize: { w: 4, h: 4 },
    minSize: { w: 3, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/MyCalendar')),
    roles: ['admin', 'manager', 'member'],
    module: 'calendar',
  },
  'time-clock': {
    id: 'time-clock',
    name: i18next.t('widgets.registry.timeClock.name'),
    description: i18next.t('widgets.registry.timeClock.description'),
    icon: Clock,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 3, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/TimeClockWidget')),
    roles: ['admin', 'manager', 'member'],
    module: 'zeiterfassung',
  },
  'team-chat': {
    id: 'team-chat',
    name: i18next.t('widgets.registry.teamChat.name'),
    description: i18next.t('widgets.registry.teamChat.description'),
    icon: MessageCircle,
    defaultSize: { w: 4, h: 4 },
    minSize: { w: 3, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/TeamChat')),
    roles: ['admin', 'manager', 'member'],
    module: 'chat',
  },
  absences: {
    id: 'absences',
    name: i18next.t('widgets.registry.absences.name'),
    description: i18next.t('widgets.registry.absences.description'),
    icon: UserX,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 3, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/Absences')),
    roles: ['admin', 'manager', 'member'],
    module: 'team',
  },
  birthdays: {
    id: 'birthdays',
    name: i18next.t('widgets.registry.birthdays.name'),
    description: i18next.t('widgets.registry.birthdays.description'),
    icon: Cake,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 3, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/Birthdays')),
    roles: ['admin', 'manager', 'member'],
    module: 'team',
  },
  'cross-module-overview': {
    id: 'cross-module-overview',
    name: i18next.t('widgets.registry.crossModuleOverview.name'),
    description: i18next.t('widgets.registry.crossModuleOverview.description'),
    icon: LayoutDashboard,
    defaultSize: { w: 6, h: 4 },
    minSize: { w: 4, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/CrossModuleOverview')),
    roles: ['admin', 'manager', 'member'],
    // No module gate: always available; individual data sources are flag-gated internally
  },
  'team-worktime': {
    id: 'team-worktime',
    name: i18next.t('widgets.registry.teamWorktime.name'),
    description: i18next.t('widgets.registry.teamWorktime.description'),
    icon: Clock4,
    defaultSize: { w: 4, h: 4 },
    minSize: { w: 3, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/TeamWorktime')),
    roles: ['admin', 'manager'],
    module: 'zeiterfassung',
  },
  'open-tickets': {
    id: 'open-tickets',
    name: i18next.t('widgets.registry.openTickets.name'),
    description: i18next.t('widgets.registry.openTickets.description'),
    icon: TicketCheck,
    defaultSize: { w: 4, h: 4 },
    minSize: { w: 3, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/OpenTickets')),
    roles: ['admin', 'manager', 'member'],
    module: 'helpdesk',
  },
}

/** Ordered list of all widget definitions for the widget picker. */
export const widgetList: WidgetDefinition[] = Object.values(widgetRegistry)
