import { createFileRoute } from '@tanstack/react-router'
import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { DataTable } from '../../../../components/DataTable'
import { Loader } from '../../../../components/Loader'
import { NetworkBulkBar } from '../../../../components/NetworkBulkBar'
import { useNetworks, type Network } from '../../../../lib/networks'
import { timeAgo } from '../../../../lib/format'
import styles from './networks.module.css'

export const Route = createFileRoute('/_app/environments/$id/networks')({
  head: () => ({ meta: [{ title: 'Networks · Rivly' }] }),
  component: NetworksPage,
})

function NetworksPage() {
  const { id } = Route.useParams()
  const { data: networks, isPending, isError } = useNetworks(Number(id))

  const columns = useMemo<ColumnDef<Network>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: (cell) => <NameCell network={cell.row.original} />,
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
        accessorKey: 'scope',
        header: 'Scope',
        cell: (cell) => <span className={styles.muted}>{cell.row.original.scope}</span>,
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
        <h1 className={styles.title}>Networks</h1>
      </header>

      {isPending && <Loader />}
      {isError && <p className={styles.message}>Could not load networks.</p>}
      {networks && (
        <DataTable
          data={networks}
          columns={columns}
          searchPlaceholder="Search networks…"
          emptyMessage="No networks on this host."
          initialPageSize={25}
          enableSelection
          getRowId={(network) => network.id}
          renderBulkActions={(selected, clear) => (
            <NetworkBulkBar envId={Number(id)} selected={selected} clear={clear} />
          )}
        />
      )}
    </div>
  )
}

function NameCell({ network }: { network: Network }) {
  return (
    <span className={styles.nameCell}>
      <span className={styles.name}>{network.name}</span>
      {!network.inUse && <span className={styles.unusedBadge}>Unused</span>}
    </span>
  )
}
