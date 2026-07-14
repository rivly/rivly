import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { api } from './api'
import type { ActionResult } from './containers'

export type Image = {
  id: string
  tags: string[]
  size: number
  created: number
  inUse: boolean
}

export function imagesQueryOptions(envId: number) {
  return queryOptions({
    queryKey: ['images', envId],
    queryFn: () => api.get<Image[]>(`/environments/${envId}/images`),
  })
}

export function useImages(envId: number) {
  return useQuery(imagesQueryOptions(envId))
}

export function useImagePrune(envId: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (all: boolean) =>
      api.post<{ imagesDeleted: number; spaceReclaimed: number }>(
        `/environments/${envId}/images/prune`,
        { all },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['images', envId] })
    },
  })
}

export type ImageAction = 'remove'

export function useImageActions(envId: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { action: ImageAction; ids: string[] }) =>
      api.post<{ results: ActionResult[] }>(
        `/environments/${envId}/images/actions`,
        input,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['images', envId] })
    },
  })
}
