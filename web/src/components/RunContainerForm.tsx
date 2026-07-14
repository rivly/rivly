import { useMemo, useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { LuPlus, LuX } from 'react-icons/lu'
import { ApiError } from '../lib/api'
import { useCreateContainer } from '../lib/containers'
import { useNetworks } from '../lib/networks'
import { toast } from '../lib/toast'
import { BackLink } from './BackLink'
import { Button } from './Button'
import { Checkbox } from './Checkbox'
import { RequiredMark } from './RequiredMark'
import { Select } from './Select'
import styles from './RunContainerForm.module.css'

type PortRow = { hostPort: string; containerPort: string; proto: string }
type EnvRow = { key: string; value: string }
type MountRow = { source: string; target: string; readOnly: boolean }

const RESTART_ITEMS = [
  { value: 'no', label: "Don't restart" },
  { value: 'unless-stopped', label: 'Unless stopped' },
  { value: 'always', label: 'Always' },
  { value: 'on-failure', label: 'On failure' },
]

const PROTO_ITEMS = [
  { value: 'tcp', label: 'TCP' },
  { value: 'udp', label: 'UDP' },
]

export function RunContainerForm({ envId }: { envId: number }) {
  const navigate = useNavigate()
  const mutation = useCreateContainer(envId)
  const { data: networks } = useNetworks(envId)

  const [name, setName] = useState('')
  const [image, setImage] = useState('')
  const [command, setCommand] = useState('')
  const [ports, setPorts] = useState<PortRow[]>([])
  const [env, setEnv] = useState<EnvRow[]>([])
  const [mounts, setMounts] = useState<MountRow[]>([])
  const [network, setNetwork] = useState('bridge')
  const [restartPolicy, setRestartPolicy] = useState('no')
  const [start, setStart] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const backTo = {
    to: '/environments/$id/containers' as const,
    params: { id: String(envId) },
  }

  const networkItems = useMemo(() => {
    const names = new Set(['bridge', 'host', 'none'])
    for (const n of networks ?? []) {
      names.add(n.name)
    }
    return [...names].map((n) => ({ value: n, label: n }))
  }, [networks])

  const submit = () => {
    const trimmedImage = image.trim()
    if (trimmedImage === '') {
      setError('Image is required.')
      return
    }
    setError(null)
    mutation.mutate(
      {
        name: name.trim() || undefined,
        image: trimmedImage,
        command: command.trim() || undefined,
        env: env.filter((e) => e.key.trim() !== ''),
        ports: ports.filter((p) => p.containerPort.trim() !== ''),
        mounts: mounts.filter((m) => m.source.trim() !== '' && m.target.trim() !== ''),
        network,
        restartPolicy,
        start,
      },
      {
        onSuccess: () => {
          const label = name.trim() || trimmedImage
          toast.success(start ? `Started ${label}` : `Created ${label}`)
          navigate(backTo)
        },
        onError: (err) =>
          setError(err instanceof ApiError ? err.message : 'Could not run the container.'),
      },
    )
  }

  return (
    <div className={styles.page}>
      <div>
        <BackLink {...backTo}>Containers</BackLink>
      </div>
      <h1 className={styles.title}>Run a container</h1>

      <form
        className={styles.form}
        onSubmit={(event) => {
          event.preventDefault()
          submit()
        }}
      >
        <section className={styles.section}>
          <label className={styles.field}>
            <span className={styles.label}>
              Image
              <RequiredMark />
            </span>
            <input
              className={styles.input}
              value={image}
              onChange={(e) => setImage(e.target.value)}
              placeholder="nginx:latest"
              autoComplete="off"
              spellCheck={false}
              autoFocus
            />
          </label>
          <label className={styles.field}>
            <span className={styles.label}>Name</span>
            <input
              className={styles.input}
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Leave blank to auto-generate"
              autoComplete="off"
              spellCheck={false}
            />
          </label>
          <label className={styles.field}>
            <span className={styles.label}>Command</span>
            <input
              className={styles.input}
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              placeholder="Overrides the image default"
              autoComplete="off"
              spellCheck={false}
            />
          </label>
        </section>

        <section className={styles.section}>
          <span className={styles.sectionTitle}>Ports</span>
          {ports.length > 0 && (
            <div className={styles.rows}>
              {ports.map((row, index) => (
                <div key={index} className={styles.portRow}>
                  <input
                    className={styles.input}
                    value={row.hostPort}
                    onChange={(e) =>
                      setPorts((p) =>
                        p.map((r, i) => (i === index ? { ...r, hostPort: e.target.value } : r)),
                      )
                    }
                    placeholder="Host"
                    autoComplete="off"
                  />
                  <span className={styles.colon}>:</span>
                  <input
                    className={styles.input}
                    value={row.containerPort}
                    onChange={(e) =>
                      setPorts((p) =>
                        p.map((r, i) => (i === index ? { ...r, containerPort: e.target.value } : r)),
                      )
                    }
                    placeholder="Container"
                    autoComplete="off"
                  />
                  <div className={styles.proto}>
                    <Select
                      items={PROTO_ITEMS}
                      value={row.proto}
                      onValueChange={(v) =>
                        setPorts((p) =>
                          p.map((r, i) => (i === index ? { ...r, proto: v ?? 'tcp' } : r)),
                        )
                      }
                      aria-label="Protocol"
                    />
                  </div>
                  <button
                    type="button"
                    className={styles.remove}
                    onClick={() => setPorts((p) => p.filter((_, i) => i !== index))}
                    aria-label="Remove port"
                  >
                    <LuX />
                  </button>
                </div>
              ))}
            </div>
          )}
          <Button
            type="button"
            variant="secondary"
            size="sm"
            icon={<LuPlus />}
            onClick={() => setPorts((p) => [...p, { hostPort: '', containerPort: '', proto: 'tcp' }])}
          >
            Add port
          </Button>
        </section>

        <section className={styles.section}>
          <span className={styles.sectionTitle}>Environment variables</span>
          {env.length > 0 && (
            <div className={styles.rows}>
              {env.map((row, index) => (
                <div key={index} className={styles.kvRow}>
                  <input
                    className={styles.input}
                    value={row.key}
                    onChange={(e) =>
                      setEnv((v) => v.map((r, i) => (i === index ? { ...r, key: e.target.value } : r)))
                    }
                    placeholder="KEY"
                    autoComplete="off"
                    spellCheck={false}
                  />
                  <input
                    className={styles.input}
                    value={row.value}
                    onChange={(e) =>
                      setEnv((v) => v.map((r, i) => (i === index ? { ...r, value: e.target.value } : r)))
                    }
                    placeholder="value"
                    autoComplete="off"
                    spellCheck={false}
                  />
                  <button
                    type="button"
                    className={styles.remove}
                    onClick={() => setEnv((v) => v.filter((_, i) => i !== index))}
                    aria-label="Remove variable"
                  >
                    <LuX />
                  </button>
                </div>
              ))}
            </div>
          )}
          <Button
            type="button"
            variant="secondary"
            size="sm"
            icon={<LuPlus />}
            onClick={() => setEnv((v) => [...v, { key: '', value: '' }])}
          >
            Add variable
          </Button>
        </section>

        <section className={styles.section}>
          <span className={styles.sectionTitle}>Volumes</span>
          {mounts.length > 0 && (
            <div className={styles.rows}>
              {mounts.map((row, index) => (
                <div key={index} className={styles.mountRow}>
                  <input
                    className={styles.input}
                    value={row.source}
                    onChange={(e) =>
                      setMounts((m) =>
                        m.map((r, i) => (i === index ? { ...r, source: e.target.value } : r)),
                      )
                    }
                    placeholder="Volume name or host path"
                    autoComplete="off"
                    spellCheck={false}
                  />
                  <span className={styles.arrow}>→</span>
                  <input
                    className={styles.input}
                    value={row.target}
                    onChange={(e) =>
                      setMounts((m) =>
                        m.map((r, i) => (i === index ? { ...r, target: e.target.value } : r)),
                      )
                    }
                    placeholder="/container/path"
                    autoComplete="off"
                    spellCheck={false}
                  />
                  <Checkbox
                    label="RO"
                    checked={row.readOnly}
                    onCheckedChange={(v) =>
                      setMounts((m) =>
                        m.map((r, i) => (i === index ? { ...r, readOnly: v === true } : r)),
                      )
                    }
                  />
                  <button
                    type="button"
                    className={styles.remove}
                    onClick={() => setMounts((m) => m.filter((_, i) => i !== index))}
                    aria-label="Remove volume"
                  >
                    <LuX />
                  </button>
                </div>
              ))}
            </div>
          )}
          <Button
            type="button"
            variant="secondary"
            size="sm"
            icon={<LuPlus />}
            onClick={() => setMounts((m) => [...m, { source: '', target: '', readOnly: false }])}
          >
            Add volume
          </Button>
        </section>

        <section className={styles.section}>
          <div className={styles.grid2}>
            <div className={styles.field}>
              <span className={styles.label}>Network</span>
              <Select
                items={networkItems}
                value={network}
                onValueChange={(v) => setNetwork(v ?? 'bridge')}
                aria-label="Network"
              />
            </div>
            <div className={styles.field}>
              <span className={styles.label}>Restart policy</span>
              <Select
                items={RESTART_ITEMS}
                value={restartPolicy}
                onValueChange={(v) => setRestartPolicy(v ?? 'no')}
                aria-label="Restart policy"
              />
            </div>
          </div>
          <Checkbox
            label="Start the container after creating it"
            checked={start}
            onCheckedChange={(v) => setStart(v === true)}
          />
        </section>

        {error && <p className={styles.error}>{error}</p>}

        <div className={styles.footer}>
          <Button type="button" variant="secondary" render={<Link {...backTo} />}>
            Cancel
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            Run container
          </Button>
        </div>
      </form>
    </div>
  )
}
