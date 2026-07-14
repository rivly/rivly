import type { InputHTMLAttributes, ReactNode } from 'react'
import { Dialog } from '@base-ui/react/dialog'
import { LuX } from 'react-icons/lu'
import { Button } from './Button'
import styles from './FormDialog.module.css'

type Props = {
  open: boolean
  onClose: () => void
  title: string
  submitLabel: string
  onSubmit: () => void
  pending?: boolean
  error?: string | null
  canSubmit?: boolean
  children: ReactNode
}

export function FormDialog({
  open,
  onClose,
  title,
  submitLabel,
  onSubmit,
  pending = false,
  error,
  canSubmit = true,
  children,
}: Props) {
  return (
    <Dialog.Root
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onClose()
        }
      }}
    >
      <Dialog.Portal>
        <Dialog.Backdrop className={styles.backdrop} />
        <Dialog.Popup className={styles.popup}>
          <header className={styles.header}>
            <Dialog.Title className={styles.title}>{title}</Dialog.Title>
            <Dialog.Close
              render={<Button variant="secondary" size="sm" iconOnly icon={<LuX />} aria-label="Close" />}
            />
          </header>
          <form
            className={styles.form}
            onSubmit={(event) => {
              event.preventDefault()
              onSubmit()
            }}
          >
            <div className={styles.body}>
              {children}
              {error && <p className={styles.error}>{error}</p>}
            </div>
            <div className={styles.footer}>
              <Dialog.Close render={<Button type="button" variant="secondary" size="sm">Cancel</Button>} />
              <Button type="submit" size="sm" loading={pending} disabled={!canSubmit}>
                {submitLabel}
              </Button>
            </div>
          </form>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

export function Field({
  label,
  optional,
  hint,
  children,
}: {
  label: string
  optional?: boolean
  hint?: string
  children: ReactNode
}) {
  return (
    <label className={styles.field}>
      <span className={styles.label}>
        {label}
        {optional && <span className={styles.optional}>optional</span>}
      </span>
      {children}
      {hint && <span className={styles.hint}>{hint}</span>}
    </label>
  )
}

export function TextField(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={styles.input} autoComplete="off" spellCheck={false} {...props} />
}
