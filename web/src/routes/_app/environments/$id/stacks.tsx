import { createFileRoute, Link } from '@tanstack/react-router'
import { useCallback, useMemo, useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { LuPencil, LuPlus } from 'react-icons/lu'
import { Button } from '../../../../components/Button'
import { DataTable } from '../../../../components/DataTable'
import { Loader } from '../../../../components/Loader'
import { StackBulkBar } from '../../../../components/StackBulkBar'
import { StackEditor, type EditorState } from '../../../../components/StackEditor'
import { Tooltip } from '../../../../components/Tooltip'
import { useStacks, type Stack } from '../../../../lib/stacks'
import styles from './stacks.module.css'

export const Route = createFileRoute('/_app/environments/$id/stacks')({
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
  const { data: stacks, isPending, isError } = useStacks(Number(id))
  const [editor, setEditor] = useState<EditorState | null>(null)

  const openEdit = useCallback((name: string) => setEditor({ mode: 'edit', name }), [])

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
            <span
              className={`${styles.badge} ${cell.row.original.type === 'rivly' ? styles.rivly : ''}`}
            >
              {cell.row.original.type === 'rivly' ? 'Rivly' : 'External'}
            </span>
          </span>
        ),
      },
      {
        id: 'actions',
        header: '',
        enableSorting: false,
        enableHiding: false,
        cell: (cell) =>
          cell.row.original.type === 'rivly' ? (
            <Tooltip content="Edit">
              <Button
                variant="secondary"
                size="sm"
                iconOnly
                icon={<LuPencil />}
                aria-label="Edit stack"
                onClick={() => openEdit(cell.row.original.name)}
              />
            </Tooltip>
          ) : null,
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
        accessorKey: 'workingDir',
        header: 'Path',
        cell: (cell) =>
          cell.row.original.workingDir ? (
            <code className={styles.path}>{cell.row.original.workingDir}</code>
          ) : (
            <span className={styles.muted}>-</span>
          ),
      },
    ],
    [id, openEdit],
  )

  return (
    <div>
      <header className={styles.head}>
        <h1 className={styles.title}>Stacks</h1>
        <Button size="sm" icon={<LuPlus />} onClick={() => setEditor({ mode: 'new' })}>
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

      <StackEditor envId={Number(id)} state={editor} onClose={() => setEditor(null)} />
    </div>
  )
}
