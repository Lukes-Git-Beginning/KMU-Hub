/**
 * TanStack Query hooks for the Produktion module.
 *
 * Queries for orders, bookings, plans, and capacity overview.
 * Mutations for order lifecycle, booking management, and plan management.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listOrders,
  getOrder,
  createOrder,
  updateOrder,
  deleteOrder,
  startOrder,
  completeOrder,
  cancelOrder,
  listMachineBookings,
  createMachineBooking,
  updateMachineBooking,
  deleteMachineBooking,
  createPlan,
  getPlan,
  updatePlan,
  getCapacityOverview,
  listBOMs,
  getBOM,
  createBOM,
  updateBOM,
  deleteBOM,
  listWorkSteps,
  createWorkStep,
  updateWorkStep,
  deleteWorkStep,
  listMachines,
  createMachine,
  updateMachine,
  deleteMachine,
  listQualityChecks,
  getQualityCheck,
  createQualityCheck,
} from '../produktion-client'
import type {
  CreateOrderInput,
  UpdateOrderInput,
  ListOrdersParams,
  CreateBookingInput,
  UpdateBookingInput,
  ListBookingsParams,
  CreatePlanInput,
  UpdatePlanInput,
  CreateBOMInput,
  UpdateBOMInput,
  ListBOMsParams,
  CreateWorkStepInput,
  UpdateWorkStepInput,
  CreateMachineInput,
  UpdateMachineInput,
  ListMachinesParams,
  CreateQualityCheckInput,
  ListQualityChecksParams,
} from '../produktion-types'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const produktionKeys = {
  all: ['produktion'] as const,
  orders: (params?: ListOrdersParams) => ['produktion', 'orders', params] as const,
  order: (id: string) => ['produktion', 'orders', id] as const,
  bookings: (params?: ListBookingsParams) => ['produktion', 'bookings', params] as const,
  plan: (id: string) => ['produktion', 'plans', id] as const,
  capacity: (planId: string, machineId: string) =>
    ['produktion', 'plans', planId, 'capacity', machineId] as const,
}

// ---------------------------------------------------------------------------
// Queries — Orders
// ---------------------------------------------------------------------------

export function useOrders(params?: ListOrdersParams) {
  return useQuery({
    queryKey: produktionKeys.orders(params),
    queryFn: () => listOrders(params),
  })
}

export function useOrder(id: string) {
  return useQuery({
    queryKey: produktionKeys.order(id),
    queryFn: () => getOrder(id),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Queries — Bookings
// ---------------------------------------------------------------------------

export function useMachineBookings(params?: ListBookingsParams) {
  return useQuery({
    queryKey: produktionKeys.bookings(params),
    queryFn: () => listMachineBookings(params),
  })
}

// ---------------------------------------------------------------------------
// Queries — Plans
// ---------------------------------------------------------------------------

export function usePlan(id: string) {
  return useQuery({
    queryKey: produktionKeys.plan(id),
    queryFn: () => getPlan(id),
    enabled: !!id,
  })
}

export function useCapacityOverview(planId: string, machineId: string) {
  return useQuery({
    queryKey: produktionKeys.capacity(planId, machineId),
    queryFn: () => getCapacityOverview(planId, machineId),
    enabled: !!planId && !!machineId,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Orders
// ---------------------------------------------------------------------------

export function useCreateOrder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateOrderInput) => createOrder(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'orders'] }),
  })
}

export function useUpdateOrder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateOrderInput & { id: string }) =>
      updateOrder(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: produktionKeys.order(variables.id) })
      qc.invalidateQueries({ queryKey: ['produktion', 'orders'] })
    },
  })
}

export function useDeleteOrder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteOrder(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'orders'] }),
  })
}

export function useStartOrder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => startOrder(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: produktionKeys.order(id) })
      qc.invalidateQueries({ queryKey: ['produktion', 'orders'] })
    },
  })
}

export function useCompleteOrder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => completeOrder(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: produktionKeys.order(id) })
      qc.invalidateQueries({ queryKey: ['produktion', 'orders'] })
    },
  })
}

export function useCancelOrder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => cancelOrder(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: produktionKeys.order(id) })
      qc.invalidateQueries({ queryKey: ['produktion', 'orders'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Bookings
// ---------------------------------------------------------------------------

export function useCreateMachineBooking() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateBookingInput) => createMachineBooking(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'bookings'] }),
  })
}

export function useUpdateMachineBooking() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateBookingInput & { id: string }) =>
      updateMachineBooking(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'bookings'] }),
  })
}

export function useDeleteMachineBooking() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteMachineBooking(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'bookings'] }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Plans
// ---------------------------------------------------------------------------

export function useCreatePlan() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreatePlanInput) => createPlan(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'plans'] }),
  })
}

export function useUpdatePlan() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdatePlanInput & { id: string }) =>
      updatePlan(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: produktionKeys.plan(variables.id) })
    },
  })
}

// ---------------------------------------------------------------------------
// Queries — BOMs
// ---------------------------------------------------------------------------

export function useBOMs(params?: ListBOMsParams) {
  return useQuery({
    queryKey: ['produktion', 'boms', params],
    queryFn: () => listBOMs(params),
  })
}

export function useBOM(id: string) {
  return useQuery({
    queryKey: ['produktion', 'boms', id],
    queryFn: () => getBOM(id),
    enabled: !!id,
  })
}

export function useCreateBOM() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateBOMInput) => createBOM(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'boms'] }),
  })
}

export function useUpdateBOM() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateBOMInput & { id: string }) => updateBOM(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'boms'] }),
  })
}

export function useDeleteBOM() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteBOM(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'boms'] }),
  })
}

// ---------------------------------------------------------------------------
// Queries — Work Steps
// ---------------------------------------------------------------------------

export function useWorkSteps(orderId: string) {
  return useQuery({
    queryKey: ['produktion', 'worksteps', orderId],
    queryFn: () => listWorkSteps(orderId),
    enabled: !!orderId,
  })
}

export function useCreateWorkStep() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ orderId, ...input }: Omit<CreateWorkStepInput, 'order_id'> & { orderId: string }) =>
      createWorkStep(orderId, input),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: ['produktion', 'worksteps', variables.orderId] }),
  })
}

export function useUpdateWorkStep() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ orderId, stepId, ...input }: UpdateWorkStepInput & { orderId: string; stepId: string }) =>
      updateWorkStep(orderId, stepId, input),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: ['produktion', 'worksteps', variables.orderId] }),
  })
}

export function useDeleteWorkStep() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ orderId, stepId }: { orderId: string; stepId: string }) =>
      deleteWorkStep(orderId, stepId),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: ['produktion', 'worksteps', variables.orderId] }),
  })
}

// ---------------------------------------------------------------------------
// Queries — Machines
// ---------------------------------------------------------------------------

export function useMachines(params?: ListMachinesParams) {
  return useQuery({
    queryKey: ['produktion', 'machines', params],
    queryFn: () => listMachines(params),
  })
}

export function useCreateMachine() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateMachineInput) => createMachine(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'machines'] }),
  })
}

export function useUpdateMachine() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateMachineInput & { id: string }) => updateMachine(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'machines'] }),
  })
}

export function useDeleteMachine() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteMachine(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'machines'] }),
  })
}

// ---------------------------------------------------------------------------
// Queries — Quality Checks
// ---------------------------------------------------------------------------

export function useQualityChecks(params?: ListQualityChecksParams) {
  return useQuery({
    queryKey: ['produktion', 'quality', params],
    queryFn: () => listQualityChecks(params),
  })
}

export function useGetQualityCheck(id: string) {
  return useQuery({
    queryKey: ['produktion', 'quality', id],
    queryFn: () => getQualityCheck(id),
    enabled: !!id,
  })
}

export function useCreateQualityCheck() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateQualityCheckInput) => createQualityCheck(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['produktion', 'quality'] }),
  })
}
