import { useEffect, useState } from 'react'
import { ApiError } from '../lib/api'
import { useCreateVolume } from '../lib/volumes'
import { toast } from '../lib/toast'
import { Field, FormDialog, TextField } from './FormDialog'

type Props = {
  envId: number
  open: boolean
  onClose: () => void
}

export function CreateVolumeDialog({ envId, open, onClose }: Props) {
  const mutation = useCreateVolume(envId)
  const [name, setName] = useState('')
  const [driver, setDriver] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (open) {
      setName('')
      setDriver('')
      setError(null)
    }
  }, [open])

  const submit = () => {
    const trimmed = name.trim()
    if (trimmed === '') {
      return
    }
    setError(null)
    mutation.mutate(
      { name: trimmed, driver: driver.trim() || undefined },
      {
        onSuccess: () => {
          toast.success(`Volume ${trimmed} created`)
          onClose()
        },
        onError: (err) =>
          setError(err instanceof ApiError ? err.message : 'Could not create the volume.'),
      },
    )
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title="Create a volume"
      submitLabel="Create volume"
      onSubmit={submit}
      pending={mutation.isPending}
      error={error}
      canSubmit={name.trim() !== ''}
    >
      <Field label="Name" required>
        <TextField
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="app_data"
          autoFocus
        />
      </Field>
      <Field label="Driver" hint="Defaults to local.">
        <TextField
          value={driver}
          onChange={(event) => setDriver(event.target.value)}
          placeholder="local"
        />
      </Field>
    </FormDialog>
  )
}
