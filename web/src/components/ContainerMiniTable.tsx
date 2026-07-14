import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { type Container } from '../lib/containers'
import { timeAgo } from '../lib/format'
import { ContainerBulkBar } from './ContainerBulkBar'
import { ContainerStateBadge } from './ContainerStateBadge'
import { DataTable } from './DataTable'
import { ImageTag } from './ImageTag'
import { NameLink } from './NameLink'
import { PublishedPorts } from './PublishedPorts'
import styles from './ContainerMiniTable.module.css'

export function ContainerMiniTable({
  envId,
  containers,
  emptyMessage,
}: {
  envId: number
  containers: Container[]
  emptyMessage: string
}) {
  const columns = useMemo<ColumnDef<Container>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: (cell) => (
          <NameLink
            to="/environments/$id/containers/$containerId"
            params={{ id: String(envId), containerId: cell.row.original.id }}
          >
            {cell.row.original.name}
          </NameLink>
        ),
      },
      {
        accessorKey: 'state',
        header: 'State',
        cell: (cell) => <ContainerStateBadge state={cell.row.original.state} />,
      },
      {
        accessorKey: 'image',
        header: 'Image',
        cell: (cell) => <ImageTag image={cell.row.original.image} />,
      },
      {
        id: 'ports',
        header: 'Published ports',
        enableSorting: false,
        cell: (cell) => <PublishedPorts ports={cell.row.original.ports} />,
      },
      {
        accessorKey: 'ip',
        header: 'IP address',
        cell: (cell) => <span className={styles.muted}>{cell.row.original.ip || '-'}</span>,
      },
      {
        accessorKey: 'created',
        header: 'Created',
        cell: (cell) => <span className={styles.muted}>{timeAgo(cell.row.original.created)}</span>,
      },
    ],
    [envId],
  )

  return (
    <DataTable
      data={containers}
      columns={columns}
      searchPlaceholder="Search containers…"
      emptyMessage={emptyMessage}
      initialPageSize={25}
      enableSelection
      getRowId={(container) => container.id}
      renderBulkActions={(selected, clear) => (
        <ContainerBulkBar envId={envId} selected={selected} clear={clear} />
      )}
    />
  )
}
