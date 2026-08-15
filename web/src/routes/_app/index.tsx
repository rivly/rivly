import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useCallback, useMemo, useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { LuPencil, LuPlus, LuTrash2 } from 'react-icons/lu'
import { Button } from '../../components/Button'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { DataTable } from '../../components/DataTable'
import { EnvironmentDialog } from '../../components/EnvironmentDialog'
import { PageHeader } from '../../components/PageHeader'
import { QueryState } from '../../components/QueryState'
import { StatusBadge } from '../../components/StatusBadge'
import {
  useDeleteEnvironment,
  useEnvironments,
  type EnvironmentDetail,
} from '../../lib/environments'
import { formatBytes, timeAgo } from '../../lib/format'
import { toast } from '../../lib/toast'
import styles from './index.module.css'

export const Route = createFileRoute('/_app/')({
  head: () => ({ meta: [{ title: 'Environments · Rivly' }] }),
  component: EnvironmentsPage,
})

function EnvironmentsPage() {
  const navigate = useNavigate()
  const { data: environments, isPending, isError } = useEnvironments()
  const deletion = useDeleteEnvironment()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<EnvironmentDetail | null>(null)

  const openAdd = () => {
    setEditing(null)
    setDialogOpen(true)
  }
  const openEdit = useCallback((env: EnvironmentDetail) => {
    setEditing(env)
    setDialogOpen(true)
  }, [])

  const remove = useCallback(
    (env: EnvironmentDetail) => {
      deletion.mutate(env.id, {
        onSuccess: () => toast.success(`Removed ${env.name}`),
        onError: () => toast.error('Could not remove', 'Please try again'),
      })
    },
    [deletion],
  )

  const columns = useMemo<ColumnDef<EnvironmentDetail>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: (cell) => cell.row.original.name,
      },
      {
        accessorKey: 'url',
        header: 'Endpoint',
        cell: (cell) => (
          <span className={styles.muted}>{cell.row.original.url}</span>
        ),
      },
      {
        accessorKey: 'status',
        header: 'Status',
        cell: (cell) => {
          const env = cell.row.original
          return (
            <span className={styles.status}>
              <StatusBadge status={env.status} />
              {env.status !== 'up' && env.lastSeen !== undefined && (
                <span className={styles.muted}>{timeAgo(env.lastSeen)}</span>
              )}
            </span>
          )
        },
      },
      {
        id: 'containers',
        header: 'Containers',
        accessorFn: (env) => env.system?.containersRunning ?? -1,
        cell: (cell) => {
          const system = cell.row.original.system
          if (!system) {
            return <span className={styles.muted}>-</span>
          }
          const total =
            system.containersRunning +
            system.containersPaused +
            system.containersStopped
          return (
            <span className={styles.count}>
              {system.containersRunning}
              <span className={styles.muted}>/{total}</span>
            </span>
          )
        },
      },
      {
        id: 'images',
        header: 'Images',
        accessorFn: (env) => env.system?.images ?? -1,
        cell: (cell) =>
          cell.row.original.system?.images ?? (
            <span className={styles.muted}>-</span>
          ),
      },
      {
        id: 'cpu',
        header: 'CPU',
        accessorFn: (env) => env.system?.ncpu ?? -1,
        cell: (cell) =>
          cell.row.original.system?.ncpu ?? (
            <span className={styles.muted}>-</span>
          ),
      },
      {
        id: 'memory',
        header: 'Memory',
        accessorFn: (env) => env.system?.memTotal ?? -1,
        cell: (cell) => {
          const system = cell.row.original.system
          return system ? (
            formatBytes(system.memTotal)
          ) : (
            <span className={styles.muted}>-</span>
          )
        },
      },
      {
        id: 'engine',
        header: 'Engine',
        accessorFn: (env) => env.system?.serverVersion ?? '',
        cell: (cell) => {
          const system = cell.row.original.system
          if (!system) {
            return <span className={styles.muted}>-</span>
          }
          return (
            <span className={styles.muted}>
              {system.serverVersion} {system.swarm ? 'Swarm' : 'Standalone'}
            </span>
          )
        },
      },
      {
        id: 'actions',
        header: 'Actions',
        enableSorting: false,
        enableHiding: false,
        cell: (cell) => (
          <div
            className={styles.actions}
            onClick={(event) => event.stopPropagation()}
          >
            <Button
              variant="secondary"
              size="sm"
              icon={<LuPencil />}
              onClick={() => openEdit(cell.row.original)}
            >
              Edit
            </Button>
            <ConfirmDialog
              trigger={
                <Button
                  variant="danger"
                  size="sm"
                  iconOnly
                  icon={<LuTrash2 />}
                  aria-label="Remove environment"
                />
              }
              title={`Remove ${cell.row.original.name}?`}
              description="Rivly forgets this endpoint and the stacks it deployed there. Nothing running on the host is touched."
              onConfirm={() => remove(cell.row.original)}
            />
          </div>
        ),
      },
    ],
    [openEdit, remove],
  )

  return (
    <div>
      <PageHeader
        title="Environments"
        subtitle="Every Docker endpoint Rivly can reach. Select one to manage its containers, images, and stacks."
        action={
          <Button size="sm" icon={<LuPlus />} onClick={openAdd}>
            Add environment
          </Button>
        }
      />

      <QueryState
        pending={isPending}
        error={isError}
        errorMessage="Could not load environments."
      >
        {environments && (
          <DataTable
            data={environments}
            columns={columns}
            searchPlaceholder="Search environments…"
            emptyMessage="No environments yet."
            initialPageSize={25}
            getRowId={(env) => String(env.id)}
            onRowClick={(env) => {
              if (env.status !== 'up') {
                toast.error(
                  'Environment unreachable',
                  `${env.name} is not responding.`,
                )
                return
              }
              navigate({
                to: '/environments/$id',
                params: { id: String(env.id) },
              })
            }}
          />
        )}
      </QueryState>

      <EnvironmentDialog
        open={dialogOpen}
        editing={editing}
        onClose={() => setDialogOpen(false)}
      />
    </div>
  )
}
