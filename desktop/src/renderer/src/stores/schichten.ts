import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface ShiftTemplate {
  id: string
  name: string
  startTime: string
  endTime: string
  color: string
  breakMinutes: number
}

export interface ShiftAssignment {
  id: string
  userId: string
  userName: string
  templateId: string
  templateName: string
  date: string
  status: 'scheduled' | 'confirmed' | 'in-progress' | 'completed' | 'absent'
}

export interface ShiftSwapRequest {
  id: string
  assignmentId: string
  requestedBy: string
  swapWithUser: string
  status: 'pending' | 'approved' | 'rejected'
  reason: string
  createdAt: string
}

interface SchichtenStore {
  templates: ShiftTemplate[]
  assignments: ShiftAssignment[]
  swapRequests: ShiftSwapRequest[]
}

const MOCK_TEMPLATES: ShiftTemplate[] = [
  { id: 'tpl-1', name: 'Frühschicht', startTime: '06:00', endTime: '14:00', color: '#f59e0b', breakMinutes: 30 },
  { id: 'tpl-2', name: 'Spätschicht', startTime: '14:00', endTime: '22:00', color: '#3b82f6', breakMinutes: 30 },
  { id: 'tpl-3', name: 'Nachtschicht', startTime: '22:00', endTime: '06:00', color: '#8b5cf6', breakMinutes: 45 },
]

const MOCK_ASSIGNMENTS: ShiftAssignment[] = [
  { id: 'sa-1', userId: 'u-1', userName: 'Markus Baumann', templateId: 'tpl-1', templateName: 'Frühschicht', date: '2026-02-16', status: 'scheduled' },
  { id: 'sa-2', userId: 'u-2', userName: 'Sandra Wyss', templateId: 'tpl-2', templateName: 'Spätschicht', date: '2026-02-16', status: 'confirmed' },
  { id: 'sa-3', userId: 'u-3', userName: 'Thomas Gerber', templateId: 'tpl-3', templateName: 'Nachtschicht', date: '2026-02-16', status: 'scheduled' },
  { id: 'sa-4', userId: 'u-4', userName: 'Petra Huber', templateId: 'tpl-1', templateName: 'Frühschicht', date: '2026-02-16', status: 'confirmed' },
  { id: 'sa-5', userId: 'u-5', userName: 'Reto Schmid', templateId: 'tpl-2', templateName: 'Spätschicht', date: '2026-02-17', status: 'scheduled' },
  { id: 'sa-6', userId: 'u-6', userName: 'Andrea Keller', templateId: 'tpl-1', templateName: 'Frühschicht', date: '2026-02-17', status: 'scheduled' },
  { id: 'sa-7', userId: 'u-7', userName: 'Daniel Frei', templateId: 'tpl-3', templateName: 'Nachtschicht', date: '2026-02-17', status: 'scheduled' },
  { id: 'sa-8', userId: 'u-8', userName: 'Lisa Brunner', templateId: 'tpl-2', templateName: 'Spätschicht', date: '2026-02-18', status: 'scheduled' },
  { id: 'sa-9', userId: 'u-1', userName: 'Markus Baumann', templateId: 'tpl-1', templateName: 'Frühschicht', date: '2026-02-18', status: 'scheduled' },
  { id: 'sa-10', userId: 'u-3', userName: 'Thomas Gerber', templateId: 'tpl-2', templateName: 'Spätschicht', date: '2026-02-18', status: 'scheduled' },
  { id: 'sa-11', userId: 'u-5', userName: 'Reto Schmid', templateId: 'tpl-3', templateName: 'Nachtschicht', date: '2026-02-18', status: 'scheduled' },
  { id: 'sa-12', userId: 'u-2', userName: 'Sandra Wyss', templateId: 'tpl-1', templateName: 'Frühschicht', date: '2026-02-19', status: 'scheduled' },
  { id: 'sa-13', userId: 'u-4', userName: 'Petra Huber', templateId: 'tpl-2', templateName: 'Spätschicht', date: '2026-02-19', status: 'scheduled' },
  { id: 'sa-14', userId: 'u-6', userName: 'Andrea Keller', templateId: 'tpl-3', templateName: 'Nachtschicht', date: '2026-02-19', status: 'scheduled' },
  { id: 'sa-15', userId: 'u-7', userName: 'Daniel Frei', templateId: 'tpl-1', templateName: 'Frühschicht', date: '2026-02-20', status: 'scheduled' },
  { id: 'sa-16', userId: 'u-8', userName: 'Lisa Brunner', templateId: 'tpl-2', templateName: 'Spätschicht', date: '2026-02-20', status: 'scheduled' },
]

const MOCK_SWAP_REQUESTS: ShiftSwapRequest[] = [
  { id: 'swap-1', assignmentId: 'sa-5', requestedBy: 'Reto Schmid', swapWithUser: 'Andrea Keller', status: 'pending', reason: 'Arzttermin am Nachmittag', createdAt: '2026-02-14T10:30:00' },
  { id: 'swap-2', assignmentId: 'sa-8', requestedBy: 'Lisa Brunner', swapWithUser: 'Thomas Gerber', status: 'approved', reason: 'Familienfeier am Abend', createdAt: '2026-02-13T16:15:00' },
]

export const useSchichtenStore = create<SchichtenStore>()(
  persist(
    () => ({
      templates: MOCK_TEMPLATES,
      assignments: MOCK_ASSIGNMENTS,
      swapRequests: MOCK_SWAP_REQUESTS,
    }),
    { name: 'kmuhub-schichten' },
  ),
)
