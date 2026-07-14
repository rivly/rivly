import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'

export type GitCredential = {
  id: number
  name: string
  username: string
  createdBy: string
  createdAt: number
  updatedAt: number
}

export function gitCredentialsQueryOptions() {
  return queryOptions({
    queryKey: ['git-credentials'],
    queryFn: () => api.get<GitCredential[]>('/git-credentials'),
  })
}

export function useGitCredentials() {
  return useQuery(gitCredentialsQueryOptions())
}

export function useCreateGitCredential() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { name: string; username: string; token: string }) =>
      api.post<GitCredential>('/git-credentials', input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['git-credentials'] })
    },
  })
}

export function useUpdateGitCredential() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { id: number; name: string; username: string; token: string }) =>
      api.put<GitCredential>(`/git-credentials/${input.id}`, {
        name: input.name,
        username: input.username,
        token: input.token,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['git-credentials'] })
    },
  })
}

export function useDeleteGitCredential() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => api.del<void>(`/git-credentials/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['git-credentials'] })
    },
  })
}
