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

export type ContainerRef = { id: string; name: string }

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

export type NetworkAttachment = { name: string; ip: string }
export type ContainerMount = {
  type: string
  source: string
  destination: string
  name: string
  rw: boolean
}

export type ContainerDetail = {
  id: string
  name: string
  image: string
  state: string
  created: number
  startedAt: string
  command: string
  restartPolicy: string
  ports: ContainerPort[]
  networks: NetworkAttachment[]
  mounts: ContainerMount[]
  env: string[]
  labels: Record<string, string>
}

export function containerDetailQueryOptions(envId: number, containerId: string) {
  return queryOptions({
    queryKey: ['container', envId, containerId],
    queryFn: () =>
      api.get<ContainerDetail>(`/environments/${envId}/containers/${containerId}`),
  })
}

export function useContainerDetail(envId: number, containerId: string) {
  return useQuery(containerDetailQueryOptions(envId, containerId))
}

export type PortMapping = { hostPort: string; containerPort: string; proto: string }
export type MountMapping = { source: string; target: string; readOnly: boolean }
export type EnvVar = { key: string; value: string }

export type RunContainerInput = {
  name?: string
  image: string
  command?: string
  env?: EnvVar[]
  ports?: PortMapping[]
  mounts?: MountMapping[]
  network?: string
  restartPolicy: string
  start: boolean
}

export function useCreateContainer(envId: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: RunContainerInput) =>
      api.post<{ id: string }>(`/environments/${envId}/containers`, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['containers', envId] })
    },
  })
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
