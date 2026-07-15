import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
} from '@tanstack/react-query'
import { api, ApiError } from './api'
import { toast } from './toast'

export type User = {
  id: number
  email: string
  displayName: string
  role: string
}

export type Credentials = {
  email: string
  password: string
  displayName?: string
}

export type SetupCredentials = Credentials & {
  token: string
}

export const setupQueryOptions = queryOptions({
  queryKey: ['setup'],
  queryFn: () => api.get<{ needsSetup: boolean }>('/setup'),
})

export const meQueryOptions = queryOptions({
  queryKey: ['me'],
  queryFn: async () => {
    try {
      return await api.get<User>('/me')
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        return null
      }
      throw err
    }
  },
  retry: false,
})

export async function loadAuthState(queryClient: QueryClient) {
  const [setup, me] = await Promise.all([
    queryClient.ensureQueryData(setupQueryOptions),
    queryClient.ensureQueryData(meQueryOptions),
  ])
  return { needsSetup: setup.needsSetup, me }
}

export function useMe() {
  return useQuery(meQueryOptions)
}

export function useSetup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (credentials: SetupCredentials) =>
      api.post<User>('/setup', credentials),
    onSuccess: (user) => {
      queryClient.setQueryData(['me'], user)
      queryClient.setQueryData(['setup'], { needsSetup: false })
    },
  })
}

export function useLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (credentials: Credentials) =>
      api.post<User>('/login', credentials),
    onSuccess: (user) => {
      queryClient.setQueryData(['me'], user)
    },
  })
}

export function useUpdateProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { displayName: string }) => api.put<User>('/me', input),
    onSuccess: (user) => {
      queryClient.setQueryData(['me'], user)
    },
  })
}

export function useChangePassword() {
  return useMutation({
    mutationFn: (input: { currentPassword: string; newPassword: string }) =>
      api.post<void>('/me/password', input),
  })
}

export function useLogout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<void>('/logout'),
    onSuccess: () => {
      queryClient.setQueryData(['me'], null)
      toast.success('Signed out', 'You have been logged out of Rivly.')
    },
  })
}
