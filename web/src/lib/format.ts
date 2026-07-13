const BYTE_UNITS = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB']

export function formatBytes(bytes: number): string {
  if (bytes <= 0) {
    return '0 B'
  }
  const exponent = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    BYTE_UNITS.length - 1,
  )
  const value = bytes / 1024 ** exponent
  return `${value.toFixed(exponent === 0 ? 0 : 1)} ${BYTE_UNITS[exponent]}`
}
