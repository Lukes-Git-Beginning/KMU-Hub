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
  listFuelLogs,
  listVehicleFuelLogs,
  createFuelLog,
  updateFuelLog,
  deleteFuelLog,
  listTripLogs,
  listVehicleTripLogs,
  createTripLog,
  updateTripLog,
  deleteTripLog,
  listVehicleDocuments,
  createVehicleDocument,
  deleteVehicleDocument,
  ingestGpsPositions,
  getVehicleRoutes,
  getGpsPositions,
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
  CreateFuelLogRequest,
  UpdateFuelLogRequest,
  CreateTripLogRequest,
  UpdateTripLogRequest,
  CreateVehicleDocumentRequest,
  IngestGpsPositionsRequest,
} from '../fuhrpark-client'

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

// ---------------------------------------------------------------------------
// Queries — Fuel Logs
// ---------------------------------------------------------------------------

export function useFuelLogs(params: { vehicle_id?: string; page?: number; page_size?: number } = {}) {
  return useQuery({
    queryKey: ['fuhrpark', 'fuel-logs', params] as const,
    queryFn: () => listFuelLogs(params),
    staleTime: 30_000,
  })
}

export function useVehicleFuelLogs(
  vehicleId: string,
  params: { page?: number; page_size?: number } = {},
) {
  return useQuery({
    queryKey: ['fuhrpark', 'fuel-logs', 'vehicle', vehicleId, params] as const,
    queryFn: () => listVehicleFuelLogs(vehicleId, params),
    staleTime: 30_000,
    enabled: !!vehicleId,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Fuel Logs
// ---------------------------------------------------------------------------

export function useCreateFuelLog() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ vehicleId, data }: { vehicleId: string; data: CreateFuelLogRequest }) =>
      createFuelLog(vehicleId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'fuel-logs'] })
    },
  })
}

export function useUpdateFuelLog() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateFuelLogRequest }) =>
      updateFuelLog(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'fuel-logs'] })
    },
  })
}

export function useDeleteFuelLog() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteFuelLog(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'fuel-logs'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Queries — Trip Logs
// ---------------------------------------------------------------------------

export function useTripLogs(params: { vehicle_id?: string; page?: number; page_size?: number } = {}) {
  return useQuery({
    queryKey: ['fuhrpark', 'trip-logs', params] as const,
    queryFn: () => listTripLogs(params),
    staleTime: 30_000,
  })
}

export function useVehicleTripLogs(
  vehicleId: string,
  params: { page?: number; page_size?: number } = {},
) {
  return useQuery({
    queryKey: ['fuhrpark', 'trip-logs', 'vehicle', vehicleId, params] as const,
    queryFn: () => listVehicleTripLogs(vehicleId, params),
    staleTime: 30_000,
    enabled: !!vehicleId,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Trip Logs
// ---------------------------------------------------------------------------

export function useCreateTripLog() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ vehicleId, data }: { vehicleId: string; data: CreateTripLogRequest }) =>
      createTripLog(vehicleId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'trip-logs'] })
    },
  })
}

export function useUpdateTripLog() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateTripLogRequest }) =>
      updateTripLog(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'trip-logs'] })
    },
  })
}

export function useDeleteTripLog() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteTripLog(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'trip-logs'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Queries — Vehicle Documents
// ---------------------------------------------------------------------------

export function useVehicleDocuments(
  vehicleId: string,
  params: { page?: number; page_size?: number } = {},
) {
  return useQuery({
    queryKey: ['fuhrpark', 'documents', vehicleId, params] as const,
    queryFn: () => listVehicleDocuments(vehicleId, params),
    staleTime: 60_000,
    enabled: !!vehicleId,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Vehicle Documents
// ---------------------------------------------------------------------------

export function useCreateVehicleDocument() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      vehicleId,
      data,
    }: {
      vehicleId: string
      data: CreateVehicleDocumentRequest
    }) => createVehicleDocument(vehicleId, data),
    onSuccess: (_, { vehicleId }) => {
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'documents', vehicleId] })
    },
  })
}

export function useDeleteVehicleDocument() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, vehicleId: _vehicleId }: { id: string; vehicleId: string }) =>
      deleteVehicleDocument(id),
    onSuccess: (_, { vehicleId }) => {
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'documents', vehicleId] })
    },
  })
}

// ---------------------------------------------------------------------------
// Queries — GPS / Tracking
// ---------------------------------------------------------------------------

export function useVehicleRoutes(
  params: { date_from?: string; date_to?: string; vehicle_id?: string } = {},
) {
  return useQuery({
    queryKey: ['fuhrpark', 'gps-routes', params] as const,
    queryFn: () => getVehicleRoutes(params),
    staleTime: 60_000,
  })
}

export function useGpsPositions(params: {
  vehicle_id: string
  from?: string
  to?: string
  limit?: number
}) {
  return useQuery({
    queryKey: ['fuhrpark', 'gps-positions', params] as const,
    queryFn: () => getGpsPositions(params),
    staleTime: 30_000,
    enabled: !!params.vehicle_id,
  })
}

// ---------------------------------------------------------------------------
// Mutations — GPS / Tracking
// ---------------------------------------------------------------------------

export function useIngestGpsPositions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: IngestGpsPositionsRequest) => ingestGpsPositions(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'gps-routes'] })
      qc.invalidateQueries({ queryKey: ['fuhrpark', 'gps-positions'] })
    },
  })
}
