import type { ContainerPort } from '../lib/containers'
import { Tooltip } from './Tooltip'
import styles from './PublishedPorts.module.css'

export function PublishedPorts({ ports }: { ports: ContainerPort[] }) {
  if (ports.length === 0) {
    return <span className={styles.muted}>-</span>
  }

  const seen = new Set<string>()
  const unique = ports.filter((port) => {
    const key = `${port.publicPort}:${port.privatePort}/${port.type}`
    if (seen.has(key)) {
      return false
    }
    seen.add(key)
    return true
  })

  return (
    <span className={styles.ports}>
      {unique.map((port) => (
        <Tooltip
          key={`${port.publicPort}:${port.privatePort}/${port.type}`}
          content={portDetail(port)}
        >
          <span className={styles.portTag}>{portLabel(port)}</span>
        </Tooltip>
      ))}
    </span>
  )
}

function portLabel(port: ContainerPort): string {
  return port.publicPort ? `${port.publicPort}:${port.privatePort}` : `${port.privatePort}`
}

function portDetail(port: ContainerPort): string {
  if (port.publicPort) {
    return `${port.ip || '0.0.0.0'}:${port.publicPort} → ${port.privatePort}/${port.type}`
  }
  return `${port.privatePort}/${port.type} · not published`
}
