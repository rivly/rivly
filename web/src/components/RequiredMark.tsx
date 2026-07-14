import styles from './RequiredMark.module.css'

export function RequiredMark() {
  return (
    <span className={styles.mark} aria-hidden>
      *
    </span>
  )
}
