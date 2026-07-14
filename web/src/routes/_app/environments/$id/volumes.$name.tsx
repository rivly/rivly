import { createFileRoute } from '@tanstack/react-router'
import { VolumeDetail } from '../../../../components/VolumeDetail'

export const Route = createFileRoute('/_app/environments/$id/volumes/$name')({
  head: () => ({ meta: [{ title: 'Volume · Rivly' }] }),
  component: VolumeDetailPage,
})

function VolumeDetailPage() {
  const { id, name } = Route.useParams()
  return <VolumeDetail envId={Number(id)} name={name} />
}
