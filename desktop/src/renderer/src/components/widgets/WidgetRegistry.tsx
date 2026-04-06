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
    name: i18next.t('widgets.registry.recentContacts.name'),
    description: i18next.t('widgets.registry.recentContacts.description'),
    icon: Users,
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 2, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/RecentContacts')),
    roles: ['admin', 'manager', 'member'],
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
  },
}

/** Ordered list of all widget definitions for the widget picker. */
export const widgetList: WidgetDefinition[] = Object.values(widgetRegistry)
