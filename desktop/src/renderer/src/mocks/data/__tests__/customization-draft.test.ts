/**
 * Modul-Editor v1 (E-1 Fundament) — draft overlay + lifecycle logic tests.
 *
 * Covers the pure data foundation (no React): the 4th `draft` overlay layer in
 * the resolver, promotion into the tenant layer, snapshot/rollback, and the
 * drafts store lifecycle (save → deploy now / scheduled → rollback).
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import {
  applyDraftToTenant,
  resolveLabelOverrides,
  resolveValueSet,
  restoreTenant,
  snapshotTenant,
  type TenantSnapshot,
} from '@/mocks/data/customization'
import {
  commitDraftNow,
  getDraft,
  rollbackDeploy,
  runDueScheduledDeploys,
  saveDraft,
  scheduleDraft,
} from '@/mocks/data/customization-drafts'
import {
  listCustomFields,
  resetCustomFieldsStore,
  type CustomFieldDefinition,
} from '@/mocks/data/custom-fields'
import type { CustomizationDraftPayload } from '@/api/customization-types'

// Keep every test isolated from the shared, seed-mutated tenant layer.
let pristine: TenantSnapshot
beforeEach(() => {
  pristine = snapshotTenant()
})
afterEach(() => {
  restoreTenant(pristine)
  resetCustomFieldsStore()
})

describe('resolveLabelOverrides — draft is the 4th layer', () => {
  it('draft wins over an existing tenant override', () => {
    // crm.deals.title is seeded as tenant='Aufträge'.
    const withoutDraft = resolveLabelOverrides('de')['crm.deals.title']
    expect(withoutDraft).toEqual({ key: 'crm.deals.title', value: 'Aufträge', provenance: 'tenant' })

    const draft = { de: { 'crm.deals.title': 'Baustellen' } }
    const withDraft = resolveLabelOverrides('de', false, draft)['crm.deals.title']
    expect(withDraft).toEqual({ key: 'crm.deals.title', value: 'Baustellen', provenance: 'draft' })
  })

  it('draft applies on top of a code default (no lower override)', () => {
    // work.tasks.title has no vendor/tenant seed → default.
    expect(resolveLabelOverrides('de')['work.tasks.title'].provenance).toBe('default')

    const draft = { de: { 'work.tasks.title': 'Aufgaben-Board' } }
    const entry = resolveLabelOverrides('de', false, draft)['work.tasks.title']
    expect(entry).toEqual({ key: 'work.tasks.title', value: 'Aufgaben-Board', provenance: 'draft' })
  })

  it('base=true ignores the draft overlay entirely', () => {
    const draft = { de: { 'crm.deals.title': 'Baustellen' } }
    expect(resolveLabelOverrides('de', true, draft)['crm.deals.title'].provenance).toBe('default')
  })
})

describe('resolveValueSet — draft is the 4th layer', () => {
  it('draft option and name win over tenant/default', () => {
    const draftOverlay = {
      deal_stages: {
        id: 'deal_stages',
        name: 'Sales-Pipeline (Entwurf)',
        options: [{ id: 'lead', label: 'Erstkontakt', order: 0, active: true }],
      },
    }
    const resolved = resolveValueSet('deal_stages', false, draftOverlay)
    expect(resolved?.name).toBe('Sales-Pipeline (Entwurf)')
    expect(resolved?.provenance).toBe('draft')
    const lead = resolved?.options.find((o) => o.id === 'lead')
    expect(lead).toMatchObject({ label: 'Erstkontakt', provenance: 'draft' })
    // Untouched options keep their lower-layer provenance — deal_stages is fully
    // covered by the vendor seed, so 'won' stays 'vendor' (only 'lead' is draft).
    expect(resolved?.options.find((o) => o.id === 'won')?.provenance).toBe('vendor')
  })
})

describe('draft lifecycle', () => {
  const payload: CustomizationDraftPayload = {
    labels: { de: { 'work.tasks.title': 'Tickets' } },
    valueSets: {},
  }

  it('saveDraft persists a blueprint without touching the tenant layer', () => {
    const draft = saveDraft({ moduleKey: 'kontakte', name: 'Blueprint A', payload })
    expect(draft.status).toBe('draft')
    // Live tenant layer unchanged.
    expect(resolveLabelOverrides('de')['work.tasks.title'].provenance).toBe('default')
  })

  it('commitDraftNow promotes the payload into the tenant layer and goes live', () => {
    const draft = commitDraftNow({ moduleKey: 'kontakte', name: 'Go live', payload, mode: 'now' })
    expect(draft.status).toBe('live')
    const entry = resolveLabelOverrides('de')['work.tasks.title']
    expect(entry).toEqual({ key: 'work.tasks.title', value: 'Tickets', provenance: 'tenant' })
  })

  it('rollbackDeploy restores the pre-deploy tenant snapshot and supersedes', () => {
    const draft = commitDraftNow({ moduleKey: 'kontakte', name: 'Roll me back', payload, mode: 'now' })
    expect(resolveLabelOverrides('de')['work.tasks.title'].value).toBe('Tickets')

    rollbackDeploy(draft.id)
    expect(getDraft(draft.id)?.status).toBe('superseded')
    // Back to the code default — the deploy was undone.
    expect(resolveLabelOverrides('de')['work.tasks.title'].provenance).toBe('default')
  })

  it('applyDraftToTenant reports applied counts and only whitelisted keys', () => {
    const res = applyDraftToTenant({
      labels: { de: { 'work.tasks.title': 'X', 'not.a.whitelisted.key': 'ignored' } },
      valueSets: {},
    })
    expect(res.labelCount).toBe(1)
  })
})

describe('draft custom fields (E-3c snapshot-diff)', () => {
  const ENTITY = 'helpdesk_ticket'
  const fieldsPayload = (fields: CustomFieldDefinition[]): CustomizationDraftPayload => ({
    labels: {},
    valueSets: {},
    customFields: { [ENTITY]: fields },
  })

  it('deploy creates a staged new field with a real store id', () => {
    const baseline = listCustomFields(ENTITY)
    const draftField: CustomFieldDefinition = {
      id: 'draft_abc123', entity: ENTITY, key: 'draft_x', label: 'Rückrufwunsch',
      type: 'boolean', required: false, options: [], visible: true, order: baseline.length, inUse: false,
    }
    commitDraftNow({ moduleKey: 'helpdesk', name: 'Add field', payload: fieldsPayload([...baseline, draftField]), mode: 'now' })

    const after = listCustomFields(ENTITY)
    expect(after).toHaveLength(baseline.length + 1)
    const created = after.find((f) => f.label === 'Rückrufwunsch')
    expect(created).toBeDefined()
    expect(created?.id.startsWith('draft_')).toBe(false) // got a real cf_ id
  })

  it('deploy updates a changed field and deletes a removed one', () => {
    const baseline = listCustomFields(ENTITY)
    const desired = baseline
      .filter((f) => f.key !== 'escalation_reason') // remove one
      .map((f) => (f.key === 'sla_tier' ? { ...f, label: 'Service-Level' } : f)) // change one
    commitDraftNow({ moduleKey: 'helpdesk', name: 'Edit fields', payload: fieldsPayload(desired), mode: 'now' })

    const after = listCustomFields(ENTITY)
    expect(after.find((f) => f.key === 'escalation_reason')).toBeUndefined()
    expect(after.find((f) => f.key === 'sla_tier')?.label).toBe('Service-Level')
  })

  it('rollback restores the field store to before the deploy', () => {
    const baseline = listCustomFields(ENTITY)
    const draftField: CustomFieldDefinition = {
      id: 'draft_zzz', entity: ENTITY, key: 'draft_y', label: 'Temp-Feld',
      type: 'text', required: false, options: [], visible: true, order: baseline.length, inUse: false,
    }
    const draft = commitDraftNow({ moduleKey: 'helpdesk', name: 'Add + rollback', payload: fieldsPayload([...baseline, draftField]), mode: 'now' })
    expect(listCustomFields(ENTITY)).toHaveLength(baseline.length + 1)

    rollbackDeploy(draft.id)
    expect(listCustomFields(ENTITY)).toHaveLength(baseline.length)
    expect(listCustomFields(ENTITY).find((f) => f.label === 'Temp-Feld')).toBeUndefined()
  })
})

describe('scheduled deploy (mock scheduler)', () => {
  const payload: CustomizationDraftPayload = {
    labels: { de: { 'work.projects.title': 'Mandate-Neu' } },
    valueSets: {},
  }

  it('promotes only drafts whose scheduledAt has passed', () => {
    const past = scheduleDraft({
      moduleKey: 'kontakte',
      name: 'Past rollout',
      payload,
      mode: 'scheduled',
      scheduledAt: '2000-01-01T06:00:00.000Z',
    })
    const future = scheduleDraft({
      moduleKey: 'kontakte',
      name: 'Future rollout',
      payload,
      mode: 'scheduled',
      scheduledAt: '2099-01-01T06:00:00.000Z',
    })
    expect(past.status).toBe('scheduled')
    expect(future.status).toBe('scheduled')

    const promoted = runDueScheduledDeploys('2020-06-01T00:00:00.000Z')
    expect(promoted.map((d) => d.id)).toContain(past.id)
    expect(promoted.map((d) => d.id)).not.toContain(future.id)
    expect(getDraft(past.id)?.status).toBe('live')
    expect(getDraft(future.id)?.status).toBe('scheduled')
  })
})
