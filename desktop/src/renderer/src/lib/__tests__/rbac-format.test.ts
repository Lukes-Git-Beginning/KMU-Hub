/**
 * Role labels must resolve against the real wire, not only the demo mock: the
 * server sends a system role's UUID as `id` and its preset key as `name`, the
 * mock keys presets by `id`. Before rolePresetKey the admin surfaces built
 * `rbac.roles.<uuid>.label` against the server.
 */
import { describe, it, expect } from 'vitest'
import type { TFunction } from 'i18next'
import { roleDescription, roleDisplayName, rolePresetKey } from '../rbac-format'

const t = ((key: string) => `t:${key}`) as unknown as TFunction

const wireMember = {
  id: '7c9e6679-7425-40de-944b-e07fc1f90ae7',
  name: 'member',
  isSystem: true,
  description: 'Standard access',
}
const mockMember = { id: 'member', name: 'Member', isSystem: true, description: '' }
const custom = { id: 'a1b2c3d4-0000-4000-8000-000000000001', name: 'Lager', isSystem: false, description: 'Eigene Rolle' }

describe('rolePresetKey', () => {
  it('reads the preset key from name on the wire and from id in the mock', () => {
    expect(rolePresetKey(wireMember)).toBe('member')
    expect(rolePresetKey(mockMember)).toBe('member')
  })

  it('is null for a custom role', () => {
    expect(rolePresetKey(custom)).toBeNull()
  })
})

describe('roleDisplayName / roleDescription', () => {
  it('uses the preset i18n keys for a system role from the server', () => {
    expect(roleDisplayName(t, wireMember)).toBe('t:rbac.roles.member.label')
    expect(roleDescription(t, wireMember)).toBe('t:rbac.roles.member.description')
  })

  it('keeps a custom role on its own name and description', () => {
    expect(roleDisplayName(t, custom)).toBe('Lager')
    expect(roleDescription(t, custom)).toBe('Eigene Rolle')
  })
})
