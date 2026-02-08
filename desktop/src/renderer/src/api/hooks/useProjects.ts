/**
 * TanStack Query hooks for Project CRUD operations.
 *
 * Includes project list/detail, members, statuses, preferences,
 * archiving, and template operations.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'

// ---------------------------------------------------------------------------
// Query params
// ---------------------------------------------------------------------------

export interface ProjectListParams {
  page?: number
  page_size?: number
  search?: string
  include_archived?: boolean
  templates_only?: boolean
}

// ---------------------------------------------------------------------------
// Project queries
// ---------------------------------------------------------------------------

export function useProjects(params?: ProjectListParams) {
  return useQuery({
    queryKey: ['projects', params],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/projects', {
        params: { query: params },
      })
      if (error) throw error
      return data
    },
  })
}

export function useProject(id: string) {
  return useQuery({
    queryKey: ['projects', id],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/projects/{id}', {
        params: { path: { id } },
      })
      if (error) throw error
      return data
    },
    enabled: !!id,
  })
}

export function useProjectMembers(projectId: string) {
  return useQuery({
    queryKey: ['projects', projectId, 'members'],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/projects/{id}/members',
        { params: { path: { id: projectId } } }
      )
      if (error) throw error
      return data
    },
    enabled: !!projectId,
  })
}

export function useProjectStatuses(projectId: string) {
  return useQuery({
    queryKey: ['projects', projectId, 'statuses'],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/projects/{id}/statuses',
        { params: { path: { id: projectId } } }
      )
      if (error) throw error
      return data
    },
    enabled: !!projectId,
  })
}

export function useProjectPreference(projectId: string) {
  return useQuery({
    queryKey: ['projects', projectId, 'preferences'],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/projects/{id}/preferences',
        { params: { path: { id: projectId } } }
      )
      if (error) throw error
      return data
    },
    enabled: !!projectId,
  })
}

// ---------------------------------------------------------------------------
// Project mutations
// ---------------------------------------------------------------------------

export function useCreateProject() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      name: string
      project_key: string
      description?: string
      statuses?: Array<{
        name: string
        color: string
        is_default: boolean
        is_closed: boolean
      }>
    }) => {
      // Create the project first
      const { data, error } = await apiClient.POST('/api/v1/projects', {
        body: {
          name: body.name,
          project_key: body.project_key,
          description: body.description,
        },
      })
      if (error) throw error

      // If statuses were provided and project created, create them
      if (body.statuses && data?.project?.id) {
        for (const status of body.statuses) {
          await apiClient.POST('/api/v1/projects/{id}/statuses', {
            params: { path: { id: data.project.id } },
            body: {
              name: status.name,
              color: status.color,
              is_default: status.is_default,
              is_closed: status.is_closed,
            },
          })
        }
      }

      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

export function useUpdateProject() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      id,
      ...body
    }: {
      id: string
      name?: string
      description?: string
    }) => {
      const { data, error } = await apiClient.PUT('/api/v1/projects/{id}', {
        params: { path: { id } },
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      queryClient.invalidateQueries({ queryKey: ['projects', variables.id] })
    },
  })
}

export function useArchiveProject() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/projects/{id}/archive',
        { params: { path: { id } } }
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Member mutations
// ---------------------------------------------------------------------------

export function useAddMember() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      projectId,
      user_id,
      role,
    }: {
      projectId: string
      user_id: string
      role: 'owner' | 'member' | 'viewer'
    }) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/projects/{id}/members',
        {
          params: { path: { id: projectId } },
          body: { user_id, role },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['projects', variables.projectId, 'members'],
      })
    },
  })
}

export function useRemoveMember() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      projectId,
      userId,
    }: {
      projectId: string
      userId: string
    }) => {
      const { data, error } = await apiClient.DELETE(
        '/api/v1/projects/{id}/members/{userId}',
        { params: { path: { id: projectId, userId } } }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['projects', variables.projectId, 'members'],
      })
    },
  })
}

// ---------------------------------------------------------------------------
// Status mutations
// ---------------------------------------------------------------------------

export function useCreateStatus() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      projectId,
      ...body
    }: {
      projectId: string
      name: string
      color?: string
      is_default?: boolean
      is_closed?: boolean
    }) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/projects/{id}/statuses',
        {
          params: { path: { id: projectId } },
          body,
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

export function useUpdateStatus() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      id,
      ...body
    }: {
      id: string
      name?: string
      color?: string
      is_default?: boolean
      is_closed?: boolean
    }) => {
      const { data, error } = await apiClient.PUT(
        '/api/v1/project-statuses/{id}',
        {
          params: { path: { id } },
          body,
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

export function useDeleteStatus() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await apiClient.DELETE(
        '/api/v1/project-statuses/{id}',
        { params: { path: { id } } }
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

export function useReorderStatuses() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      projectId,
      status_ids,
    }: {
      projectId: string
      status_ids: string[]
    }) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/projects/{id}/statuses/reorder',
        {
          params: { path: { id: projectId } },
          body: { status_ids },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Preference mutations
// ---------------------------------------------------------------------------

export function useSetPreference() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      projectId,
      ...body
    }: {
      projectId: string
      view_type?: 'list' | 'kanban' | 'gantt'
      list_group_by?: string
      list_sort_by?: string
      list_sort_desc?: boolean
    }) => {
      const { data, error } = await apiClient.PUT(
        '/api/v1/projects/{id}/preferences',
        {
          params: { path: { id: projectId } },
          body,
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['projects', variables.projectId, 'preferences'],
      })
    },
  })
}

// ---------------------------------------------------------------------------
// Template mutations
// ---------------------------------------------------------------------------

export function useSaveAsTemplate() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      projectId,
      template_name,
    }: {
      projectId: string
      template_name: string
    }) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/projects/{id}/template',
        {
          params: { path: { id: projectId } },
          body: { template_name },
        }
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

export function useCreateFromTemplate() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      template_id: string
      name: string
      project_key: string
    }) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/projects/from-template',
        { body }
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}
