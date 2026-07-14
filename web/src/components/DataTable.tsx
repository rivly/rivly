import { useMemo, useState, type CSSProperties, type ReactNode } from 'react'
import { Menu } from '@base-ui/react/menu'
import { Checkbox as BaseCheckbox } from '@base-ui/react/checkbox'
import {
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type Column,
  type ColumnDef,
  type RowData,
  type RowSelectionState,
  type SortingState,
  type VisibilityState,
} from '@tanstack/react-table'
import {
  LuCheck,
  LuChevronDown,
  LuChevronLeft,
  LuChevronRight,
  LuChevronUp,
  LuChevronsUpDown,
  LuInbox,
  LuMinus,
  LuSearch,
  LuSlidersHorizontal,
} from 'react-icons/lu'
import { Button } from './Button'
import { Select } from './Select'
import styles from './DataTable.module.css'

declare module '@tanstack/react-table' {
  interface ColumnMeta<TData extends RowData, TValue> {
    sticky?: 'left' | 'right'
  }
}

function pinnedStyle<TData>(
  column: Column<TData, unknown>,
): CSSProperties | undefined {
  const pinned = column.getIsPinned()
  if (!pinned) {
    return undefined
  }
  return {
    position: 'sticky',
    left: pinned === 'left' ? column.getStart('left') : undefined,
    right: pinned === 'right' ? column.getAfter('right') : undefined,
    width: column.getSize(),
    minWidth: column.getSize(),
    maxWidth: column.getSize(),
    zIndex: 1,
  }
}

function pinnedClass<TData>(
  column: Column<TData, unknown>,
  styleMap: Record<string, string>,
): string {
  const pinned = column.getIsPinned()
  if (!pinned) {
    return ''
  }
  if (pinned === 'left' && column.getIsLastColumn('left')) {
    return `${styleMap.pinned} ${styleMap.pinnedLeftEdge}`
  }
  if (pinned === 'right' && column.getIsFirstColumn('right')) {
    return `${styleMap.pinned} ${styleMap.pinnedRightEdge}`
  }
  return styleMap.pinned
}

const PAGE_SIZES = [
  { label: '10', value: '10' },
  { label: '25', value: '25' },
  { label: '50', value: '50' },
  { label: '100', value: '100' },
]

type DataTableProps<T> = {
  data: T[]
  columns: ColumnDef<T, any>[]
  searchPlaceholder?: string
  emptyMessage?: string
  initialPageSize?: number
  onRowClick?: (row: T) => void
  enableSelection?: boolean
  getRowId?: (row: T) => string
  renderBulkActions?: (rows: T[], clearSelection: () => void) => ReactNode
}

