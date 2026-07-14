import { createFileRoute } from '@tanstack/react-router'
import { RunContainerForm } from '../../../../components/RunContainerForm'

export const Route = createFileRoute('/_app/environments/$id/containers/new')({
  head: () => ({ meta: [{ title: 'Run a container · Rivly' }] }),
  component: RunContainerPage,
})

function RunContainerPage() {
  const { id } = Route.useParams()
  return <RunContainerForm envId={Number(id)} />
}
