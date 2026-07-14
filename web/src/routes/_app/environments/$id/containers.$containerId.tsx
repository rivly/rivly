import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { LuScrollText, LuTerminal } from 'react-icons/lu'
import { BackLink } from '../../../../components/BackLink'
import { Button } from '../../../../components/Button'
import { ContainerActionButtons } from '../../../../components/ContainerActionButtons'
import { ContainerStateBadge } from '../../../../components/ContainerStateBadge'
import { DetailHeader } from '../../../../components/DetailHeader'
import { Loader } from '../../../../components/Loader'
import { LogsViewer } from '../../../../components/LogsViewer'
import { TerminalViewer } from '../../../../components/TerminalViewer'
import { useContainerDetail, type ContainerPort } from '../../../../lib/containers'
import { useContainerStats } from '../../../../lib/stats'
import { formatBytes, formatDateTime } from '../../../../lib/format'
import styles from './containerDetail.module.css'

export const Route = createFileRoute('/_app/environments/$id/containers/$containerId')({
  head: () => ({ meta: [{ title: 'Container · Rivly' }] }),
  component: ContainerDetailPage,
})

function ContainerDetailPage() {
  const { id, containerId } = Route.useParams()
  const envId = Number(id)
  const navigate = useNavigate()
  const { data, isPending, isError } = useContainerDetail(envId, containerId)
  const [logsOpen, setLogsOpen] = useState(false)
  const [execOpen, setExecOpen] = useState(false)

  const backTo = {
    to: '/environments/$id/containers' as const,
    params: { id },
  }

  if (isPending) {
    return <Loader />
  }
  if (isError || !data) {
    return (
      <div>
        <BackLink {...backTo}>Containers</BackLink>
        <p className={styles.message}>Could not load this container.</p>
      </div>
    )
  }

  const running = data.state === 'running'
  const ref = { id: data.id, name: data.name }

  return (
    <div className={styles.page}>
      <div>
        <BackLink {...backTo}>Containers</BackLink>
      </div>

      <DetailHeader
        title={data.name}
        badges={<ContainerStateBadge state={data.state} />}
        actions={
          <>
            <Button variant="secondary" size="sm" icon={<LuScrollText />} onClick={() => setLogsOpen(true)}>
              Logs
            </Button>
            {running && (
              <Button variant="secondary" size="sm" icon={<LuTerminal />} onClick={() => setExecOpen(true)}>
                Terminal
              </Button>
            )}
            <ContainerActionButtons
              envId={envId}
              items={[{ id: data.id, state: data.state }]}
              onDone={(action) => {
                if (action === 'remove') {
                  navigate(backTo)
                }
              }}
            />
          </>
        }
      />

      <code className={styles.image}>{data.image}</code>

      {running && <StatsRow envId={envId} containerId={containerId} />}

      <div className={styles.sections}>
        <Section title="Labels">
          {Object.keys(data.labels).length === 0 ? (
            <Empty>No labels</Empty>
          ) : (
            <KeyValues rows={Object.entries(data.labels)} mono />
          )}
        </Section>

        <div className={styles.sectionsMain}>
        <Section title="Configuration">
          <KeyValues
            rows={[
              ['Command', data.command || '-'],
              ['Restart policy', data.restartPolicy || 'no'],
              ['Created', data.created ? formatDateTime(data.created) : '-'],
            ]}
          />
        </Section>

        <Section title="Ports">
          {data.ports.length === 0 ? (
            <Empty>No published ports</Empty>
          ) : (
            <div className={styles.tags}>
              {uniquePortLabels(data.ports).map((label) => (
                <span key={label} className={styles.portTag}>
                  {label}
                </span>
              ))}
            </div>
          )}
        </Section>

        <Section title="Networks">
          {data.networks.length === 0 ? (
            <Empty>Not attached to any network</Empty>
          ) : (
            <KeyValues rows={data.networks.map((n) => [n.name, n.ip || '-'])} mono />
          )}
        </Section>

        <Section title="Mounts">
          {data.mounts.length === 0 ? (
            <Empty>No mounts</Empty>
          ) : (
            <div className={styles.mounts}>
              {data.mounts.map((m, i) => (
                <div key={i} className={styles.mount}>
                  <code className={styles.mono}>{m.name || m.source}</code>
                  <span className={styles.arrow}>→</span>
                  <code className={styles.mono}>{m.destination}</code>
                  <span className={styles.mountMeta}>
                    {m.type} · {m.rw ? 'rw' : 'ro'}
                  </span>
                </div>
              ))}
            </div>
          )}
        </Section>

        <Section title="Environment">
          {data.env.length === 0 ? (
            <Empty>No environment variables</Empty>
          ) : (
            <div className={styles.envList}>
              {data.env.map((line, i) => (
                <code key={i} className={styles.envLine}>
                  {line}
                </code>
              ))}
            </div>
          )}
        </Section>
        </div>

      </div>

      <LogsViewer envId={envId} container={logsOpen ? ref : null} onClose={() => setLogsOpen(false)} />
      <TerminalViewer envId={envId} container={execOpen ? ref : null} onClose={() => setExecOpen(false)} />
    </div>
  )
}

