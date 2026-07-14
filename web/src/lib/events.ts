import { useEffect } from 'react'
import { useQueryClient, type QueryClient } from '@tanstack/react-query'
import type { EnvironmentDetail } from './environments'

type ServerEvent = { type: string; data: unknown }

const handlers: Record<
  string,
  (data: unknown, queryClient: QueryClient) => void
> = {
  'environment.updated': (data, queryClient) => {
    const env = data as EnvironmentDetail
    queryClient.setQueryData<EnvironmentDetail[]>(['environments'], (list) =>
      list ? list.map((e) => (e.id === env.id ? env : e)) : list,
    )
    queryClient.setQueryData(['environments', env.id], env)
    queryClient.invalidateQueries({ queryKey: ['containers', env.id] })
    queryClient.invalidateQueries({ queryKey: ['images', env.id] })
    queryClient.invalidateQueries({ queryKey: ['volumes', env.id] })
  },
}

export function useServerEvents() {
  const queryClient = useQueryClient()

  useEffect(() => {
    const source = new EventSource('/api/v1/events')

    source.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as ServerEvent
        handlers[message.type]?.(message.data, queryClient)
      } catch {
        return
      }
    }

    source.onopen = () => {
      queryClient.invalidateQueries({ queryKey: ['environments'] })
    }

    return () => source.close()
  }, [queryClient])
}
