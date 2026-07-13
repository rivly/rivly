import type { EnvironmentStatus } from '../lib/environments'
import styles from './StatusBadge.module.css'

type Props = {
  status: EnvironmentStatus
}

export function StatusBadge({ status }: Props) {
  return (
    <span className={`${styles.badge} ${status === 'up' ? styles.up : styles.down}`}>
      <span className={styles.dot} />
      {status === 'up' ? 'Up' : 'Down'}
    </span>
  )
}
