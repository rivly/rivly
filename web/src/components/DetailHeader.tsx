import type { ReactNode } from 'react'
import styles from './DetailHeader.module.css'

export function DetailHeader({
  title,
  badges,
  actions,
}: {
  title: string
  badges?: ReactNode
  actions?: ReactNode
}) {
  return (
    <header className={styles.head}>
      <div className={styles.heading}>
        <h1 className={styles.title}>{title}</h1>
        {badges}
      </div>
      {actions && <div className={styles.actions}>{actions}</div>}
    </header>
  )
}
