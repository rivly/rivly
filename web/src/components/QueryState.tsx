import type { ReactNode } from 'react'
import { Loader } from './Loader'
import styles from './QueryState.module.css'

type Props = {
  pending: boolean
  error: boolean
  errorMessage: string
  children: ReactNode
}

export function QueryState({ pending, error, errorMessage, children }: Props) {
  if (pending) {
    return <Loader />
  }
  if (error) {
    return <p className={styles.message}>{errorMessage}</p>
  }
  return <>{children}</>
}
