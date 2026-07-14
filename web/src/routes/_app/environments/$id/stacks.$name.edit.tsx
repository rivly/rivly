import { createFileRoute } from '@tanstack/react-router'
import { StackEditor } from '../../../../components/StackEditor'

export const Route = createFileRoute('/_app/environments/$id/stacks/$name/edit')({
  head: () => ({ meta: [{ title: 'Edit stack · Rivly' }] }),
  component: EditStackPage,
})

function EditStackPage() {
  const { id, name } = Route.useParams()
  return <StackEditor envId={Number(id)} name={name} />
}
