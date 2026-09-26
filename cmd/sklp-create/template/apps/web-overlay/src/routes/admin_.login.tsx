import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { AdminLoginForm } from '@lalternative/admin'
import '@lalternative/admin/styles.css'
import { authClient } from '@/lib/auth-client'
import { getProfile } from '@/lib/services/auth'
import { AdminScreen } from '@/components/admin-screen'

export const Route = createFileRoute('/admin_/login')({
  component: AdminLoginPage,
})

function AdminLoginPage() {
  const navigate = useNavigate()
  return (
    <AdminScreen>
      <AdminLoginForm
        authClient={authClient}
        getProfile={getProfile}
        sso={{
          signIn: () => authClient.signIn.urbangate({ callbackURL: '/admin' }),
          only: true,
        }}
        onSuccess={() => navigate({ to: '/admin' })}
      />
    </AdminScreen>
  )
}
