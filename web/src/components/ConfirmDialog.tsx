import type { ReactElement, ReactNode } from 'react'
import { AlertDialog } from '@base-ui/react/alert-dialog'
import { Button } from './Button'
import styles from './ConfirmDialog.module.css'

type Props = {
  trigger: ReactElement
  title: ReactNode
  description: ReactNode
  confirmLabel?: string
  confirmVariant?: 'danger' | 'primary'
  onConfirm: () => void
}

export function ConfirmDialog({
  trigger,
  title,
  description,
  confirmLabel = 'Remove',
  confirmVariant = 'danger',
  onConfirm,
}: Props) {
  return (
    <AlertDialog.Root>
      <AlertDialog.Trigger render={trigger} />
      <AlertDialog.Portal>
        <AlertDialog.Backdrop className={styles.backdrop} />
        <AlertDialog.Popup className={styles.dialog}>
          <AlertDialog.Title className={styles.title}>{title}</AlertDialog.Title>
          <AlertDialog.Description className={styles.text}>{description}</AlertDialog.Description>
          <div className={styles.actions}>
            <AlertDialog.Close render={<Button variant="secondary" size="sm">Cancel</Button>} />
            <AlertDialog.Close
              render={
                <Button variant={confirmVariant} size="sm" onClick={onConfirm}>
                  {confirmLabel}
                </Button>
              }
            />
          </div>
        </AlertDialog.Popup>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  )
}
