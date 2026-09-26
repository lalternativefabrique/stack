import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { ResetPasswordForm } from '@lalternative/auth'
import { authClient } from '@/lib/auth-client'
import { AuthScreen } from '@/components/auth-screen'

export const Route = createFileRoute('/reset-password')({
  validateSearch: (search: Record<string, unknown>) => ({
    email: typeof search.email === 'string' ? search.email : '',
  }),
  component: ResetPassword,
})

function ResetPassword() {
  const navigate = useNavigate()
  const { email } = Route.useSearch()

  return (
    <AuthScreen subtitle="Nouveau mot de passe">
      <ResetPasswordForm
        authClient={authClient}
        email={email}
        linkComponent={Link}
        loginUrl="/sign-in"
        onSuccess={() => navigate({ to: '/sign-in' })}
      />
    </AuthScreen>
  )
}
