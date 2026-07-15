import { Field as BaseField } from '@base-ui/react/field'
import type { ComponentProps, ReactNode } from 'react'
import { RequiredMark } from './RequiredMark'
import styles from './Field.module.css'

type Props = Omit<ComponentProps<typeof BaseField.Control>, 'className'> & {
  label: string
  name: string
  hint?: ReactNode
}

export function Field({ label, name, hint, ...props }: Props) {
  return (
    <BaseField.Root name={name} className={styles.root}>
      <BaseField.Label className={styles.label}>
        {label}
        {props.required && <RequiredMark />}
      </BaseField.Label>
      <BaseField.Control className={styles.control} {...props} />
      {hint && (
        <BaseField.Description className={styles.hint}>
          {hint}
        </BaseField.Description>
      )}
      <BaseField.Error className={styles.error} match="valueMissing">
        This field is required.
      </BaseField.Error>
      <BaseField.Error className={styles.error} match="typeMismatch">
        Please enter a valid email address.
      </BaseField.Error>
    </BaseField.Root>
  )
}
