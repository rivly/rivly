import styles from './ImageTag.module.css'

export function ImageTag({ image }: { image: string }) {
  return <code className={styles.tag}>{image}</code>
}
