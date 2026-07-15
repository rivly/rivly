import { createFileRoute } from '@tanstack/react-router'
import { useState, type FormEvent } from 'react'
import { Form } from '@base-ui/react/form'
import { Button } from '../../components/Button'
import { Field } from '../../components/Field'
import { FormError } from '../../components/FormError'
import { PageHeader } from '../../components/PageHeader'
import { PasswordField } from '../../components/PasswordField'
import { ApiError } from '../../lib/api'
import { useChangePassword, useMe, useUpdateProfile } from '../../lib/auth'
import { toast } from '../../lib/toast'
import styles from './account.module.css'

export const Route = createFileRoute('/_app/account')({
  head: () => ({ meta: [{ title: 'Account · Rivly' }] }),
  component: AccountPage,
})

function AccountPage() {
  const { data: me } = useMe()

  return (
    <div>
      <PageHeader
        title="Account"
        subtitle="Your name as it appears on the stacks you deploy, and your password."
      />
      <div className={styles.sections}>
        <ProfileSection key={me?.displayName} name={me?.displayName ?? ''} email={me?.email ?? ''} />
        <PasswordSection />
      </div>
    </div>
  )
}

function ProfileSection({ name, email }: { name: string; email: string }) {
  const update = useUpdateProfile()
  const [displayName, setDisplayName] = useState(name)
  const [error, setError] = useState('')

  const changed = displayName.trim() !== name && displayName.trim() !== ''

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    if (!event.currentTarget.checkValidity()) {
      return
    }
    update.mutate(
      { displayName: displayName.trim() },
      {
        onSuccess: () => toast.success('Name updated'),
        onError: (err) =>
          setError(err instanceof ApiError ? err.message : 'Something went wrong'),
      },
    )
  }

  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>Profile</h2>
      <Form className={styles.form} onSubmit={onSubmit} noValidate>
        <Field
          label="Name"
          name="displayName"
          autoComplete="name"
          value={displayName}
          onChange={(event) => setDisplayName(event.target.value)}
          maxLength={100}
          required
        />
        <Field label="Email" name="email" value={email} readOnly />
        <p className={styles.hint}>
          Your email is used to sign in and cannot be changed.
        </p>
        <FormError message={error} />
        <div className={styles.actions}>
          <Button type="submit" loading={update.isPending} disabled={!changed}>
            Save
          </Button>
        </div>
      </Form>
    </section>
  )
}

function PasswordSection() {
  const change = useChangePassword()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')

  const filled = current !== '' && next !== '' && confirm !== ''

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    if (!event.currentTarget.checkValidity()) {
      return
    }
    if (next !== confirm) {
      setError('The new passwords do not match')
      return
    }
    change.mutate(
      { currentPassword: current, newPassword: next },
      {
        onSuccess: () => {
          toast.success('Password changed', 'Other sessions have been signed out.')
          setCurrent('')
          setNext('')
          setConfirm('')
        },
        onError: (err) =>
          setError(err instanceof ApiError ? err.message : 'Something went wrong'),
      },
    )
  }

  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>Password</h2>
      <Form className={styles.form} onSubmit={onSubmit} noValidate>
        <PasswordField
          label="Current password"
          name="currentPassword"
          autoComplete="current-password"
          value={current}
          onChange={(event) => setCurrent(event.target.value)}
          required
        />
        <PasswordField
          label="New password"
          name="newPassword"
          autoComplete="new-password"
          value={next}
          onChange={(event) => setNext(event.target.value)}
          minLength={8}
          required
        />
        <PasswordField
          label="Confirm new password"
          name="confirmPassword"
          autoComplete="new-password"
          value={confirm}
          onChange={(event) => setConfirm(event.target.value)}
          minLength={8}
          required
        />
        <p className={styles.hint}>
          At least 8 characters. Changing it signs out every other session.
        </p>
        <FormError message={error} />
        <div className={styles.actions}>
          <Button type="submit" loading={change.isPending} disabled={!filled}>
            Change password
          </Button>
        </div>
      </Form>
    </section>
  )
}
