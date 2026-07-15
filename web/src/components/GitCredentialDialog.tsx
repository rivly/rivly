import { useEffect, useState } from 'react'
import { ApiError } from '../lib/api'
import {
  useCreateGitCredential,
  useUpdateGitCredential,
  type GitCredential,
} from '../lib/gitCredentials'
import { toast } from '../lib/toast'
import { Field, FormDialog, TextField } from './FormDialog'

type Props = {
  open: boolean
  onClose: () => void
  editing: GitCredential | null
}

export function GitCredentialDialog({ open, onClose, editing }: Props) {
  const isEdit = editing !== null
  const create = useCreateGitCredential()
  const update = useUpdateGitCredential()

  const [name, setName] = useState('')
  const [username, setUsername] = useState('')
  const [token, setToken] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (open) {
      setName(editing?.name ?? '')
      setUsername(editing?.username ?? '')
      setToken('')
      setError(null)
    }
  }, [open, editing])

  const canSubmit = name.trim() !== '' && username.trim() !== '' && (isEdit || token !== '')
  const pending = create.isPending || update.isPending

  const onError = (err: unknown) =>
    setError(err instanceof ApiError ? err.message : 'Could not save the credential.')

  const submit = () => {
    setError(null)
    if (isEdit && editing) {
      update.mutate(
        { id: editing.id, name: name.trim(), username: username.trim(), token },
        {
          onSuccess: () => {
            toast.success(`Credential ${name.trim()} updated`)
            onClose()
          },
          onError,
        },
      )
    } else {
      create.mutate(
        { name: name.trim(), username: username.trim(), token },
        {
          onSuccess: () => {
            toast.success(`Credential ${name.trim()} added`)
            onClose()
          },
          onError,
        },
      )
    }
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit credential' : 'Add a credential'}
      submitLabel={isEdit ? 'Save' : 'Add credential'}
      onSubmit={submit}
      pending={pending}
      error={error}
      canSubmit={canSubmit}
    >
      <Field label="Name" required hint="A friendly label to identify this credential">
        <TextField value={name} onChange={(e) => setName(e.target.value)} autoFocus />
      </Field>
      <Field label="Username" required hint="Your Git provider username.">
        <TextField value={username} onChange={(e) => setUsername(e.target.value)} />
      </Field>
      <Field
        label="Token"
        required={!isEdit}
        hint={
          isEdit
            ? 'Leave blank to keep the current token'
            : 'Personal access token with read access to the repositories'
        }
      >
        <TextField
          type="password"
          value={token}
          onChange={(e) => setToken(e.target.value)}
          placeholder={isEdit ? '••••••••' : ''}
        />
      </Field>
    </FormDialog>
  )
}
