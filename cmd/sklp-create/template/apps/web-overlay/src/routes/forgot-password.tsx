import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { ForgotPasswordForm } from '@lalternative/auth'
import { authClient } from '@/lib/auth-client'
import { AuthScreen } from '@/components/auth-screen'

export const Route = createFileRoute('/forgot-password')({ component: ForgotPassword })

function ForgotPassword() {
  const navigate = useNavigate()

  return (
    <AuthScreen subtitle="Mot de passe oublié">
      <ForgotPasswordForm
        authClient={authClient}
        linkComponent={Link}
        loginUrl="/sign-in"
        onSuccess={(email) => navigate({ to: '/reset-password', search: { email } })}
      />
    </AuthScreen>
  )
}
