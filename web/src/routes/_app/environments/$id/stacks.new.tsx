import { createFileRoute } from '@tanstack/react-router'
import { StackEditor } from '../../../../components/StackEditor'

export const Route = createFileRoute('/_app/environments/$id/stacks/new')({
  head: () => ({ meta: [{ title: 'Deploy a stack · Rivly' }] }),
  component: DeployStackPage,
})

function DeployStackPage() {
  const { id } = Route.useParams()
  return <StackEditor envId={Number(id)} />
}
