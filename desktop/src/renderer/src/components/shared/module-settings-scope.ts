import { createContext, useContext } from 'react'
import type { SettingsScope } from '@/lib/module-settings'

/**
 * Scope context for ModuleSettingsShell sections. Lets custom (non-native)
 * controls inside a section read whether they are currently editable, in
 * addition to the native fieldset[disabled] guard.
 *
 * Kept in its own (component-free) module so ModuleSettingsShell.tsx exports
 * only components — required for React Fast Refresh.
 */
export interface ScopeContextValue {
  scope: SettingsScope
  editable: boolean
}

export const ModuleSettingsScopeContext = createContext<ScopeContextValue>({
  scope: 'personal',
  editable: true,
})

/** Read the scope + editability of the surrounding ModuleSettingsShell section. */
export const useSettingsSectionScope = (): ScopeContextValue => useContext(ModuleSettingsScopeContext)
