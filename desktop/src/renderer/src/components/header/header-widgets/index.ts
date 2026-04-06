/**
 * Header widget registry — defines available mini-widgets for the header bar.
 */
import i18next from 'i18next'
import type { ComponentType } from 'react'
import type { LucideIcon } from 'lucide-react'
import {
  Calendar,
  CloudSun,
  Timer,
  MessageSquare,
  StickyNote,
} from 'lucide-react'
import { NextMeetingWidget } from './NextMeetingWidget'
import { WeatherWidget } from './WeatherWidget'
import { PomodoroWidget } from './PomodoroWidget'
import { UnreadCountWidget } from './UnreadCountWidget'
import { QuickNoteWidget } from './QuickNoteWidget'

export interface HeaderWidgetDef {
  id: string
  name: string
  description: string
  icon: LucideIcon
  component: ComponentType
}

export const headerWidgetRegistry: Record<string, HeaderWidgetDef> = {
  'next-meeting': {
    id: 'next-meeting',
    name: i18next.t('header.widgetRegistry.nextMeeting.name'),
    description: i18next.t('header.widgetRegistry.nextMeeting.description'),
    icon: Calendar,
    component: NextMeetingWidget,
  },
  weather: {
    id: 'weather',
    name: i18next.t('header.widgetRegistry.weather.name'),
    description: i18next.t('header.widgetRegistry.weather.description'),
    icon: CloudSun,
    component: WeatherWidget,
  },
  pomodoro: {
    id: 'pomodoro',
    name: i18next.t('header.widgetRegistry.pomodoro.name'),
    description: i18next.t('header.widgetRegistry.pomodoro.description'),
    icon: Timer,
    component: PomodoroWidget,
  },
  'unread-count': {
    id: 'unread-count',
    name: i18next.t('header.widgetRegistry.unreadCount.name'),
    description: i18next.t('header.widgetRegistry.unreadCount.description'),
    icon: MessageSquare,
    component: UnreadCountWidget,
  },
  'quick-note': {
    id: 'quick-note',
    name: i18next.t('header.widgetRegistry.quickNote.name'),
    description: i18next.t('header.widgetRegistry.quickNote.description'),
    icon: StickyNote,
    component: QuickNoteWidget,
  },
}

export const headerWidgetList: HeaderWidgetDef[] = Object.values(headerWidgetRegistry)
