import { createFileRoute } from '@tanstack/react-router'
import { NetworkDetail } from '../../../../components/NetworkDetail'

export const Route = createFileRoute('/_app/environments/$id/networks/$networkId')({
  head: () => ({ meta: [{ title: 'Network · Rivly' }] }),
  component: NetworkDetailPage,
})

function NetworkDetailPage() {
  const { id, networkId } = Route.useParams()
  return <NetworkDetail envId={Number(id)} networkId={networkId} />
}