export function DataTable<T>({
  data,
  columns,
  searchPlaceholder = 'Search…',
  emptyMessage = 'No results.',
  initialPageSize = 10,
  onRowClick,
  enableSelection = false,
  getRowId,
  renderBulkActions,
}: DataTableProps<T>) {
  const [sorting, setSorting] = useState<SortingState>([])
  const [globalFilter, setGlobalFilter] = useState('')
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({})

  const allColumns = useMemo(
    () => (enableSelection ? [selectionColumn<T>(), ...columns] : columns),
    [columns, enableSelection],
  )

  const columnPinning = useMemo(() => {
    const left: string[] = []
    const right: string[] = []
    for (const column of allColumns) {
      const id =
        column.id ??
        ('accessorKey' in column ? String(column.accessorKey) : undefined)
      if (!id) {
        continue
      }
      if (column.meta?.sticky === 'left') {
        left.push(id)
      }
      if (column.meta?.sticky === 'right') {
        right.push(id)
      }
    }
    return { left, right }
  }, [allColumns])

  const table = useReactTable({
    data,
    columns: allColumns,
    state: { sorting, globalFilter, columnVisibility, columnPinning, rowSelection },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    enableRowSelection: enableSelection,
    getRowId,
    globalFilterFn: 'includesString',
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    initialState: { pagination: { pageSize: initialPageSize } },
  })

  const rows = table.getRowModel().rows
  const hideable = table
    .getAllLeafColumns()
    .filter((column) => column.getCanHide() && column.id !== '__select')
  const pagination = table.getState().pagination
  const totalRows = table.getFilteredRowModel().rows.length
  const selectedRows = enableSelection
    ? table.getSelectedRowModel().rows.map((row) => row.original)
    : []

  return (
    <div className={styles.wrap}>
      <div className={styles.toolbar}>
        {selectedRows.length > 0 && renderBulkActions ? (
          renderBulkActions(selectedRows, () => table.resetRowSelection())
        ) : (
          <div className={styles.search}>
            <LuSearch className={styles.searchIcon} />
            <input
              className={styles.searchInput}
              value={globalFilter}
              onChange={(event) => setGlobalFilter(event.target.value)}
              placeholder={searchPlaceholder}
              aria-label="Search"
            />
          </div>
        )}

        <Menu.Root>
          <Menu.Trigger
            render={
              <Button variant="secondary" size="sm" icon={<LuSlidersHorizontal />}>
                Columns
              </Button>
            }
          />
          <Menu.Portal>
            <Menu.Positioner
              className={styles.menuPositioner}
              sideOffset={6}
              align="end"
            >
              <Menu.Popup className={styles.menuPopup}>
                {hideable.map((column) => (
                  <Menu.CheckboxItem
                    key={column.id}
                    className={styles.menuItem}
                    checked={column.getIsVisible()}
                    onCheckedChange={() => column.toggleVisibility()}
                    closeOnClick={false}
                  >
                    <span className={styles.menuCheck}>
                      <Menu.CheckboxItemIndicator>
                        <LuCheck />
                      </Menu.CheckboxItemIndicator>
                    </span>
                    {columnLabel(column)}
                  </Menu.CheckboxItem>
                ))}
              </Menu.Popup>
            </Menu.Positioner>
          </Menu.Portal>
        </Menu.Root>
      </div>

      <div className={styles.tableWrap}>
        <table className={styles.table}>
          <thead>
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  if (header.isPlaceholder) {
                    return (
                      <th
                        key={header.id}
                        className={`${styles.th} ${pinnedClass(header.column, styles)}`}
                        style={pinnedStyle(header.column)}
                      />
                    )
                  }
                  const content = flexRender(
                    header.column.columnDef.header,
                    header.getContext(),
                  )
                  return (
                    <th
                      key={header.id}
                      className={`${styles.th} ${header.column.id === '__select' ? styles.selectCell : ''} ${pinnedClass(header.column, styles)}`}
                      style={pinnedStyle(header.column)}
                    >
                      {header.column.getCanSort() ? (
                        <button
                          type="button"
                          className={styles.sortButton}
                          onClick={header.column.getToggleSortingHandler()}
                        >
                          {content}
                          <SortIcon sorted={header.column.getIsSorted()} />
                        </button>
                      ) : (
                        <span className={styles.thPlain}>{content}</span>
                      )}
                    </th>
                  )
                })}
              </tr>
            ))}
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr>
                <td
                  className={styles.empty}
                  colSpan={table.getVisibleLeafColumns().length}
                >
                  <LuInbox className={styles.emptyIcon} />
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              rows.map((row) => (
                <tr
                  key={row.id}
                  className={onRowClick ? styles.rowClickable : undefined}
                  onClick={onRowClick ? () => onRowClick(row.original) : undefined}
                >
                  {row.getVisibleCells().map((cell) => (
                    <td
                      key={cell.id}
                      className={`${styles.td} ${cell.column.id === '__select' ? styles.selectCell : ''} ${pinnedClass(cell.column, styles)}`}
                      style={pinnedStyle(cell.column)}
                    >
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <div className={styles.footer}>
        <div className={styles.footerGroup}>
          <span className={styles.footerLabel}>Rows per page</span>
          <Select
            items={PAGE_SIZES}
            value={String(pagination.pageSize)}
            onValueChange={(value) =>
              table.setPageSize(Number(value) || initialPageSize)
            }
            aria-label="Rows per page"
          />
        </div>

        <div className={styles.footerGroup}>
          <span className={styles.footerLabel}>
            {totalRows} {totalRows === 1 ? 'row' : 'rows'}
          </span>
          <div className={styles.pagination}>
            <button
              type="button"
              className={styles.pageButton}
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
              aria-label="Previous page"
            >
              <LuChevronLeft />
            </button>
            <span className={styles.pageIndicator}>
              Page {pagination.pageIndex + 1} of {table.getPageCount() || 1}
            </span>
            <button
              type="button"
              className={styles.pageButton}
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
              aria-label="Next page"
            >
              <LuChevronRight />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function SortIcon({ sorted }: { sorted: false | 'asc' | 'desc' }) {
  if (sorted === 'asc') {
    return <LuChevronUp className={styles.sortIcon} />
  }
  if (sorted === 'desc') {
    return <LuChevronDown className={styles.sortIcon} />
  }
  return <LuChevronsUpDown className={`${styles.sortIcon} ${styles.sortMuted}`} />
}

function columnLabel<T>(column: Column<T, unknown>): string {
  const header = column.columnDef.header
  return typeof header === 'string' ? header : column.id
}

function selectionColumn<T>(): ColumnDef<T, any> {
  return {
    id: '__select',
    size: 52,
    enableSorting: false,
    enableHiding: false,
    meta: { sticky: 'left' },
    header: ({ table }) => (
      <SelectionCheckbox
        checked={
          table.getIsAllPageRowsSelected()
            ? true
            : table.getIsSomePageRowsSelected()
              ? 'indeterminate'
              : false
        }
        onChange={(value) => table.toggleAllPageRowsSelected(value)}
        label="Select all rows"
      />
    ),
    cell: ({ row }) => (
      <SelectionCheckbox
        checked={row.getIsSelected()}
        onChange={(value) => row.toggleSelected(value)}
        label="Select row"
      />
    ),
  }
}

function SelectionCheckbox({
  checked,
  onChange,
  label,
}: {
  checked: boolean | 'indeterminate'
  onChange: (value: boolean) => void
  label: string
}) {
  return (
    <BaseCheckbox.Root
      className={styles.checkbox}
      checked={checked === true}
      indeterminate={checked === 'indeterminate'}
      onCheckedChange={(value) => onChange(value)}
      aria-label={label}
      onClick={(event) => event.stopPropagation()}
    >
      <BaseCheckbox.Indicator className={styles.checkboxIndicator}>
        {checked === 'indeterminate' ? <LuMinus /> : <LuCheck />}
      </BaseCheckbox.Indicator>
    </BaseCheckbox.Root>
  )
}
