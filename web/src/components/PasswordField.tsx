import { Field as BaseField } from '@base-ui/react/field'
import { useState, type ComponentProps, type ReactNode } from 'react'
import { LuEye, LuEyeOff } from 'react-icons/lu'
import { RequiredMark } from './RequiredMark'
import styles from './PasswordField.module.css'

type Props = Omit<ComponentProps<typeof BaseField.Control>, 'className' | 'type'> & {
  label: string
  name: string
  action?: ReactNode
}

export function PasswordField({ label, name, action, ...props }: Props) {
  const [visible, setVisible] = useState(false)
  return (
    <BaseField.Root name={name} className={styles.root}>
      <div className={styles.labelRow}>
        <BaseField.Label className={styles.label}>
          {label}
          {props.required && <RequiredMark />}
        </BaseField.Label>
        {action}
      </div>
      <div className={styles.controlWrap}>
        <BaseField.Control
          type={visible ? 'text' : 'password'}
          className={styles.control}
          {...props}
        />
        <button
          type="button"
          className={styles.toggle}
          onClick={() => setVisible((v) => !v)}
          aria-label={visible ? 'Hide password' : 'Show password'}
        >
          {visible ? <LuEyeOff /> : <LuEye />}
        </button>
      </div>
      <BaseField.Error className={styles.error} match="valueMissing">
        This field is required.
      </BaseField.Error>
      <BaseField.Error className={styles.error} match="tooShort">
        Password must be at least 8 characters.
      </BaseField.Error>
    </BaseField.Root>
  )
}
