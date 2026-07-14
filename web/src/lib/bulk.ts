import type { ActionResult } from './containers'
import { toast } from './toast'

type ReportOptions = {
  verb: string
  noun: string
  failHint: (failed: number) => string
  clear: () => void
}

export function reportBulk(results: ActionResult[], opts: ReportOptions) {
  const ok = results.filter((result) => result.ok).length
  const failed = results.length - ok
  if (failed === 0) {
    toast.success(`${opts.verb} ${ok} ${opts.noun}${ok > 1 ? 's' : ''}`)
  } else {
    toast.error(`${opts.verb} ${ok}/${results.length} ${opts.noun}s`, opts.failHint(failed))
  }
  opts.clear()
}

export function bulkActionError() {
  toast.error('Action failed', 'Could not reach the environment')
}
