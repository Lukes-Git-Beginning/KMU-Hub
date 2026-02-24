/**
 * Header widget registry — defines available mini-widgets for the header bar.
 */
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
    name: 'Naechster Termin',
    description: 'Zeigt den naechsten Kalender-Termin an.',
    icon: Calendar,
    component: NextMeetingWidget,
  },
  weather: {
    id: 'weather',
    name: 'Wetter',
    description: 'Aktuelle Temperatur und Wetterlage.',
    icon: CloudSun,
    component: WeatherWidget,
  },
  pomodoro: {
    id: 'pomodoro',
    name: 'Pomodoro',
    description: 'Fokus-Timer mit 25/5 Minuten Rhythmus.',
    icon: Timer,
    component: PomodoroWidget,
  },
  'unread-count': {
    id: 'unread-count',
    name: 'Ungelesen',
    description: 'Zeigt die Anzahl ungelesener Nachrichten.',
    icon: MessageSquare,
    component: UnreadCountWidget,
  },
  'quick-note': {
    id: 'quick-note',
    name: 'Kurznotiz',
    description: 'Schnelle Notiz oder Erinnerung.',
    icon: StickyNote,
    component: QuickNoteWidget,
  },
}

export const headerWidgetList: HeaderWidgetDef[] = Object.values(headerWidgetRegistry)
