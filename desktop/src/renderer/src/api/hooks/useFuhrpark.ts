/**
 * TanStack Query hooks for the Fuhrpark (fleet management) module.
 *
 * Queries for vehicles, services, damages, history, and TÜV checks.
 * Mutations for full vehicle lifecycle, service scheduling, and damage reporting.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listVehicles,
  getVehicle,
  createVehicle,
  updateVehicle,
  deleteVehicle,
  getVehicleHistory,
  listVehicleServices,
  scheduleService,
  listServices,
  updateService,
  deleteService,
  completeService,
  listUpcomingServices,
  listVehicleDamages,
  reportDamage,
  listDamages,
  updateDamage,
  resolveDamage,
  checkTuevDue,
} from '../fuhrpark-client'
import type {
  CreateVehicleInput,
  UpdateVehicleInput,
  ListVehiclesParams,
  ScheduleServiceInput,
  UpdateServiceInput,
  CompleteServiceInput,
  ListServicesParams,
  ReportDamageInput,
  UpdateDamageInput,
  ResolveDamageInput,
  ListDamagesParams,
  ListVehicleHistoryParams,
  CheckTuevDueParams,
  ListUpcomingServicesParams,
} from '../fuhrpark-types'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const fuhrparkKeys = {
  all: ['fuhrpark'] as const,
  vehicles: (params?: ListVehiclesParams) => ['fuhrpark', 'vehicles', params] as const,
  vehicle: (id: string) => ['fuhrpark', 'vehicle', id] as const,
  vehicleHistory: (id: string, params?: ListVehicleHistoryParams) =>
    ['fuhrpark', 'vehicle', id, 'history', params] as const,
  vehicleServices: (vehicleId: string, params?: ListServicesParams) =>
    ['fuhrpark', 'vehicle', vehicleId, 'services', params] as const,
  vehicleDamages: (vehicleId: string, params?: ListDamagesParams) =>
    ['fuhrpark', 'vehicle', vehicleId, 'damages', params] as const,
  services: (params?: ListServicesParams) => ['fuhrpark', 'services', params] as const,
  upcomingServices: (params?: ListUpcomingServicesParams) =>
    ['fuhrpark', 'services', 'upcoming', params] as const,
  damages: (params?: ListDamagesParams) => ['fuhrpark', 'damages', params] as const,
  tuevDue: (params?: CheckTuevDueParams) => ['fuhrpark', 'tuev-due', params] as const,
}

// ---------------------------------------------------------------------------
// Queries — Vehicles
// ---------------------------------------------------------------------------

export function useVehiclesList(params?: ListVehiclesParams) {
  return useQuery({
    queryKey: fuhrparkKeys.vehicles(params),
    queryFn: () => listVehicles(params),
    staleTime: 30_000,
  })
}

export function useVehicle(id: string) {
  return useQuery({
    queryKey: fuhrparkKeys.vehicle(id),
    queryFn: () => getVehicle(id),
    enabled: !!id,
    staleTime: 30_000,
  })
}

export function useVehicleHistory(id: string, params?: ListVehicleHistoryParams) {
  return useQuery({
    queryKey: fuhrparkKeys.vehicleHistory(id, params),
    queryFn: () => getVehicleHistory(id, params),
    enabled: !!id,
    staleTime: 30_000,
  })
}

// ---------------------------------------------------------------------------
// Queries — Services
// ---------------------------------------------------------------------------

export function useVehicleServices(vehicleId: string, params?: ListServicesParams) {
  return useQuery({
    queryKey: fuhrparkKeys.vehicleServices(vehicleId, params),
    queryFn: () => listVehicleServices(vehicleId, params),
    enabled: !!vehicleId,
    staleTime: 30_000,
  })
}

export function useServicesList(params?: ListServicesParams) {
  return useQuery({
    queryKey: fuhrparkKeys.services(params),
    queryFn: () => listServices(params),
    staleTime: 30_000,
  })
}

export function useUpcomingServices(params?: ListUpcomingServicesParams) {
  return useQuery({
    queryKey: fuhrparkKeys.upcomingServices(params),
    queryFn: () => listUpcomingServices(params),
    staleTime: 60_000,
  })
}

// ---------------------------------------------------------------------------
// Queries — Damages
// ---------------------------------------------------------------------------

export function useVehicleDamages(vehicleId: string, params?: ListDamagesParams) {
  return useQuery({
    queryKey: fuhrparkKeys.vehicleDamages(vehicleId, params),
    queryFn: () => listVehicleDamages(vehicleId, params),
    enabled: !!vehicleId,
    staleTime: 30_000,
  })
}

export function useDamagesList(params?: ListDamagesParams) {
  return useQuery({
    queryKey: fuhrparkKeys.damages(params),
    queryFn: () => listDamages(params),
    staleTime: 30_000,
  })
}

// ---------------------------------------------------------------------------
// Queries — TÜV check
// ---------------------------------------------------------------------------

export function useTuevDueCheck(params?: CheckTuevDueParams) {
  return useQuery({
    queryKey: fuhrparkKeys.tuevDue(params),
    queryFn: () => checkTuevDue(params),
    staleTime: 5 * 60_000,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Vehicles
// ---------------------------------------------------------------------------

export function useCreateVehicle() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateVehicleInput) => createVehicle(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['fuhrpark', 'vehicles'] }),
  })
}

export function useUpdateVehicle() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateVehicleInput & { id: string }) =>
      updateVehicle(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: fuhrparkKeys.vehicle(variables.id) })
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'vehicles'] })
    },
  })
}

export function useDeleteVehicle() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteVehicle(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['fuhrpark', 'vehicles'] }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Services
// ---------------------------------------------------------------------------

export function useScheduleService() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ vehicleId, ...body }: ScheduleServiceInput & { vehicleId: string }) =>
      scheduleService(vehicleId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: fuhrparkKeys.vehicleServices(variables.vehicleId),
      })
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'services'] })
      qc.invalidateQueries({ queryKey: fuhrparkKeys.upcomingServices() })
    },
  })
}

export function useUpdateService() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: UpdateServiceInput & { id: string }) =>
      updateService(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['fuhrpark', 'services'] }),
  })
}

export function useDeleteService() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteService(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'services'] })
      qc.invalidateQueries({ queryKey: fuhrparkKeys.upcomingServices() })
    },
  })
}

export function useCompleteService() {
  const qc = useQueryClient()
  return useMutation({
    // Bug #21: vehicleId is required so we can invalidate the per-vehicle services cache
    mutationFn: ({ id, vehicleId: _vehicleId, ...body }: CompleteServiceInput & { id: string; vehicleId: string }) =>
      completeService(id, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: fuhrparkKeys.vehicleServices(variables.vehicleId) })
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'services'] })
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'vehicles'] })
      qc.invalidateQueries({ queryKey: fuhrparkKeys.upcomingServices() })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Damages
// ---------------------------------------------------------------------------

export function useReportDamage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ vehicleId, ...body }: ReportDamageInput & { vehicleId: string }) =>
      reportDamage(vehicleId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: fuhrparkKeys.vehicleDamages(variables.vehicleId),
      })
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'damages'] })
    },
  })
}

export function useUpdateDamage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: UpdateDamageInput & { id: string }) =>
      updateDamage(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['fuhrpark', 'damages'] }),
  })
}

export function useResolveDamage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: ResolveDamageInput & { id: string }) =>
      resolveDamage(id, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'damages'] })
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'vehicles'] })
    },
  })
}
