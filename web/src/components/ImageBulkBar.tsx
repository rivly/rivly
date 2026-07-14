import { useImageActions, type Image } from '../lib/images'
import { bulkActionError, reportBulk } from '../lib/bulk'
import { RemoveBulkBar } from './RemoveBulkBar'

type Props = {
  envId: number
  selected: Image[]
  clear: () => void
}

export function ImageBulkBar({ envId, selected, clear }: Props) {
  const mutation = useImageActions(envId)

  const remove = () => {
    mutation.mutate(
      { action: 'remove', ids: selected.map((image) => image.id) },
      {
        onSuccess: (data) =>
          reportBulk(data.results, {
            verb: 'Removed',
            noun: 'image',
            failHint: (failed) => `${failed} still in use or shared`,
            clear,
          }),
        onError: bulkActionError,
      },
    )
  }

  return (
    <RemoveBulkBar
      selectedCount={selected.length}
      inUseCount={selected.filter((image) => image.inUse).length}
      noun="image"
      pending={mutation.isPending}
      onRemove={remove}
      clear={clear}
    />
  )
}
