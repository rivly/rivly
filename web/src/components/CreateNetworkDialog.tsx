import { useEffect, useState } from 'react'
import { ApiError } from '../lib/api'
import { useCreateNetwork } from '../lib/networks'
import { toast } from '../lib/toast'
import { Field, FormDialog, TextField } from './FormDialog'

type Props = {
  envId: number
  open: boolean
  onClose: () => void
}

export function CreateNetworkDialog({ envId, open, onClose }: Props) {
  const mutation = useCreateNetwork(envId)
  const [name, setName] = useState('')
  const [driver, setDriver] = useState('')
  const [subnet, setSubnet] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (open) {
      setName('')
      setDriver('')
      setSubnet('')
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
      {
        name: trimmed,
        driver: driver.trim() || undefined,
        subnet: subnet.trim() || undefined,
      },
      {
        onSuccess: () => {
          toast.success(`Network ${trimmed} created`)
          onClose()
        },
        onError: (err) =>
          setError(err instanceof ApiError ? err.message : 'Could not create the network.'),
      },
    )
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title="Create a network"
      submitLabel="Create network"
      onSubmit={submit}
      pending={mutation.isPending}
      error={error}
      canSubmit={name.trim() !== ''}
    >
      <Field label="Name" required>
        <TextField
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="app_net"
          autoFocus
        />
      </Field>
      <Field label="Driver" hint="Defaults to bridge.">
        <TextField
          value={driver}
          onChange={(event) => setDriver(event.target.value)}
          placeholder="bridge"
        />
      </Field>
      <Field label="Subnet" hint="Leave empty for Docker to assign one.">
        <TextField
          value={subnet}
          onChange={(event) => setSubnet(event.target.value)}
          placeholder="172.20.0.0/16"
        />
      </Field>
    </FormDialog>
  )
}
