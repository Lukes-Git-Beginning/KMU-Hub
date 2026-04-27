/**
 * useTenantMode — reads and sets the tenant mode (centralized | distributed).
 *
 * Mode is stored in authStore.tenant.mode and persisted to localStorage.
 * Components that consume this hook re-render automatically when mode changes.
 *
 * centralized: All admin settings visible inside Cosmi → Einstellungen.
 * distributed: Admin settings in separate /admin/* hub; Settings = personal only.
 */
import { useAuthStore } from '@/stores/auth'
import type { TenantMode } from '@/stores/auth'

export interface UseTenantModeReturn {
  mode: TenantMode
  isCentralized: boolean
  isDistributed: boolean
  setMode: (mode: TenantMode) => void
}

export function useTenantMode(): UseTenantModeReturn {
  const mode = useAuthStore((s) => s.tenant.mode)
  const setTenantMode = useAuthStore((s) => s.setTenantMode)

  return {
    mode,
    isCentralized: mode === 'centralized',
    isDistributed: mode === 'distributed',
    setMode: setTenantMode,
  }
}
