import type { StackState } from '../lib/stacks'
import styles from './StackStateBadge.module.css'

const LABEL: Record<StackState, string> = {
  running: 'Running',
  partial: 'Partial',
  stopped: 'Stopped',
}

const TONE: Record<StackState, string> = {
  running: styles.running,
  partial: styles.partial,
  stopped: styles.stopped,
}

export function StackStateBadge({ state }: { state: StackState }) {
  return (
    <span className={`${styles.state} ${TONE[state]}`}>
      <span className={styles.dot} />
      {LABEL[state]}
    </span>
  )
}
