import { Toast } from '@base-ui/react/toast'

export const toastManager = Toast.createToastManager()

type ToastType = 'success' | 'error' | 'info'

function make(type: ToastType, priority: 'low' | 'high') {
  return (title: string, description?: string) =>
    toastManager.add({ title, description, type, priority })
}

export const toast = {
  success: make('success', 'low'),
  error: make('error', 'high'),
  info: make('info', 'low'),
}
