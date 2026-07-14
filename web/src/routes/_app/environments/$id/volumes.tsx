import { createFileRoute } from '@tanstack/react-router'
import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { DataTable } from '../../../../components/DataTable'
import { Loader } from '../../../../components/Loader'
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

  const columns = useMemo<ColumnDef<Volume>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: (cell) => <NameCell volume={cell.row.original} />,
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
      <header className={styles.head}>
        <h1 className={styles.title}>Volumes</h1>
      </header>

      {isPending && <Loader />}
      {isError && <p className={styles.message}>Could not load volumes.</p>}
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
    </div>
  )
}

function NameCell({ volume }: { volume: Volume }) {
  return (
    <span className={styles.nameCell}>
      <span className={styles.name}>{volume.name}</span>
      {!volume.inUse && <span className={styles.unusedBadge}>Unused</span>}
    </span>
  )
}
