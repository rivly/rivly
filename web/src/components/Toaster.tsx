import { Toast } from '@base-ui/react/toast'
import type { ReactNode } from 'react'
import type { IconType } from 'react-icons'
import { LuCircleAlert, LuCircleCheck, LuInfo, LuX } from 'react-icons/lu'
import { toastManager } from '../lib/toast'
import styles from './Toaster.module.css'

const ICONS: Record<string, IconType> = {
  success: LuCircleCheck,
  error: LuCircleAlert,
  info: LuInfo,
}

export function Toaster({ children }: { children: ReactNode }) {
  return (
    <Toast.Provider toastManager={toastManager} limit={4}>
      {children}
      <Toast.Portal>
        <Toast.Viewport className={styles.viewport}>
          <ToastList />
        </Toast.Viewport>
      </Toast.Portal>
    </Toast.Provider>
  )
}

function ToastList() {
  const { toasts } = Toast.useToastManager()
  return (
    <>
      {toasts.map((toast) => {
        const type = toast.type ?? 'info'
        const Icon = ICONS[type] ?? LuInfo
        return (
          <Toast.Root
            key={toast.id}
            toast={toast}
            className={`${styles.toast} ${styles[type] ?? ''}`}
          >
            <Icon className={styles.icon} />
            <div className={styles.text}>
              <Toast.Title className={styles.title} />
              {toast.description && (
                <Toast.Description className={styles.description} />
              )}
            </div>
            <Toast.Close className={styles.close} aria-label="Dismiss">
              <LuX />
            </Toast.Close>
          </Toast.Root>
        )
      })}
    </>
  )
}
