import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation } from 'react-router-dom'
import { X } from 'lucide-react'
import { useUIStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { userHasRole } from '@/config/roles'
import {
  SETTINGS_ENTRIES,
  resolveEntryForPath,
  type SettingsEntry,
  type SettingsGroup,
} from './module-settings-registry'

const GROUP_ORDER: SettingsGroup[] = ['module', 'cosmi']
const GROUP_LABEL: Record<SettingsGroup, string> = {
  module: 'moduleSettings.groups.module',
  cosmi: 'moduleSettings.groups.cosmi',
}

/**
 * SettingsOverlay — the Module-Settings window.
 *
 * Opens as an overlay (no routing) from the bottom-left "Einstellungen" button.
 * Left column lists the modules + a "Cosmi (Allgemein)" group; the right side
 * shows the selected entry's settings. Context-aware: preselects the entry of
 * the currently active module.
 */
export function SettingsOverlay() {
  const { t } = useTranslation()
  const location = useLocation()
  const user = useAuthStore((s) => s.user)

  const isOpen = useUIStore((s) => s.isSettingsOverlayOpen)
  const requestedEntry = useUIStore((s) => s.settingsOverlayEntry)
  const close = useUIStore((s) => s.closeSettingsOverlay)

  // RBAC-filtered entries
  const entries = useMemo(
    () => SETTINGS_ENTRIES.filter((e) => !e.roles || userHasRole(user, e.roles)),
    [user],
  )

  // The entry matching the current route (for the "Aktiv" marker + preselect)
  const activeRouteEntry = resolveEntryForPath(location.pathname)

  const [selectedId, setSelectedId] = useState<string | undefined>()

  // When the overlay opens, pick the initial entry: explicit request → active
  // module → first available.
  useEffect(() => {
    if (!isOpen) return
    const initial =
      (requestedEntry && entries.some((e) => e.id === requestedEntry) ? requestedEntry : undefined) ??
      (activeRouteEntry && entries.some((e) => e.id === activeRouteEntry) ? activeRouteEntry : undefined) ??
      entries[0]?.id
    // eslint-disable-next-line react-hooks/set-state-in-effect -- reset selection when the overlay opens
    setSelectedId(initial)
  }, [isOpen, requestedEntry, activeRouteEntry, entries])

  // Esc to close
  useEffect(() => {
    if (!isOpen) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [isOpen, close])

  if (!isOpen) return null

  const selected: SettingsEntry | undefined = entries.find((e) => e.id === selectedId)
  const SelectedComponent = selected?.component

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={t('moduleSettings.title')}
      className="fixed inset-0 z-[120] flex items-center justify-center bg-black/40 p-4 animate-fade-in"
      onClick={(e) => {
        if (e.target === e.currentTarget) close()
      }}
    >
      <div className="flex h-[90vh] w-full max-w-7xl overflow-hidden rounded-2xl border border-border bg-card shadow-2xl animate-scale-in">
        {/* ── Left nav ── */}
        <aside className="flex w-64 shrink-0 flex-col border-r border-border bg-secondary/20">
          <div className="px-5 pb-2 pt-5">
            <h2 className="text-sm font-semibold text-foreground">{t('moduleSettings.title')}</h2>
          </div>
          <nav className="flex-1 space-y-5 overflow-y-auto px-3 pb-5 pt-2">
            {GROUP_ORDER.map((group) => {
              const groupEntries = entries.filter((e) => e.group === group)
              if (groupEntries.length === 0) return null
              return (
                <div key={group}>
                  <p className="mb-1 px-3 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                    {t(GROUP_LABEL[group])}
                  </p>
                  <div className="space-y-0.5">
                    {groupEntries.map((entry) => {
                      const Icon = entry.icon
                      const isSelected = entry.id === selectedId
                      const isActiveModule = entry.id === activeRouteEntry
                      return (
                        <button
                          key={entry.id}
                          onClick={() => setSelectedId(entry.id)}
                          className={`flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm transition-colors ${
                            isSelected
                              ? 'bg-primary-light font-medium text-primary'
                              : 'text-foreground hover:bg-secondary'
                          }`}
                        >
                          <Icon className="h-4 w-4 shrink-0" />
                          <span className="truncate">{t(entry.labelKey)}</span>
                          {isActiveModule && (
                            <span className="ml-auto shrink-0 rounded-full bg-primary/15 px-1.5 py-0.5 text-[9px] font-medium uppercase tracking-wide text-primary">
                              {t('moduleSettings.activeBadge')}
                            </span>
                          )}
                        </button>
                      )
                    })}
                  </div>
                </div>
              )
            })}
          </nav>
        </aside>

        {/* ── Content ── */}
        <div className="relative flex-1 overflow-y-auto">
          <button
            onClick={close}
            aria-label={t('common.close', { defaultValue: 'Schließen' })}
            className="absolute right-4 top-4 z-10 flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
          <div className="p-6">
            {SelectedComponent ? (
              <SelectedComponent />
            ) : (
              <p className="text-sm text-muted-foreground">{t('moduleSettings.empty')}</p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
