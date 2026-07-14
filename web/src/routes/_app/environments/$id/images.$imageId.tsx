import { createFileRoute } from '@tanstack/react-router'
import { ImageDetail } from '../../../../components/ImageDetail'

export const Route = createFileRoute('/_app/environments/$id/images/$imageId')({
  head: () => ({ meta: [{ title: 'Image · Rivly' }] }),
  component: ImageDetailPage,
})

function ImageDetailPage() {
  const { id, imageId } = Route.useParams()
  return <ImageDetail envId={Number(id)} imageId={imageId} />
}
