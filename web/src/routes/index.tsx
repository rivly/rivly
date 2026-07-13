import { createFileRoute, redirect, useNavigate } from '@tanstack/react-router'
import { Button } from '../components/Button'
import { loadAuthState, useLogout, useMe } from '../lib/auth'
import styles from './index.module.css'

export const Route = createFileRoute('/')({
  beforeLoad: async ({ context }) => {
    const { needsSetup, me } = await loadAuthState(context.queryClient)
    if (!me) {
      throw redirect({ to: needsSetup ? '/setup' : '/login' })
    }
  },
  component: DashboardPage,
})

function DashboardPage() {
  const navigate = useNavigate()
  const { data: me } = useMe()
  const logout = useLogout()

  return (
    <main className={styles.wrap}>
      <h1>Rivly</h1>
      <p className={styles.muted}>
        Signed in as {me?.email}.
      </p>
      <Button
        variant="secondary"
        loading={logout.isPending}
        onClick={() =>
          logout.mutate(undefined, {
            onSuccess: () => navigate({ to: '/login' }),
          })
        }
      >
        Log out
      </Button>
    </main>
  )
}
