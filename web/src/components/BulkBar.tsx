import type { ReactNode } from 'react'
import { LuX } from 'react-icons/lu'
import styles from './BulkBar.module.css'

export function BulkBar({
  count,
  clear,
  children,
}: {
  count: number
  clear: () => void
  children: ReactNode
}) {
  return (
    <div className={styles.bar}>
      <button
        type="button"
        className={styles.clear}
        onClick={clear}
        aria-label="Clear selection"
      >
        <LuX />
      </button>
      <span className={styles.count}>{count} selected</span>
      <div className={styles.actions}>{children}</div>
    </div>
  )
}
