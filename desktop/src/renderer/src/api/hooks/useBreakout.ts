/**
 * TanStack Query hooks for breakout room management (Wave 6A).
 *
 * Design:
 *   - useBreakoutAssignment: polls every 8 s for the caller's room assignment.
 *     Authoritative for room-switch decisions (DataChannel is an accelerator only).
 *   - useBreakoutRooms: host-only overview, also polls at 8 s.
 *   - Mutations invalidate the relevant query keys on success.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getBreakoutAssignment,
  listBreakoutRooms,
  createBreakoutRooms,
  assignBreakoutParticipant,
  closeBreakoutRooms,
  returnToMainRoom,
} from '../video-client'
import type { CreateBreakoutRoomsRequest } from '../video-types'

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

/**
 * Poll for the current user's breakout room assignment.
 * Returns `room: BreakoutRoom | null` — null means the caller is in the main room.
 * Polls every 8 s while meetingId is set; the DataChannel subscriber in
 * VideoCallView can invalidate the cache early for the main→breakout fast path.
 */
export function useBreakoutAssignment(meetingId: string | undefined) {
  return useQuery({
    queryKey: ['breakout', 'assignment', meetingId],
    queryFn: () => getBreakoutAssignment(meetingId!),
    enabled: !!meetingId,
    refetchInterval: 8_000,
    select: (data) => data.room,
  })
}

/**
 * Host-only: poll all breakout rooms for the meeting with participant lists.
 * Polls every 8 s to reflect assignment changes made by the host.
 */
export function useBreakoutRooms(meetingId: string | undefined) {
  return useQuery({
    queryKey: ['breakout', 'rooms', meetingId],
    queryFn: () => listBreakoutRooms(meetingId!),
    enabled: !!meetingId,
    refetchInterval: 8_000,
    select: (data) => data.rooms,
  })
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

/** Create N breakout rooms for a meeting (host only). */
export function useCreateBreakoutRooms(meetingId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: CreateBreakoutRoomsRequest) => createBreakoutRooms(meetingId, req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['breakout', 'rooms', meetingId] })
    },
  })
}

/** Assign a participant to a room or back to the main room (host only). */
export function useAssignBreakoutParticipant(meetingId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      targetUserId,
      breakoutRoomId,
    }: {
      targetUserId: string
      breakoutRoomId: string | null
    }) => assignBreakoutParticipant(meetingId, targetUserId, breakoutRoomId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['breakout', 'rooms', meetingId] })
    },
  })
}

/** Close all breakout rooms (host only). */
export function useCloseBreakoutRooms(meetingId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => closeBreakoutRooms(meetingId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['breakout', 'rooms', meetingId] })
      queryClient.invalidateQueries({ queryKey: ['breakout', 'assignment', meetingId] })
    },
  })
}

/**
 * Return a participant to the main room.
 * Calling without arguments returns the current user themselves.
 * Passing a targetUserId returns another participant (host only).
 */
export function useReturnToMainRoom(meetingId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (targetUserId?: string) => returnToMainRoom(meetingId, targetUserId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['breakout', 'assignment', meetingId] })
    },
  })
}
