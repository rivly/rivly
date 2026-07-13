import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { AppShell } from '../components/AppShell'
import { loadAuthState } from '../lib/auth'

export const Route = createFileRoute('/_app')({
  beforeLoad: async ({ context }) => {
    const { needsSetup, me } = await loadAuthState(context.queryClient)
    if (!me) {
      throw redirect({ to: needsSetup ? '/setup' : '/login' })
    }
  },
  component: AppLayout,
})

function AppLayout() {
  return (
    <AppShell>
      <Outlet />
    </AppShell>
  )
}
