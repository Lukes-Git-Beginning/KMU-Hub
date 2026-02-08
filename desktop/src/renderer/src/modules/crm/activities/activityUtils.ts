/**
 * Shared utility functions for activity type display.
 *
 * Provides German labels and icons for each activity type.
 */
import {
  Phone,
  Users,
  StickyNote,
  Mail,
  CheckSquare,
  type LucideIcon,
} from 'lucide-react'

type ActivityType = 'call' | 'meeting' | 'note' | 'email' | 'task'

const typeLabels: Record<ActivityType, string> = {
  call: 'Anruf',
  meeting: 'Meeting',
  note: 'Notiz',
  email: 'E-Mail',
  task: 'Aufgabe',
}

const typeIcons: Record<ActivityType, LucideIcon> = {
  call: Phone,
  meeting: Users,
  note: StickyNote,
  email: Mail,
  task: CheckSquare,
}

export function activityTypeLabel(type?: string): string {
  return typeLabels[(type as ActivityType)] ?? type ?? 'Unbekannt'
}

export function activityTypeIcon(type?: string): LucideIcon {
  return typeIcons[(type as ActivityType)] ?? StickyNote
}

export const ACTIVITY_TYPES: { value: ActivityType; label: string }[] = [
  { value: 'call', label: 'Anruf' },
  { value: 'meeting', label: 'Meeting' },
  { value: 'note', label: 'Notiz' },
  { value: 'email', label: 'E-Mail' },
  { value: 'task', label: 'Aufgabe' },
]
