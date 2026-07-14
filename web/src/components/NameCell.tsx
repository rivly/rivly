import type { ReactNode } from 'react'
import { UnusedBadge } from './UnusedBadge'
import styles from './NameCell.module.css'

export function NameCell({ children, inUse }: { children: ReactNode; inUse: boolean }) {
  return (
    <span className={styles.cell}>
      <span className={styles.name}>{children}</span>
      {!inUse && <UnusedBadge />}
    </span>
  )
}
