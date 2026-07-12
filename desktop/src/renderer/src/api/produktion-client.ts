/**
 * Lightweight fetch wrapper for Produktion API endpoints.
 *
 * Follows the same pattern as wiki-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry.
 */
import type {
  ProductionOrder,
  MachineBooking,
  ProductionPlan,
  CapacityOverview,
  CreateOrderInput,
  UpdateOrderInput,
  ListOrdersParams,
  CreateBookingInput,
  UpdateBookingInput,
  ListBookingsParams,
  CreatePlanInput,
  UpdatePlanInput,
  ListOrdersResponse,
  ListBookingsResponse,
  BOMResponse,
  CreateBOMInput,
  UpdateBOMInput,
  ListBOMsParams,
  ListBOMsResponse,
  WorkStepResponse,
  CreateWorkStepInput,
  UpdateWorkStepInput,
  ListWorkStepsResponse,
  MachineResponse,
  CreateMachineInput,
  UpdateMachineInput,
  ListMachinesParams,
  ListMachinesResponse,
  QualityCheckResponse,
  CreateQualityCheckInput,
  ListQualityChecksParams,
  ListQualityChecksResponse,
} from './produktion-types'
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

function request<T>(opts: RequestOptions): Promise<T> {
  return authenticatedRequest<T>(opts)
}

// The gateway wraps single-order responses in the proto OrderResponse envelope
// ({ order: {...} }, snake_case via protojson). Tolerate a bare object too, so
// an older/flat mock doesn't hard-fail.
function unwrapOrder(value: { order?: ProductionOrder } | ProductionOrder): ProductionOrder {
  return (value as { order?: ProductionOrder }).order ?? (value as ProductionOrder)
}

// ---------------------------------------------------------------------------
// Base path
// ---------------------------------------------------------------------------

const BASE = '/api/v1/produktion'

// ---------------------------------------------------------------------------
// Production Orders
// ---------------------------------------------------------------------------

