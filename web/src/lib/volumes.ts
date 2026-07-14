import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { api } from './api'
import type { ActionResult } from './containers'

export type Volume = {
  name: string
  driver: string
  mountpoint: string
  stack: string
  created: number
  inUse: boolean
}

export function volumesQueryOptions(envId: number) {
  return queryOptions({
    queryKey: ['volumes', envId],
    queryFn: () => api.get<Volume[]>(`/environments/${envId}/volumes`),
  })
}

export function useVolumes(envId: number) {
  return useQuery(volumesQueryOptions(envId))
}

export type VolumeAction = 'remove'

export function useVolumeActions(envId: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { action: VolumeAction; ids: string[] }) =>
      api.post<{ results: ActionResult[] }>(
        `/environments/${envId}/volumes/actions`,
        input,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['volumes', envId] })
    },
  })
}
