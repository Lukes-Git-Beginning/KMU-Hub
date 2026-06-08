import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Task → label assignments (mock-first).
 *
 * Labels themselves (name + colour) live in the tenant work-settings taxonomy
 * (stores/workSettings). This store only maps a task to the label ids assigned
 * to it. The backend has no structured task labels yet (tasks carry free-text
 * `tags`), so assignments are persisted locally — backend gap in backend-gaps.md.
 */
interface TaskLabelsState {
  /** task_id → label_id[] */
  byTask: Record<string, string[]>
  setLabels: (taskId: string, labelIds: string[]) => void
  toggleLabel: (taskId: string, labelId: string) => void
  /** Drop a label id from every task (called when a label is deleted). */
  purgeLabel: (labelId: string) => void
}

// Seed a few tasks so chips are visible across the demo board.
const SEED: Record<string, string[]> = {
  'tsk-001': ['wl-design'],
  'tsk-002': ['wl-feature', 'wl-blocked'],
  'tsk-003': ['wl-bug'],
  'tsk-004': ['wl-feature', 'wl-design'],
  'tsk-006': ['wl-feature'],
  'tsk-008': ['wl-bug'],
  'tsk-033': ['wl-blocked'],
}

export const useTaskLabelsStore = create<TaskLabelsState>()(
  persist(
    (set) => ({
      byTask: SEED,
      setLabels: (taskId, labelIds) =>
        set((s) => ({ byTask: { ...s.byTask, [taskId]: labelIds } })),
      toggleLabel: (taskId, labelId) =>
        set((s) => {
          const current = s.byTask[taskId] ?? []
          const next = current.includes(labelId)
            ? current.filter((id) => id !== labelId)
            : [...current, labelId]
          return { byTask: { ...s.byTask, [taskId]: next } }
        }),
      purgeLabel: (labelId) =>
        set((s) => {
          const byTask: Record<string, string[]> = {}
          for (const [taskId, ids] of Object.entries(s.byTask)) {
            byTask[taskId] = ids.filter((id) => id !== labelId)
          }
          return { byTask }
        }),
    }),
    { name: 'cosmi-task-labels' },
  ),
)
