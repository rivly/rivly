import { useEffect, useState } from 'react'
import { ApiError } from '../lib/api'
import {
  useCreateRegistry,
  useTestRegistry,
  useUpdateRegistry,
  type Registry,
} from '../lib/registries'
import { toast } from '../lib/toast'
import { Button } from './Button'
import { Field, FormDialog, TextField } from './FormDialog'

type Props = {
  open: boolean
  onClose: () => void
  editing: Registry | null
}

export function RegistryDialog({ open, onClose, editing }: Props) {
  const isEdit = editing !== null
  const create = useCreateRegistry()
  const update = useUpdateRegistry()
  const test = useTestRegistry()

  const [name, setName] = useState('')
  const [server, setServer] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (open) {
      setName(editing?.name ?? '')
      setServer(editing?.server ?? '')
      setUsername(editing?.username ?? '')
      setPassword('')
      setError(null)
    }
  }, [open, editing])

  const canSubmit =
    name.trim() !== '' &&
    server.trim() !== '' &&
    username.trim() !== '' &&
    (isEdit || password !== '')
  const canTest =
    server.trim() !== '' && username.trim() !== '' && (isEdit || password !== '')
  const pending = create.isPending || update.isPending

  const onError = (err: unknown) =>
    setError(err instanceof ApiError ? err.message : 'Could not save the registry.')

  const submit = () => {
    setError(null)
    if (isEdit && editing) {
      update.mutate(
        {
          id: editing.id,
          name: name.trim(),
          server: server.trim(),
          username: username.trim(),
          password,
        },
        {
          onSuccess: () => {
            toast.success(`Registry ${editing.server} updated`)
            onClose()
          },
          onError,
        },
      )
    } else {
      create.mutate(
        { name: name.trim(), server: server.trim(), username: username.trim(), password },
        {
          onSuccess: () => {
            toast.success(`Registry ${server.trim()} added`)
            onClose()
          },
          onError,
        },
      )
    }
  }

  const runTest = () => {
    test.mutate(
      {
        id: editing?.id,
        server: server.trim(),
        username: username.trim(),
        password,
      },
      {
        onSuccess: () => toast.success('Connection successful'),
        onError: (err) =>
          toast.error(
            'Connection failed',
            err instanceof ApiError ? err.message : 'Check the credentials',
          ),
      },
    )
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit registry' : 'Add a registry'}
      submitLabel={isEdit ? 'Save' : 'Add registry'}
      onSubmit={submit}
      pending={pending}
      error={error}
      canSubmit={canSubmit}
      extraAction={
        <Button
          type="button"
          variant="secondary"
          size="sm"
          loading={test.isPending}
          disabled={!canTest}
          onClick={runTest}
        >
          Test connection
        </Button>
      }
    >
      <Field label="Name" required hint="A friendly label to identify this registry">
        <TextField
          value={name}
          onChange={(e) => setName(e.target.value)}
          autoComplete="off"
          autoFocus
        />
      </Field>
      <Field label="Registry URL" required hint="e.g. ghcr.io, registry.gitlab.com, docker.io">
        <TextField value={server} onChange={(e) => setServer(e.target.value)} />
      </Field>
      <Field label="Username" required>
        <TextField
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          autoComplete="off"
        />
      </Field>
      <Field
        label="Password"
        required={!isEdit}
        hint={isEdit ? 'Leave blank to keep the current password' : undefined}
      >
        <TextField
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder={isEdit ? '••••••••' : ''}
          autoComplete="off"
        />
      </Field>
    </FormDialog>
  )
}
