import { useEffect, useState } from 'react'

export type ContainerStats = {
  cpuPercent: number
  memUsage: number
  memLimit: number
  memPercent: number
  netRx: number
  netTx: number
  blockRead: number
  blockWrite: number
  pids: number
}

export type StatsStatus = 'connecting' | 'streaming' | 'ended' | 'error'

export function useContainerStats(envId: number, containerId: string) {
  const [stats, setStats] = useState<ContainerStats | null>(null)
  const [status, setStatus] = useState<StatsStatus>('connecting')

  useEffect(() => {
    setStats(null)
    setStatus('connecting')

    const source = new EventSource(
      `/api/v1/environments/${envId}/containers/${containerId}/stats`,
    )

    source.onopen = () => setStatus('streaming')
    source.onmessage = (event) => {
      try {
        setStats(JSON.parse(event.data) as ContainerStats)
        setStatus('streaming')
      } catch {
        return
      }
    }
    source.addEventListener('end', () => {
      setStatus('ended')
      source.close()
    })
    source.onerror = () => setStatus('error')

    return () => source.close()
  }, [envId, containerId])

  return { stats, status }
}
