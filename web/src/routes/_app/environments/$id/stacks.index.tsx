import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { LuInfo, LuPencil, LuPlus } from 'react-icons/lu'
import { Button } from '../../../../components/Button'
import { DataTable } from '../../../../components/DataTable'
import { Loader } from '../../../../components/Loader'
import { StackBulkBar } from '../../../../components/StackBulkBar'
import { Tooltip } from '../../../../components/Tooltip'
import { useStacks, type Stack } from '../../../../lib/stacks'
import { formatDateTime } from '../../../../lib/format'
import styles from './stacks.module.css'

export const Route = createFileRoute('/_app/environments/$id/stacks/')({
  head: () => ({ meta: [{ title: 'Stacks · Rivly' }] }),
  component: StacksPage,
})

const STATE_TONE: Record<Stack['state'], string> = {
  running: styles.running,
  partial: styles.partial,
  stopped: styles.stopped,
}

const STATE_LABEL: Record<Stack['state'], string> = {
  running: 'Running',
  partial: 'Partial',
  stopped: 'Stopped',
}

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
            <Link
              to="/environments/$id/containers"
              params={{ id }}
              search={{ stack: cell.row.original.name }}
              className={styles.name}
            >
              {cell.row.original.name}
            </Link>
            {cell.row.original.type === 'external' && (
              <Tooltip content="This stack was created outside Rivly, so control over it is limited.">
                <span className={styles.badge}>
                  <LuInfo />
                  Limited
                </span>
              </Tooltip>
            )}
          </span>
        ),
      },
      {
        accessorKey: 'state',
        header: 'State',
        cell: (cell) => (
          <span className={`${styles.state} ${STATE_TONE[cell.row.original.state]}`}>
            <span className={styles.stateDot} />
            {STATE_LABEL[cell.row.original.state]}
          </span>
        ),
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
      <header className={styles.head}>
        <h1 className={styles.title}>Stacks</h1>
        <Button
          size="sm"
          icon={<LuPlus />}
          render={<Link to="/environments/$id/stacks/new" params={{ id }} />}
        >
          Deploy stack
        </Button>
      </header>

      {isPending && <Loader />}
      {isError && <p className={styles.message}>Could not load stacks.</p>}
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
