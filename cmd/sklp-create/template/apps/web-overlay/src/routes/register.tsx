import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { RegisterForm } from '@lalternative/auth'
import { authClient } from '@/lib/auth-client'
import { AuthScreen } from '@/components/auth-screen'

export const Route = createFileRoute('/register')({ component: Register })

function Register() {
  const navigate = useNavigate()

  return (
    <AuthScreen>
      <RegisterForm
        authClient={authClient}
        linkComponent={Link}
        loginUrl="/sign-in"
        onSuccess={(email) => navigate({ to: '/verify-email', search: { email } })}
      />
    </AuthScreen>
  )
}
