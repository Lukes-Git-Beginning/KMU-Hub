/**
 * The gateway answers bookings with { booking }, plans with { plan } and the
 * capacity overview with { overview } (proto envelope, pinned in the backend
 * e2e). The client hands the hooks the flat object, like unwrapOrder does for
 * orders; a bare object still passes through.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import {
  createMachineBooking,
  createPlan,
  getCapacityOverview,
  getPlan,
  updateMachineBooking,
  updatePlan,
} from '../produktion-client'

function respondWith(body: unknown) {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } }),
    ),
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('produktion client envelopes', () => {
  it('unwraps { booking } on create and update', async () => {
    respondWith({ booking: { id: 'b-1' } })
    expect((await createMachineBooking({} as never)).id).toBe('b-1')
    respondWith({ booking: { id: 'b-2' } })
    expect((await updateMachineBooking('b-2', {} as never)).id).toBe('b-2')
  })

  it('unwraps { plan } on create, get and update', async () => {
    respondWith({ plan: { id: 'p-1' } })
    expect((await createPlan({} as never)).id).toBe('p-1')
    respondWith({ plan: { id: 'p-2' } })
    expect((await getPlan('p-2')).id).toBe('p-2')
    respondWith({ plan: { id: 'p-3' } })
    expect((await updatePlan('p-3', {} as never)).id).toBe('p-3')
  })

  it('unwraps { overview } for the capacity view', async () => {
    respondWith({ overview: { machine_id: 'm-1' } })
    expect((await getCapacityOverview('p-1', 'm-1')).machine_id).toBe('m-1')
  })

  it('passes a bare object through', async () => {
    respondWith({ id: 'p-flat' })
    expect((await getPlan('p-flat')).id).toBe('p-flat')
  })
})
