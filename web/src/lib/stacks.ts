import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { api } from './api'
import type { ActionResult } from './containers'

export type StackState = 'running' | 'partial' | 'stopped'

export type Stack = {
  name: string
  type: string
  services: number
  running: number
  total: number
  state: StackState
  workingDir: string
  createdAt: number
  updatedAt: number
  createdBy: string
  updatedBy: string
}

export function stacksQueryOptions(envId: number) {
  return queryOptions({
    queryKey: ['stacks', envId],
    queryFn: () => api.get<Stack[]>(`/environments/${envId}/stacks`),
  })
}

export function useStacks(envId: number) {
  return useQuery(stacksQueryOptions(envId))
}

export type StackAction = 'start' | 'stop' | 'restart' | 'remove'

export function useStackActions(envId: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { action: StackAction; ids: string[] }) =>
      api.post<{ results: ActionResult[] }>(
        `/environments/${envId}/stacks/actions`,
        input,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stacks', envId] })
      queryClient.invalidateQueries({ queryKey: ['containers', envId] })
    },
  })
}

export type EnvVar = { key: string; value: string }

export function fetchStackContent(envId: number, name: string) {
  return api.get<{ name: string; content: string; env: EnvVar[] }>(
    `/environments/${envId}/stacks/${name}`,
  )
}

export function useDeployStack(envId: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { name: string; content: string; env: EnvVar[] }) =>
      api.post<{ name: string }>(`/environments/${envId}/stacks`, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stacks', envId] })
      queryClient.invalidateQueries({ queryKey: ['containers', envId] })
    },
  })
}
