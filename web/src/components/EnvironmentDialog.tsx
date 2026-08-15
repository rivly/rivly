import { useEffect, useState } from 'react'
import { ApiError } from '../lib/api'
import {
  useCreateEnvironment,
  useUpdateEnvironment,
  type EnvironmentDetail,
} from '../lib/environments'
import { toast } from '../lib/toast'
import { Field, FormDialog, TextField } from './FormDialog'

type Props = {
  open: boolean
  onClose: () => void
  editing: EnvironmentDetail | null
}

export function EnvironmentDialog({ open, onClose, editing }: Props) {
  const isEdit = editing !== null
  const create = useCreateEnvironment()
  const update = useUpdateEnvironment()

  const [name, setName] = useState('')
  const [url, setUrl] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (open) {
      setName(editing?.name ?? '')
      setUrl(editing?.url ?? '')
      setError(null)
    }
  }, [open, editing])

  const canSubmit = name.trim() !== '' && url.trim() !== ''
  const pending = create.isPending || update.isPending

  const onError = (err: unknown) =>
    setError(
      err instanceof ApiError ? err.message : 'Could not save the environment.',
    )

  const submit = () => {
    setError(null)
    const input = { name: name.trim(), url: url.trim() }
    if (isEdit && editing) {
      update.mutate(
        { id: editing.id, ...input },
        {
          onSuccess: () => {
            toast.success(`Environment ${input.name} updated`)
            onClose()
          },
          onError,
        },
      )
      return
    }
    create.mutate(input, {
      onSuccess: () => {
        toast.success(`Environment ${input.name} added`)
        onClose()
      },
      onError,
    })
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit environment' : 'Add an environment'}
      submitLabel={isEdit ? 'Save' : 'Add environment'}
      onSubmit={submit}
      pending={pending}
      error={error}
      canSubmit={canSubmit}
    >
      <Field label="Name" required hint="A friendly label to identify this host">
        <TextField
          value={name}
          onChange={(e) => setName(e.target.value)}
          autoComplete="off"
          autoFocus
        />
      </Field>
      <Field label="Endpoint" required>
        <TextField
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          autoComplete="off"
          spellCheck={false}
        />
      </Field>
    </FormDialog>
  )
}
