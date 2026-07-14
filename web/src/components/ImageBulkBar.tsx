import { useState } from 'react'
import { AlertDialog } from '@base-ui/react/alert-dialog'
import { LuTrash2, LuX } from 'react-icons/lu'
import { useImageActions, type Image } from '../lib/images'
import { toast } from '../lib/toast'
import { Button } from './Button'
import styles from './ContainerBulkBar.module.css'

type Props = {
  envId: number
  selected: Image[]
  clear: () => void
}

export function ImageBulkBar({ envId, selected, clear }: Props) {
  const mutation = useImageActions(envId)
  const [confirmRemove, setConfirmRemove] = useState(false)
  const inUseCount = selected.filter((image) => image.inUse).length

  const remove = () => {
    setConfirmRemove(false)
    mutation.mutate(
      { action: 'remove', ids: selected.map((image) => image.id) },
      {
        onSuccess: (data) => {
          const ok = data.results.filter((result) => result.ok).length
          const failed = data.results.length - ok
          if (failed === 0) {
            toast.success(`Removed ${ok} image${ok > 1 ? 's' : ''}`)
          } else {
            toast.error(
              `Removed ${ok}/${data.results.length} images`,
              `${failed} still in use or shared`,
            )
          }
          clear()
        },
        onError: () => toast.error('Action failed', 'Could not reach the environment'),
      },
    )
  }

  return (
    <div className={styles.bar}>
      <button
        type="button"
        className={styles.clear}
        onClick={clear}
        aria-label="Clear selection"
      >
        <LuX />
      </button>
      <span className={styles.count}>{selected.length} selected</span>

      <div className={styles.actions}>
        <AlertDialog.Root open={confirmRemove} onOpenChange={setConfirmRemove}>
          <AlertDialog.Trigger
            render={
              <Button variant="danger" size="sm" icon={<LuTrash2 />} loading={mutation.isPending}>
                Remove
              </Button>
            }
          />
          <AlertDialog.Portal>
            <AlertDialog.Backdrop className={styles.backdrop} />
            <AlertDialog.Popup className={styles.dialog}>
              <AlertDialog.Title className={styles.dialogTitle}>
                Remove {selected.length} image{selected.length > 1 ? 's' : ''}?
              </AlertDialog.Title>
              <AlertDialog.Description className={styles.dialogText}>
                {inUseCount > 0
                  ? `${inUseCount} of them ${inUseCount > 1 ? 'are' : 'is'} in use by a container and will be skipped. This cannot be undone.`
                  : 'This permanently removes the selected images. This cannot be undone.'}
              </AlertDialog.Description>
              <div className={styles.dialogActions}>
                <AlertDialog.Close render={<Button variant="secondary" size="sm">Cancel</Button>} />
                <Button variant="danger" size="sm" onClick={remove}>
                  Remove
                </Button>
              </div>
            </AlertDialog.Popup>
          </AlertDialog.Portal>
        </AlertDialog.Root>
      </div>
    </div>
  )
}
