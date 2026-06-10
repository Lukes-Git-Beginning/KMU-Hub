import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Tenant-wide dokumente settings — administered by a module lead via the
 * "Für alle" section of the dokumente settings panel. Mock-first: persisted
 * locally until the backend provides a document-settings API (settings
 * foundation landed on main; wiring tracked in .planning/backend-gaps.md).
 */

/** File-type groups a tenant may allow for upload (extension buckets). */
export type DokumenteFileTypeGroup =
  | 'documents'
  | 'spreadsheets'
  | 'presentations'
  | 'images'
  | 'media'
  | 'archives'
  | 'other'

export const FILE_TYPE_GROUP_EXTENSIONS: Record<DokumenteFileTypeGroup, string[]> = {
  documents: ['pdf', 'doc', 'docx', 'odt', 'txt', 'md'],
  spreadsheets: ['xls', 'xlsx', 'ods', 'csv'],
  presentations: ['ppt', 'pptx', 'odp'],
  images: ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp'],
  media: ['mp3', 'mp4', 'wav', 'mov', 'webm'],
  archives: ['zip', 'rar', '7z', 'tar', 'gz'],
  other: [],
}

export type DokumenteShareScope = 'private' | 'team'

/** Storage quota per pricing tier (GB) — display-only, defined by the plan. */
export const STORAGE_QUOTA_BY_TIER_GB = [
  { tier: 'starter', quotaGb: 50 },
  { tier: 'business', quotaGb: 250 },
  { tier: 'enterprise', quotaGb: 1000 },
] as const

interface DokumenteSettingsState {
  /** File-type groups allowed for upload. */
  allowedFileTypeGroups: DokumenteFileTypeGroup[]
  /** Default visibility of newly uploaded files. */
  defaultShareScope: DokumenteShareScope
  /** Whether the OnlyOffice editor is offered for office files. */
  onlyOfficeEnabled: boolean
  /** Days a deleted file stays in the trash before purge. */
  trashRetentionDays: number

  toggleFileTypeGroup: (group: DokumenteFileTypeGroup) => void
  setDefaultShareScope: (scope: DokumenteShareScope) => void
  setOnlyOfficeEnabled: (enabled: boolean) => void
  setTrashRetentionDays: (days: number) => void
}

export const useDokumenteSettingsStore = create<DokumenteSettingsState>()(
  persist(
    (set) => ({
      allowedFileTypeGroups: [
        'documents',
        'spreadsheets',
        'presentations',
        'images',
        'media',
        'archives',
      ],
      defaultShareScope: 'private',
      onlyOfficeEnabled: true,
      trashRetentionDays: 30,
      toggleFileTypeGroup: (group) =>
        set((s) => ({
          allowedFileTypeGroups: s.allowedFileTypeGroups.includes(group)
            ? s.allowedFileTypeGroups.filter((g) => g !== group)
            : [...s.allowedFileTypeGroups, group],
        })),
      setDefaultShareScope: (defaultShareScope) => set({ defaultShareScope }),
      setOnlyOfficeEnabled: (onlyOfficeEnabled) => set({ onlyOfficeEnabled }),
      setTrashRetentionDays: (trashRetentionDays) => set({ trashRetentionDays }),
    }),
    { name: 'cosmi-dokumente-settings' },
  ),
)
