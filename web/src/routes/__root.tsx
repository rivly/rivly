import {
  createRootRouteWithContext,
  HeadContent,
  Outlet,
} from '@tanstack/react-router'
import type { QueryClient } from '@tanstack/react-query'

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  head: () => ({ meta: [{ title: 'Rivly' }] }),
  component: RootLayout,
})

function RootLayout() {
  return (
    <>
      <HeadContent />
      <Outlet />
    </>
  )
}
