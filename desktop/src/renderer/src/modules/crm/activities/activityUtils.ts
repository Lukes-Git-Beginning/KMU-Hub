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
  call: 'crm.activities.type.call',
  meeting: 'crm.activities.type.meeting',
  note: 'crm.activities.type.note',
  email: 'crm.activities.type.email',
  task: 'crm.activities.type.task',
}

const typeIcons: Record<ActivityType, LucideIcon> = {
  call: Phone,
  meeting: Users,
  note: StickyNote,
  email: Mail,
  task: CheckSquare,
}

export function activityTypeLabel(type?: string): string {
  return typeLabels[(type as ActivityType)] ?? type ?? 'crm.activities.type.unknown'
}

export function activityTypeIcon(type?: string): LucideIcon {
  return typeIcons[(type as ActivityType)] ?? StickyNote
}

export const ACTIVITY_TYPES: { value: ActivityType; labelKey: string }[] = [
  { value: 'call', labelKey: 'crm.activities.type.call' },
  { value: 'meeting', labelKey: 'crm.activities.type.meeting' },
  { value: 'note', labelKey: 'crm.activities.type.note' },
  { value: 'email', labelKey: 'crm.activities.type.email' },
  { value: 'task', labelKey: 'crm.activities.type.task' },
]
