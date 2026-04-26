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
