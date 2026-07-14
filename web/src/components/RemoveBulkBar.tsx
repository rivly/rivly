import { useState } from 'react'
import { AlertDialog } from '@base-ui/react/alert-dialog'
import { LuTrash2, LuX } from 'react-icons/lu'
import { Button } from './Button'
import styles from './ContainerBulkBar.module.css'

type Props = {
  selectedCount: number
  inUseCount: number
  noun: string
  pending: boolean
  onRemove: () => void
  clear: () => void
}

export function RemoveBulkBar({
  selectedCount,
  inUseCount,
  noun,
  pending,
  onRemove,
  clear,
}: Props) {
  const [confirm, setConfirm] = useState(false)
  const plural = selectedCount > 1 ? 's' : ''

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
      <span className={styles.count}>{selectedCount} selected</span>

      <div className={styles.actions}>
        <AlertDialog.Root open={confirm} onOpenChange={setConfirm}>
          <AlertDialog.Trigger
            render={
              <Button variant="danger" size="sm" icon={<LuTrash2 />} loading={pending}>
                Remove
              </Button>
            }
          />
          <AlertDialog.Portal>
            <AlertDialog.Backdrop className={styles.backdrop} />
            <AlertDialog.Popup className={styles.dialog}>
              <AlertDialog.Title className={styles.dialogTitle}>
                Remove {selectedCount} {noun}
                {plural}?
              </AlertDialog.Title>
              <AlertDialog.Description className={styles.dialogText}>
                {inUseCount > 0
                  ? `${inUseCount} of them ${inUseCount > 1 ? 'are' : 'is'} in use and will be skipped. This cannot be undone.`
                  : `This permanently removes the selected ${noun}s. This cannot be undone.`}
              </AlertDialog.Description>
              <div className={styles.dialogActions}>
                <AlertDialog.Close render={<Button variant="secondary" size="sm">Cancel</Button>} />
                <Button
                  variant="danger"
                  size="sm"
                  onClick={() => {
                    setConfirm(false)
                    onRemove()
                  }}
                >
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
