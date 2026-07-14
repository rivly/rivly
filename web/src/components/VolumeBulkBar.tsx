import { useVolumeActions, type Volume } from '../lib/volumes'
import { toast } from '../lib/toast'
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
        onSuccess: (data) => {
          const ok = data.results.filter((result) => result.ok).length
          const failed = data.results.length - ok
          if (failed === 0) {
            toast.success(`Removed ${ok} volume${ok > 1 ? 's' : ''}`)
          } else {
            toast.error(
              `Removed ${ok}/${data.results.length} volumes`,
              `${failed} still in use`,
            )
          }
          clear()
        },
        onError: () => toast.error('Action failed', 'Could not reach the environment'),
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
