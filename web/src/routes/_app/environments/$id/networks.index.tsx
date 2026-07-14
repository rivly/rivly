import { createFileRoute } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { LuPlus } from 'react-icons/lu'
import { Button } from '../../../../components/Button'
import { CreateNetworkDialog } from '../../../../components/CreateNetworkDialog'
import { DataTable } from '../../../../components/DataTable'
import { NameLink } from '../../../../components/NameLink'
import { NetworkBulkBar } from '../../../../components/NetworkBulkBar'
import { PageHeader } from '../../../../components/PageHeader'
import { QueryState } from '../../../../components/QueryState'
import { UnusedBadge } from '../../../../components/UnusedBadge'
import { useNetworks, type Network } from '../../../../lib/networks'
import { timeAgo } from '../../../../lib/format'
import styles from './networks.module.css'

export const Route = createFileRoute('/_app/environments/$id/networks/')({
  head: () => ({ meta: [{ title: 'Networks · Rivly' }] }),
  component: NetworksPage,
})

function NetworksPage() {
  const { id } = Route.useParams()
  const { data: networks, isPending, isError } = useNetworks(Number(id))
  const [createOpen, setCreateOpen] = useState(false)

  const columns = useMemo<ColumnDef<Network>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: (cell) => (
          <span className={styles.nameCell}>
            <NameLink
              to="/environments/$id/networks/$networkId"
              params={{ id, networkId: cell.row.original.id }}
            >
              {cell.row.original.name}
            </NameLink>
            {!cell.row.original.inUse && <UnusedBadge />}
          </span>
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
    [id],
  )

  return (
    <div>
      <PageHeader
        title="Networks"
        action={
          <Button size="sm" icon={<LuPlus />} onClick={() => setCreateOpen(true)}>
            Create network
          </Button>
        }
      />

      <QueryState pending={isPending} error={isError} errorMessage="Could not load networks.">
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
      </QueryState>

      <CreateNetworkDialog
        envId={Number(id)}
        open={createOpen}
        onClose={() => setCreateOpen(false)}
      />
    </div>
  )
}
