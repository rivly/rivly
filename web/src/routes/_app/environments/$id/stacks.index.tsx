import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { LuPencil, LuPlus } from 'react-icons/lu'
import { Button } from '../../../../components/Button'
import { DataTable } from '../../../../components/DataTable'
import { LimitedBadge } from '../../../../components/LimitedBadge'
import { NameLink } from '../../../../components/NameLink'
import { PageHeader } from '../../../../components/PageHeader'
import { QueryState } from '../../../../components/QueryState'
import { StackBulkBar } from '../../../../components/StackBulkBar'
import { StackStateBadge } from '../../../../components/StackStateBadge'
import { useStacks, type Stack } from '../../../../lib/stacks'
import { formatDateTime } from '../../../../lib/format'
import styles from './stacks.module.css'

export const Route = createFileRoute('/_app/environments/$id/stacks/')({
  head: () => ({ meta: [{ title: 'Stacks · Rivly' }] }),
  component: StacksPage,
})

function StacksPage() {
  const { id } = Route.useParams()
  const navigate = useNavigate()
  const { data: stacks, isPending, isError } = useStacks(Number(id))

  const columns = useMemo<ColumnDef<Stack>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: (cell) => (
          <span className={styles.nameCell}>
            <NameLink
              to="/environments/$id/stacks/$name"
              params={{ id, name: cell.row.original.name }}
            >
              {cell.row.original.name}
            </NameLink>
            {cell.row.original.type === 'external' && <LimitedBadge />}
          </span>
        ),
      },
      {
        accessorKey: 'state',
        header: 'State',
        cell: (cell) => <StackStateBadge state={cell.row.original.state} />,
      },
      {
        accessorKey: 'services',
        header: 'Services',
        cell: (cell) => (
          <span className={styles.muted}>
            {cell.row.original.running}/{cell.row.original.total} running
          </span>
        ),
      },
      {
        accessorKey: 'createdAt',
        header: 'Created',
        cell: (cell) => (
          <WhenCell at={cell.row.original.createdAt} by={cell.row.original.createdBy} />
        ),
      },
      {
        accessorKey: 'updatedAt',
        header: 'Updated',
        cell: (cell) => (
          <WhenCell at={cell.row.original.updatedAt} by={cell.row.original.updatedBy} />
        ),
      },
      {
        id: 'actions',
        header: 'Actions',
        enableSorting: false,
        enableHiding: false,
        cell: (cell) =>
          cell.row.original.type === 'rivly' ? (
            <Button
              variant="secondary"
              size="sm"
              icon={<LuPencil />}
              onClick={() =>
                navigate({
                  to: '/environments/$id/stacks/$name/edit',
                  params: { id, name: cell.row.original.name },
                })
              }
            >
              Edit stack
            </Button>
          ) : null,
      },
    ],
    [id, navigate],
  )

  return (
    <div>
      <PageHeader
        title="Stacks"
        action={
          <Button
            size="sm"
            icon={<LuPlus />}
            render={<Link to="/environments/$id/stacks/new" params={{ id }} />}
          >
            Deploy stack
          </Button>
        }
      />

      <QueryState pending={isPending} error={isError} errorMessage="Could not load stacks.">
        {stacks && (
          <DataTable
            data={stacks}
            columns={columns}
            searchPlaceholder="Search stacks…"
            emptyMessage="No stacks on this host."
            initialPageSize={25}
            enableSelection
            getRowId={(stack) => stack.name}
            renderBulkActions={(selected, clear) => (
              <StackBulkBar envId={Number(id)} selected={selected} clear={clear} />
            )}
          />
        )}
      </QueryState>
    </div>
  )
}

function WhenCell({ at, by }: { at: number; by: string }) {
  return (
    <span className={styles.when}>
      {at ? (
        <>
          <span className={styles.whenDate}>{formatDateTime(at)}</span>
          <span className={styles.whenBy}>{by || ' '}</span>
        </>
      ) : (
        <span className={styles.muted}>-</span>
      )}
    </span>
  )
}
