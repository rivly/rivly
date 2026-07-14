import { createFileRoute, Link } from '@tanstack/react-router'
import { useCallback, useMemo, useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { LuPlus, LuScrollText, LuTerminal } from 'react-icons/lu'
import { Button } from '../../../../components/Button'
import { ContainerBulkBar } from '../../../../components/ContainerBulkBar'
import { DataTable } from '../../../../components/DataTable'
import { Loader } from '../../../../components/Loader'
import { LogsViewer } from '../../../../components/LogsViewer'
import { TerminalViewer } from '../../../../components/TerminalViewer'
import { Tooltip } from '../../../../components/Tooltip'
import {
  useContainers,
  type Container,
  type ContainerPort,
} from '../../../../lib/containers'
import { timeAgo } from '../../../../lib/format'
import styles from './containers.module.css'

export const Route = createFileRoute('/_app/environments/$id/containers/')({
  head: () => ({ meta: [{ title: 'Containers · Rivly' }] }),
  validateSearch: (search: Record<string, unknown>): { stack?: string } => ({
    stack: typeof search.stack === 'string' ? search.stack : undefined,
  }),
  component: ContainersPage,
})

const STATE_TONE: Record<string, string> = {
  running: styles.running,
  paused: styles.paused,
  restarting: styles.paused,
  created: styles.info,
  exited: styles.danger,
  removing: styles.neutral,
  dead: styles.danger,
}

function ContainersPage() {
  const { id } = Route.useParams()
  const { stack } = Route.useSearch()
  const { data: containers, isPending, isError } = useContainers(Number(id))
  const [logsFor, setLogsFor] = useState<Container | null>(null)
  const [execFor, setExecFor] = useState<Container | null>(null)

  const openLogs = useCallback((container: Container) => setLogsFor(container), [])
  const openExec = useCallback((container: Container) => setExecFor(container), [])

  const columns = useMemo<ColumnDef<Container>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        size: 200,
        meta: { sticky: 'left' },
        cell: (cell) => (
          <Link
            to="/environments/$id/containers/$containerId"
            params={{ id, containerId: cell.row.original.id }}
            className={styles.name}
          >
            {cell.row.original.name}
          </Link>
        ),
      },
      {
        id: 'actions',
        header: 'Actions',
        size: 104,
        enableSorting: false,
        enableHiding: false,
        meta: { sticky: 'left' },
        cell: (cell) => (
          <RowActions container={cell.row.original} onLogs={openLogs} onExec={openExec} />
        ),
      },
      {
        accessorKey: 'stack',
        header: 'Stack',
        cell: (cell) =>
          cell.row.original.stack || <span className={styles.muted}>-</span>,
      },
      {
        accessorKey: 'state',
        header: 'State',
        cell: (cell) => <StateCell state={cell.row.original.state} />,
      },
      {
        accessorKey: 'image',
        header: 'Image',
        cell: (cell) => <code className={styles.image}>{cell.row.original.image}</code>,
      },
      {
        id: 'ports',
        header: 'Published ports',
        enableSorting: false,
        cell: (cell) => <PortsCell ports={cell.row.original.ports} />,
      },
      {
        accessorKey: 'ip',
        header: 'IP address',
        cell: (cell) => (
          <span className={styles.muted}>{cell.row.original.ip || '-'}</span>
        ),
      },
      {
        accessorKey: 'created',
        header: 'Created',
        cell: (cell) => (
          <span className={styles.muted}>{timeAgo(cell.row.original.created)}</span>
        ),
      },
    ],
    [openLogs, openExec, id],
  )

  return (
    <div>
      <header className={styles.head}>
        <h1 className={styles.title}>Containers</h1>
        <Button
          size="sm"
          icon={<LuPlus />}
          render={<Link to="/environments/$id/containers/new" params={{ id }} />}
        >
          Run container
        </Button>
      </header>

      {isPending && <Loader />}
      {isError && (
        <p className={styles.message}>Could not load containers.</p>
      )}
      {containers && (
        <DataTable
          data={containers}
          columns={columns}
          searchPlaceholder="Search containers…"
          emptyMessage="No containers on this host."
          initialPageSize={25}
          initialGlobalFilter={stack}
          enableSelection
          getRowId={(container) => container.id}
          renderBulkActions={(selected, clear) => (
            <ContainerBulkBar
              envId={Number(id)}
              selected={selected}
              clear={clear}
            />
          )}
        />
      )}

      <LogsViewer envId={Number(id)} container={logsFor} onClose={() => setLogsFor(null)} />
      <TerminalViewer envId={Number(id)} container={execFor} onClose={() => setExecFor(null)} />
    </div>
  )
}

function RowActions({
  container,
  onLogs,
  onExec,
}: {
  container: Container
  onLogs: (container: Container) => void
  onExec: (container: Container) => void
}) {
  return (
    <span className={styles.actions}>
      <Tooltip content="Logs">
        <Button
          variant="secondary"
          size="sm"
          iconOnly
          icon={<LuScrollText />}
          aria-label="Logs"
          onClick={() => onLogs(container)}
        />
      </Tooltip>
      <Tooltip content={container.state === 'running' ? 'Terminal' : 'Container is not running'}>
        <Button
          variant="secondary"
          size="sm"
          iconOnly
          icon={<LuTerminal />}
          aria-label="Terminal"
          disabled={container.state !== 'running'}
          focusableWhenDisabled
          onClick={() => onExec(container)}
        />
      </Tooltip>
    </span>
  )
}

function StateCell({ state }: { state: string }) {
  return (
    <span className={`${styles.stateBadge} ${STATE_TONE[state] ?? styles.neutral}`}>
      <span className={styles.stateDot} />
      {state}
    </span>
  )
}

function PortsCell({ ports }: { ports: Container['ports'] }) {
  if (ports.length === 0) {
    return <span className={styles.muted}>-</span>
  }

  const seen = new Set<string>()
  const unique = ports.filter((port) => {
    const key = `${port.publicPort}:${port.privatePort}/${port.type}`
    if (seen.has(key)) {
      return false
    }
    seen.add(key)
    return true
  })

  return (
    <span className={styles.ports}>
      {unique.map((port) => (
        <Tooltip
          key={`${port.publicPort}:${port.privatePort}/${port.type}`}
          content={portDetail(port)}
        >
          <span className={styles.portTag}>{portLabel(port)}</span>
        </Tooltip>
      ))}
    </span>
  )
}

function portLabel(port: ContainerPort): string {
  return port.publicPort
    ? `${port.publicPort}:${port.privatePort}`
    : `${port.privatePort}`
}

function portDetail(port: ContainerPort): string {
  if (port.publicPort) {
    return `${port.ip || '0.0.0.0'}:${port.publicPort} → ${port.privatePort}/${port.type}`
  }
  return `${port.privatePort}/${port.type} · not published`
}
