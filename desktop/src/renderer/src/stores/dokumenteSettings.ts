import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Tenant-wide dokumente settings — administered by a module lead via the
 * "Für alle" section of the dokumente settings panel.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/dokumente/tenant):
 *   - `initFromServer()` once per session hydrates the store from the backend
 *     (call on panel mount); falls back to the local defaults when nothing is
 *     stored yet.
 *   - Each setter writes through to the tenant-scope endpoint (Modul-Leiter /
 *     admin only; the panel gates the UI). localStorage stays as the optimistic
 *     cache so the UI is instant and survives offline.
 */

const MODULE_ID = 'dokumente'

/** Lazily PUT the tenant-scope settings; failures are non-critical (next
 *  initFromServer re-syncs). Only the persisted keys are sent. */
async function persistTenant(settings: Record<string, unknown>): Promise<void> {
  try {
    const { authenticatedRequest } = await import('@/api/utils/authenticatedFetch')
    await authenticatedRequest({
      method: 'PUT',
      path: `/api/v1/settings/${MODULE_ID}/tenant`,
      body: { settings },
    })
  } catch {
    // Silent — store already updated; re-syncs on next session.
  }
}

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
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean

  toggleFileTypeGroup: (group: DokumenteFileTypeGroup) => void
  setDefaultShareScope: (scope: DokumenteShareScope) => void
  setOnlyOfficeEnabled: (enabled: boolean) => void
  setTrashRetentionDays: (days: number) => void
  /** Hydrate from GET /settings/dokumente/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The four tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: DokumenteSettingsState): Record<string, unknown> {
  return {
    allowedFileTypeGroups: s.allowedFileTypeGroups,
    defaultShareScope: s.defaultShareScope,
    onlyOfficeEnabled: s.onlyOfficeEnabled,
    trashRetentionDays: s.trashRetentionDays,
  }
}

export const useDokumenteSettingsStore = create<DokumenteSettingsState>()(
  persist(
    (set, get) => ({
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
      serverInitialized: false,
      toggleFileTypeGroup: (group) => {
        set((s) => ({
          allowedFileTypeGroups: s.allowedFileTypeGroups.includes(group)
            ? s.allowedFileTypeGroups.filter((g) => g !== group)
            : [...s.allowedFileTypeGroups, group],
        }))
        void persistTenant(tenantPayload(get()))
      },
      setDefaultShareScope: (defaultShareScope) => {
        set({ defaultShareScope })
        void persistTenant(tenantPayload(get()))
      },
      setOnlyOfficeEnabled: (onlyOfficeEnabled) => {
        set({ onlyOfficeEnabled })
        void persistTenant(tenantPayload(get()))
      },
      setTrashRetentionDays: (trashRetentionDays) => {
        set({ trashRetentionDays })
        void persistTenant(tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        try {
          const { authenticatedRequest } = await import('@/api/utils/authenticatedFetch')
          type EntriesResp = { entries?: Array<{ key: string; value: unknown }> }
          const resp = await authenticatedRequest<EntriesResp>({
            method: 'GET',
            path: `/api/v1/settings/${MODULE_ID}/tenant`,
          })
          const map = Object.fromEntries((resp.entries ?? []).map((e) => [e.key, e.value]))
          set((s) => ({
            allowedFileTypeGroups: Array.isArray(map.allowedFileTypeGroups)
              ? (map.allowedFileTypeGroups as DokumenteFileTypeGroup[])
              : s.allowedFileTypeGroups,
            defaultShareScope:
              map.defaultShareScope === 'private' || map.defaultShareScope === 'team'
                ? map.defaultShareScope
                : s.defaultShareScope,
            onlyOfficeEnabled:
              typeof map.onlyOfficeEnabled === 'boolean' ? map.onlyOfficeEnabled : s.onlyOfficeEnabled,
            trashRetentionDays:
              typeof map.trashRetentionDays === 'number' ? map.trashRetentionDays : s.trashRetentionDays,
            serverInitialized: true,
          }))
        } catch {
          // Offline / no backend — keep local defaults; retry next session.
          set({ serverInitialized: true })
        }
      },
    }),
    { name: 'cosmi-dokumente-settings' },
  ),
)
