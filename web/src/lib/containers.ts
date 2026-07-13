import { queryOptions, useQuery } from '@tanstack/react-query'
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
