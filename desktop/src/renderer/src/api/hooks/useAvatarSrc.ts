/**
 * Resolves a stored object key (e.g. users.avatar_url, item attachment keys) to
 * a short-lived viewable URL via the presigned-download endpoint.
 *
 * The presigned URL is valid ~1h; we cache per object key and refetch well
 * before expiry so the same key isn't re-signed on every render.
 */
import { useQuery } from '@tanstack/react-query'
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'

interface PresignDownloadResponse {
  download_url: string
  object_key: string
}

export function useAvatarSrc(objectKey: string | null | undefined): string | undefined {
  const key = objectKey?.trim() || null

  const { data } = useQuery({
    queryKey: ['files', 'presign-download', key],
    enabled: !!key,
    staleTime: 50 * 60 * 1000, // 50 min, below the 1h presign-download expiry
    gcTime: 60 * 60 * 1000,
    queryFn: () =>
      authenticatedRequest<PresignDownloadResponse>({
        method: 'GET',
        path: '/api/v1/files/presign-download',
        params: { object_key: key as string },
      }),
  })

  return data?.download_url
}
