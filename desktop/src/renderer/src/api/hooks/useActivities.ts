/**
 * TanStack Query hooks for Activity CRUD operations.
 *
 * Activities represent interactions (calls, meetings, notes, emails, tasks)
 * linked to contacts, companies, or deals.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'

export interface ActivityListParams {
  page?: number
  page_size?: number
  activity_type?: 'call' | 'meeting' | 'note' | 'email' | 'task'
  contact_id?: string
  company_id?: string
  deal_id?: string
  assigned_to?: string
  is_completed?: boolean
  sort_by?: 'created_at' | 'due_date' | 'subject'
  sort_desc?: boolean
}

export function useActivities(params?: ActivityListParams) {
  return useQuery({
    queryKey: ['activities', params],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/activities', {
        params: { query: params },
      })
      if (error) throw error
      return data
    },
  })
}

export function useActivity(id: string) {
  return useQuery({
    queryKey: ['activities', id],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/activities/{id}', {
        params: { path: { id } },
      })
      if (error) throw error
      return data
    },
    enabled: !!id,
  })
}

export function useCreateActivity() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      activity_type: 'call' | 'meeting' | 'note' | 'email' | 'task'
      subject: string
      description?: string
      contact_id?: string
      company_id?: string
      deal_id?: string
      assigned_to?: string
      due_date?: string
    }) => {
      const { data, error } = await apiClient.POST('/api/v1/activities', {
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['activities'] })
    },
  })
}

export function useUpdateActivity() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      id,
      ...body
    }: {
      id: string
      subject?: string
      description?: string
      contact_id?: string
      company_id?: string
      deal_id?: string
      assigned_to?: string
      due_date?: string
    }) => {
      const { data, error } = await apiClient.PUT('/api/v1/activities/{id}', {
        params: { path: { id } },
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['activities'] })
      queryClient.invalidateQueries({ queryKey: ['activities', variables.id] })
    },
  })
}

export function useDeleteActivity() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await apiClient.DELETE(
        '/api/v1/activities/{id}',
        {
          params: { path: { id } },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['activities'] })
    },
  })
}

export function useCompleteActivity() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/activities/{id}/complete',
        {
          params: { path: { id } },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, id) => {
      queryClient.invalidateQueries({ queryKey: ['activities'] })
      queryClient.invalidateQueries({ queryKey: ['activities', id] })
    },
  })
}