export function listOrders(params?: ListOrdersParams) {
  return request<ListOrdersResponse>({
    method: 'GET',
    path: `${BASE}/orders`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export async function getOrder(id: string) {
  const raw = await request<{ order?: ProductionOrder } | ProductionOrder>({
    method: 'GET',
    path: `${BASE}/orders/${id}`,
  })
  return unwrapOrder(raw)
}

export async function createOrder(body: CreateOrderInput) {
  const raw = await request<{ order?: ProductionOrder } | ProductionOrder>({
    method: 'POST',
    path: `${BASE}/orders`,
    body,
  })
  return unwrapOrder(raw)
}

export async function updateOrder(id: string, body: UpdateOrderInput) {
  const raw = await request<{ order?: ProductionOrder } | ProductionOrder>({
    method: 'PATCH',
    path: `${BASE}/orders/${id}`,
    body,
  })
  return unwrapOrder(raw)
}

export function deleteOrder(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/orders/${id}` })
}

export async function startOrder(id: string) {
  const raw = await request<{ order?: ProductionOrder } | ProductionOrder>({
    method: 'POST',
    path: `${BASE}/orders/${id}/start`,
  })
  return unwrapOrder(raw)
}

export async function completeOrder(id: string) {
  const raw = await request<{ order?: ProductionOrder } | ProductionOrder>({
    method: 'POST',
    path: `${BASE}/orders/${id}/complete`,
  })
  return unwrapOrder(raw)
}

export async function cancelOrder(id: string) {
  const raw = await request<{ order?: ProductionOrder } | ProductionOrder>({
    method: 'POST',
    path: `${BASE}/orders/${id}/cancel`,
  })
  return unwrapOrder(raw)
}

// ---------------------------------------------------------------------------
// Machine Bookings
// ---------------------------------------------------------------------------

export function listMachineBookings(params?: ListBookingsParams) {
  return request<ListBookingsResponse>({
    method: 'GET',
    path: `${BASE}/bookings`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function createMachineBooking(body: CreateBookingInput) {
  return request<MachineBooking>({ method: 'POST', path: `${BASE}/bookings`, body })
}

export function updateMachineBooking(id: string, body: UpdateBookingInput) {
  return request<MachineBooking>({ method: 'PATCH', path: `${BASE}/bookings/${id}`, body })
}

export function deleteMachineBooking(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/bookings/${id}` })
}

// ---------------------------------------------------------------------------
// Production Plans
// ---------------------------------------------------------------------------

export function createPlan(body: CreatePlanInput) {
  return request<ProductionPlan>({ method: 'POST', path: `${BASE}/plans`, body })
}

export function getPlan(id: string) {
  return request<ProductionPlan>({ method: 'GET', path: `${BASE}/plans/${id}` })
}

export function updatePlan(id: string, body: UpdatePlanInput) {
  return request<ProductionPlan>({ method: 'PATCH', path: `${BASE}/plans/${id}`, body })
}

export function getCapacityOverview(planId: string, machineId: string) {
  return request<CapacityOverview>({
    method: 'GET',
    path: `${BASE}/plans/${planId}/capacity`,
    params: { machine_id: machineId },
  })
}

// ---------------------------------------------------------------------------
// BOMs (Stücklisten)
// ---------------------------------------------------------------------------

export function listBOMs(params?: ListBOMsParams) {
  return request<ListBOMsResponse>({
    method: 'GET',
    path: `${BASE}/boms`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getBOM(id: string) {
  return request<{ bom: BOMResponse }>({ method: 'GET', path: `${BASE}/boms/${id}` })
}

export function createBOM(body: CreateBOMInput) {
  return request<{ bom: BOMResponse }>({ method: 'POST', path: `${BASE}/boms`, body })
}

export function updateBOM(id: string, body: UpdateBOMInput) {
  return request<{ bom: BOMResponse }>({ method: 'PATCH', path: `${BASE}/boms/${id}`, body })
}

export function deleteBOM(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/boms/${id}` })
}

// ---------------------------------------------------------------------------
// Work Steps (Arbeitsschritte)
// ---------------------------------------------------------------------------

export function listWorkSteps(orderId: string) {
  return request<ListWorkStepsResponse>({
    method: 'GET',
    path: `${BASE}/orders/${orderId}/steps`,
  })
}

export function createWorkStep(orderId: string, body: Omit<CreateWorkStepInput, 'order_id'>) {
  return request<{ step: WorkStepResponse }>({
    method: 'POST',
    path: `${BASE}/orders/${orderId}/steps`,
    body,
  })
}

export function updateWorkStep(orderId: string, stepId: string, body: UpdateWorkStepInput) {
  return request<{ step: WorkStepResponse }>({
    method: 'PATCH',
    path: `${BASE}/orders/${orderId}/steps/${stepId}`,
    body,
  })
}

export function deleteWorkStep(orderId: string, stepId: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/orders/${orderId}/steps/${stepId}` })
}

// ---------------------------------------------------------------------------
// Machines (Maschinen)
// ---------------------------------------------------------------------------

export function listMachines(params?: ListMachinesParams) {
  return request<ListMachinesResponse>({
    method: 'GET',
    path: `${BASE}/machines`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getMachine(id: string) {
  return request<{ machine: MachineResponse }>({ method: 'GET', path: `${BASE}/machines/${id}` })
}

export function createMachine(body: CreateMachineInput) {
  return request<{ machine: MachineResponse }>({ method: 'POST', path: `${BASE}/machines`, body })
}

export function updateMachine(id: string, body: UpdateMachineInput) {
  return request<{ machine: MachineResponse }>({ method: 'PATCH', path: `${BASE}/machines/${id}`, body })
}

export function deleteMachine(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/machines/${id}` })
}

// ---------------------------------------------------------------------------
// Quality Checks (Qualitätsprüfungen)
// ---------------------------------------------------------------------------

export function listQualityChecks(params?: ListQualityChecksParams) {
  return request<ListQualityChecksResponse>({
    method: 'GET',
    path: `${BASE}/quality`,
    params: params as Record<string, string | number | boolean | undefined>,
  })
}

export function getQualityCheck(id: string) {
  return request<{ check: QualityCheckResponse }>({ method: 'GET', path: `${BASE}/quality/${id}` })
}

export function createQualityCheck(body: CreateQualityCheckInput) {
  return request<{ check: QualityCheckResponse }>({ method: 'POST', path: `${BASE}/quality`, body })
}
