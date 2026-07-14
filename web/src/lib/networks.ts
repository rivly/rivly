import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { api } from './api'
import type { ActionResult } from './containers'

export type Network = {
  id: string
  name: string
  driver: string
  scope: string
  stack: string
  created: number
  inUse: boolean
}

export function networksQueryOptions(envId: number) {
  return queryOptions({
    queryKey: ['networks', envId],
    queryFn: () => api.get<Network[]>(`/environments/${envId}/networks`),
  })
}

export function useNetworks(envId: number) {
  return useQuery(networksQueryOptions(envId))
}

export type NetworkSubnet = { subnet: string; gateway: string }
export type NetworkContainer = { id: string; name: string; ipv4: string }

export type NetworkDetail = {
  id: string
  name: string
  driver: string
  scope: string
  internal: boolean
  attachable: boolean
  created: number
  subnets: NetworkSubnet[]
  labels: Record<string, string> | null
  containers: NetworkContainer[]
}

export function useNetworkDetail(envId: number, networkId: string) {
  return useQuery({
    queryKey: ['network', envId, networkId],
    queryFn: () =>
      api.get<NetworkDetail>(`/environments/${envId}/networks/${networkId}`),
  })
}

export type CreateNetworkInput = { name: string; driver?: string; subnet?: string }

export function useCreateNetwork(envId: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateNetworkInput) =>
      api.post<{ id: string; name: string; warning?: string }>(
        `/environments/${envId}/networks`,
        input,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['networks', envId] })
    },
  })
}

export type NetworkAction = 'remove'

export function useNetworkActions(envId: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { action: NetworkAction; ids: string[] }) =>
      api.post<{ results: ActionResult[] }>(
        `/environments/${envId}/networks/actions`,
        input,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['networks', envId] })
    },
  })
}
