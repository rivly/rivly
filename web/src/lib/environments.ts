import { queryOptions, useQuery } from '@tanstack/react-query'
import { api } from './api'

export type EnvironmentStatus = 'up' | 'down'

export type Environment = {
  id: number
  name: string
  kind: string
  url: string
  status: EnvironmentStatus
  lastSeen?: number
}

export type SystemInfo = {
  serverVersion: string
  osType: string
  architecture: string
  kernelVersion: string
  operatingSystem: string
  name: string
  swarm: boolean
  nodes: number
  ncpu: number
  memTotal: number
  containers: number
  containersRunning: number
  containersPaused: number
  containersStopped: number
  images: number
}

export type EnvironmentDetail = Environment & { system?: SystemInfo }

export const environmentsQueryOptions = queryOptions({
  queryKey: ['environments'],
  queryFn: () => api.get<EnvironmentDetail[]>('/environments'),
})

export function environmentQueryOptions(id: number) {
  return queryOptions({
    queryKey: ['environments', id],
    queryFn: () => api.get<EnvironmentDetail>(`/environments/${id}`),
  })
}

export function useEnvironments() {
  return useQuery(environmentsQueryOptions)
}

export function useEnvironment(id: number) {
  return useQuery(environmentQueryOptions(id))
}
