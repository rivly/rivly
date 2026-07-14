import { createFileRoute } from '@tanstack/react-router'
import { StackDetail } from '../../../../components/StackDetail'

export const Route = createFileRoute('/_app/environments/$id/stacks/$name/')({
  head: () => ({ meta: [{ title: 'Stack · Rivly' }] }),
  component: StackDetailPage,
})

function StackDetailPage() {
  const { id, name } = Route.useParams()
  return <StackDetail envId={Number(id)} name={name} />
}
