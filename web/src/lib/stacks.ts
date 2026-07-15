import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { api } from './api'
import type { ActionResult } from './containers'

export type StackState = 'running' | 'partial' | 'stopped'

export type StackSource = 'content' | 'git'

export type Stack = {
  name: string
  type: string
  source: StackSource
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

export type GitSource = {
  url: string
  ref: string
  path: string
  credentialId: number
  autoUpdate: boolean
  pollInterval: number
}

export type GitDetail = GitSource & {
  commit: string
  lastCheckedAt: number
  lastError: string
}

export type StackDetail = {
  name: string
  source: StackSource
  content: string
  env: EnvVar[]
  git: GitDetail | null
}

export function fetchStackContent(envId: number, name: string) {
  return api.get<StackDetail>(`/environments/${envId}/stacks/${name}`)
}

export function useStackDetail(envId: number, name: string, enabled: boolean) {
  return useQuery({
    queryKey: ['stack', envId, name],
    queryFn: () => fetchStackContent(envId, name),
    enabled,
  })
}

export type DeployStackInput = {
  name: string
  source: StackSource
  content: string
  env: EnvVar[]
  git?: GitSource
}

export function useDeployStack(envId: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: DeployStackInput) =>
      api.post<{ name: string }>(`/environments/${envId}/stacks`, input),
    onSuccess: (_result, input) => {
      queryClient.invalidateQueries({ queryKey: ['stacks', envId] })
      queryClient.invalidateQueries({ queryKey: ['containers', envId] })
      queryClient.invalidateQueries({ queryKey: ['stack', envId, input.name] })
    },
  })
}
