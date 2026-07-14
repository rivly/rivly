import { useVolumeActions, type Volume } from '../lib/volumes'
import { bulkActionError, reportBulk } from '../lib/bulk'
import { RemoveBulkBar } from './RemoveBulkBar'

type Props = {
  envId: number
  selected: Volume[]
  clear: () => void
}

export function VolumeBulkBar({ envId, selected, clear }: Props) {
  const mutation = useVolumeActions(envId)

  const remove = () => {
    mutation.mutate(
      { action: 'remove', ids: selected.map((volume) => volume.name) },
      {
        onSuccess: (data) =>
          reportBulk(data.results, {
            verb: 'Removed',
            noun: 'volume',
            failHint: (failed) => `${failed} still in use`,
            clear,
          }),
        onError: bulkActionError,
      },
    )
  }

  return (
    <RemoveBulkBar
      selectedCount={selected.length}
      inUseCount={selected.filter((volume) => volume.inUse).length}
      noun="volume"
      pending={mutation.isPending}
      onRemove={remove}
      clear={clear}
    />
  )
}
