import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'

export type Registry = {
  id: number
  name: string
  server: string
  username: string
  createdBy: string
  createdAt: number
  updatedAt: number
}

export function registriesQueryOptions() {
  return queryOptions({
    queryKey: ['registries'],
    queryFn: () => api.get<Registry[]>('/registries'),
  })
}

export function useRegistries() {
  return useQuery(registriesQueryOptions())
}

export function useCreateRegistry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { name: string; server: string; username: string; password: string }) =>
      api.post<Registry>('/registries', input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['registries'] })
    },
  })
}

export function useUpdateRegistry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: {
      id: number
      name: string
      server: string
      username: string
      password: string
    }) =>
      api.put<Registry>(`/registries/${input.id}`, {
        name: input.name,
        server: input.server,
        username: input.username,
        password: input.password,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['registries'] })
    },
  })
}

export function useDeleteRegistry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.del<void>(`/registries/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['registries'] })
    },
  })
}

export function useTestRegistry() {
  return useMutation({
    mutationFn: (input: {
      id?: number
      server: string
      username: string
      password: string
    }) => api.post<{ ok: boolean }>('/registries/test', input),
  })
}
