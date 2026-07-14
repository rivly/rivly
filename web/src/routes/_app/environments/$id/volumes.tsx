import { createFileRoute } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { LuPlus } from 'react-icons/lu'
import { Button } from '../../../../components/Button'
import { CreateVolumeDialog } from '../../../../components/CreateVolumeDialog'
import { DataTable } from '../../../../components/DataTable'
import { NameCell } from '../../../../components/NameCell'
import { PageHeader } from '../../../../components/PageHeader'
import { QueryState } from '../../../../components/QueryState'
import { Tooltip } from '../../../../components/Tooltip'
import { VolumeBulkBar } from '../../../../components/VolumeBulkBar'
import { useVolumes, type Volume } from '../../../../lib/volumes'
import { timeAgo } from '../../../../lib/format'
import styles from './volumes.module.css'

export const Route = createFileRoute('/_app/environments/$id/volumes')({
  head: () => ({ meta: [{ title: 'Volumes · Rivly' }] }),
  component: VolumesPage,
})

function VolumesPage() {
  const { id } = Route.useParams()
  const { data: volumes, isPending, isError } = useVolumes(Number(id))
  const [createOpen, setCreateOpen] = useState(false)

  const columns = useMemo<ColumnDef<Volume>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: (cell) => (
          <NameCell inUse={cell.row.original.inUse}>{cell.row.original.name}</NameCell>
        ),
      },
      {
        accessorKey: 'stack',
        header: 'Stack',
        cell: (cell) =>
          cell.row.original.stack || <span className={styles.muted}>-</span>,
      },
      {
        accessorKey: 'driver',
        header: 'Driver',
        cell: (cell) => <span className={styles.muted}>{cell.row.original.driver}</span>,
      },
      {
        accessorKey: 'mountpoint',
        header: 'Mount point',
        cell: (cell) => (
          <Tooltip content={cell.row.original.mountpoint}>
            <code className={styles.mount}>{cell.row.original.mountpoint}</code>
          </Tooltip>
        ),
      },
      {
        accessorKey: 'created',
        header: 'Created',
        cell: (cell) =>
          cell.row.original.created ? (
            <span className={styles.muted}>{timeAgo(cell.row.original.created)}</span>
          ) : (
            <span className={styles.muted}>-</span>
          ),
      },
    ],
    [],
  )

  return (
    <div>
      <PageHeader
        title="Volumes"
        action={
          <Button size="sm" icon={<LuPlus />} onClick={() => setCreateOpen(true)}>
            Create volume
          </Button>
        }
      />

      <QueryState pending={isPending} error={isError} errorMessage="Could not load volumes.">
        {volumes && (
          <DataTable
            data={volumes}
            columns={columns}
            searchPlaceholder="Search volumes…"
            emptyMessage="No volumes on this host."
            initialPageSize={25}
            enableSelection
            getRowId={(volume) => volume.name}
            renderBulkActions={(selected, clear) => (
              <VolumeBulkBar envId={Number(id)} selected={selected} clear={clear} />
            )}
          />
        )}
      </QueryState>

      <CreateVolumeDialog
        envId={Number(id)}
        open={createOpen}
        onClose={() => setCreateOpen(false)}
      />
    </div>
  )
}
