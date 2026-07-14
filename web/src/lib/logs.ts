import { useEffect, useRef, useState } from 'react'

export type LogStream = 'stdout' | 'stderr'

export type LogLine = {
  id: number
  stream: LogStream
  message: string
}

export type LogStatus = 'connecting' | 'streaming' | 'ended' | 'error'

const MAX_LINES = 5000

export function useContainerLogs(envId: number, containerId: string, tail = 200) {
  const [lines, setLines] = useState<LogLine[]>([])
  const [status, setStatus] = useState<LogStatus>('connecting')
  const nextId = useRef(0)

  useEffect(() => {
    setLines([])
    setStatus('connecting')
    nextId.current = 0

    const url = `/api/v1/environments/${envId}/containers/${containerId}/logs?tail=${tail}`
    const source = new EventSource(url)

    source.onopen = () => setStatus('streaming')

    source.onmessage = (event) => {
      let parsed: { stream: LogStream; message: string }
      try {
        parsed = JSON.parse(event.data)
      } catch {
        return
      }
      setLines((prev) => {
        const next = [...prev, { ...parsed, id: nextId.current++ }]
        return next.length > MAX_LINES ? next.slice(next.length - MAX_LINES) : next
      })
    }

    source.addEventListener('end', () => {
      setStatus('ended')
      source.close()
    })

    source.onerror = () => setStatus('error')

    return () => source.close()
  }, [envId, containerId, tail])

  return { lines, status }
}
