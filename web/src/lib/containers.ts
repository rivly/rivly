import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { api } from './api'

export type ContainerPort = {
  privatePort: number
  publicPort: number
  type: string
  ip?: string
}

export type Container = {
  id: string
  name: string
  image: string
  state: string
  status: string
  stack: string
  created: number
  ip: string
  ports: ContainerPort[]
}

export function containersQueryOptions(envId: number) {
  return queryOptions({
    queryKey: ['containers', envId],
    queryFn: () => api.get<Container[]>(`/environments/${envId}/containers`),
  })
}

export function useContainers(envId: number) {
  return useQuery(containersQueryOptions(envId))
}

export type ContainerAction =
  | 'start'
  | 'stop'
  | 'restart'
  | 'pause'
  | 'unpause'
  | 'kill'
  | 'remove'

export type ActionResult = { id: string; ok: boolean; error?: string }

export function useContainerActions(envId: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { action: ContainerAction; ids: string[] }) =>
      api.post<{ results: ActionResult[] }>(
        `/environments/${envId}/containers/actions`,
        input,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['containers', envId] })
    },
  })
}
