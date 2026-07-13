import { createFileRoute } from '@tanstack/react-router'
import { useMe } from '../../lib/auth'
import styles from './index.module.css'

export const Route = createFileRoute('/_app/')({
  component: HomePage,
})

function HomePage() {
  const { data: me } = useMe()

  return (
    <div className={styles.page}>
      <header className={styles.head}>
        <h1 className={styles.title}>Home</h1>
        <p className={styles.subtitle}>
          Signed in as {me?.email}.
        </p>
      </header>
    </div>
  )
}