function StatsRow({ envId, containerId }: { envId: number; containerId: string }) {
  const { stats } = useContainerStats(envId, containerId)

  return (
    <div className={styles.stats}>
      <StatTile label="CPU" value={stats ? `${stats.cpuPercent.toFixed(1)}%` : '-'} percent={stats?.cpuPercent} />
      <StatTile
        label="Memory"
        value={stats ? `${formatBytes(stats.memUsage)} / ${formatBytes(stats.memLimit)}` : '-'}
        percent={stats?.memPercent}
      />
      <StatTile
        label="Network I/O"
        value={stats ? `↓ ${formatBytes(stats.netRx)}   ↑ ${formatBytes(stats.netTx)}` : '-'}
      />
      <StatTile
        label="Block I/O"
        value={stats ? `R ${formatBytes(stats.blockRead)}   W ${formatBytes(stats.blockWrite)}` : '-'}
      />
      <StatTile label="PIDs" value={stats ? String(stats.pids) : '-'} />
    </div>
  )
}

function StatTile({ label, value, percent }: { label: string; value: string; percent?: number }) {
  return (
    <div className={styles.tile}>
      <span className={styles.tileLabel}>{label}</span>
      <span className={styles.tileValue}>{value}</span>
      {percent !== undefined && (
        <span className={styles.tileBar}>
          <span className={styles.tileBarFill} style={{ width: `${Math.min(percent, 100)}%` }} />
        </span>
      )}
    </div>
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>{title}</h2>
      {children}
    </section>
  )
}

function KeyValues({ rows, mono }: { rows: [string, string][]; mono?: boolean }) {
  return (
    <dl className={styles.kv}>
      {rows.map(([k, v], i) => (
        <div key={i} className={styles.kvRow}>
          <dt className={styles.kvKey}>{k}</dt>
          <dd className={`${styles.kvValue} ${mono ? styles.mono : ''}`}>{v}</dd>
        </div>
      ))}
    </dl>
  )
}

function Empty({ children }: { children: ReactNode }) {
  return <p className={styles.empty}>{children}</p>
}

function uniquePortLabels(ports: ContainerPort[]): string[] {
  const seen = new Set<string>()
  const labels: string[] = []
  for (const p of ports) {
    let label: string
    if (!p.publicPort) {
      label = `${p.privatePort}/${p.type}`
    } else {
      const ip = p.ip && p.ip !== '0.0.0.0' && p.ip !== '::' ? `${p.ip}:` : ''
      label = `${ip}${p.publicPort}:${p.privatePort}/${p.type}`
    }
    if (!seen.has(label)) {
      seen.add(label)
      labels.push(label)
    }
  }
  return labels
}
