import styles from './FormError.module.css'

export function FormError({ message }: { message?: string }) {
  if (!message) {
    return null
  }
  return (
    <p className={styles.error} role="alert">
      {message}
    </p>
  )
}
