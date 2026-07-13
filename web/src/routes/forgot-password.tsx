import { createFileRoute, Link } from '@tanstack/react-router'
import { AuthLayout } from '../components/AuthLayout'
import styles from './forgot-password.module.css'

export const Route = createFileRoute('/forgot-password')({
  component: ForgotPasswordPage,
})

function ForgotPasswordPage() {
  return (
    <AuthLayout title="Reset your password">
      <p className={styles.text}>
        Password reset is coming soon. In the meantime, ask an administrator to
        reset your account.
      </p>
      <Link to="/login" className={styles.back}>
        Back to sign in
      </Link>
    </AuthLayout>
  )
}
