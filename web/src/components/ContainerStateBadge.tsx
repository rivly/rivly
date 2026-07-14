import styles from './ContainerStateBadge.module.css'

const TONE: Record<string, string> = {
  running: styles.running,
  paused: styles.paused,
  restarting: styles.paused,
  created: styles.info,
  exited: styles.danger,
  removing: styles.neutral,
  dead: styles.danger,
}

export function ContainerStateBadge({ state }: { state: string }) {
  return (
    <span className={`${styles.badge} ${TONE[state] ?? styles.neutral}`}>
      <span className={styles.dot} />
      {state}
    </span>
  )
}
