import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/')({
  component: () => (
    <main>
      <h1>Rivly</h1>
      <p>Dashboard under construction.</p>
    </main>
  ),
})
