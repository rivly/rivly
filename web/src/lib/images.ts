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

export type ImageContainer = { id: string; name: string }

export type ImageDetail = {
  id: string
  tags: string[]
  digests: string[]
  size: number
  created: number
  architecture: string
  os: string
  author: string
  workingDir: string
  command: string[] | null
  entrypoint: string[] | null
  env: string[] | null
  exposedPorts: string[] | null
  labels: Record<string, string> | null
  containers: ImageContainer[]
}

export function useImageDetail(envId: number, imageId: string) {
  return useQuery({
    queryKey: ['image', envId, imageId],
    queryFn: () => api.get<ImageDetail>(`/environments/${envId}/images/${imageId}`),
  })
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
