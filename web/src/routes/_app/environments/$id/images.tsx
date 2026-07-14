import { createFileRoute } from '@tanstack/react-router'
import { useMemo } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { DataTable } from '../../../../components/DataTable'
import { ImageBulkBar } from '../../../../components/ImageBulkBar'
import { Loader } from '../../../../components/Loader'
import { useImages, type Image } from '../../../../lib/images'
import { formatBytes, timeAgo } from '../../../../lib/format'
import styles from './images.module.css'

export const Route = createFileRoute('/_app/environments/$id/images')({
  head: () => ({ meta: [{ title: 'Images · Rivly' }] }),
  component: ImagesPage,
})

function ImagesPage() {
  const { id } = Route.useParams()
  const { data: images, isPending, isError } = useImages(Number(id))

  const columns = useMemo<ColumnDef<Image>[]>(
    () => [
      {
        id: 'image',
        header: 'Image',
        accessorFn: (row) => row.tags.join(' '),
        cell: (cell) => <ImageCell image={cell.row.original} />,
      },
      {
        accessorKey: 'id',
        header: 'Image ID',
        cell: (cell) => (
          <code className={styles.id}>{cell.row.original.id.slice(0, 12)}</code>
        ),
      },
      {
        accessorKey: 'size',
        header: 'Size',
        cell: (cell) => (
          <span className={styles.muted}>{formatBytes(cell.row.original.size)}</span>
        ),
      },
      {
        accessorKey: 'created',
        header: 'Created',
        cell: (cell) => (
          <span className={styles.muted}>{timeAgo(cell.row.original.created)}</span>
        ),
      },
    ],
    [],
  )

  return (
    <div>
      <header className={styles.head}>
        <h1 className={styles.title}>Images</h1>
      </header>

      {isPending && <Loader />}
      {isError && <p className={styles.message}>Could not load images.</p>}
      {images && (
        <DataTable
          data={images}
          columns={columns}
          searchPlaceholder="Search images…"
          emptyMessage="No images on this host."
          initialPageSize={25}
          enableSelection
          getRowId={(image) => image.id}
          renderBulkActions={(selected, clear) => (
            <ImageBulkBar envId={Number(id)} selected={selected} clear={clear} />
          )}
        />
      )}
    </div>
  )
}

function ImageCell({ image }: { image: Image }) {
  return (
    <span className={styles.imageCell}>
      {image.tags.length > 0 ? (
        <span className={styles.name}>{image.tags.join(', ')}</span>
      ) : (
        <span className={styles.untagged}>{'<none>'}</span>
      )}
      {!image.inUse && <span className={styles.unusedBadge}>Unused</span>}
    </span>
  )
}
