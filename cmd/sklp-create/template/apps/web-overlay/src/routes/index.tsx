import { createFileRoute, Link, redirect, useNavigate } from '@tanstack/react-router'
import { authClient } from '@/lib/auth-client'
import { getServerSession } from '@/lib/get-server-session'

export const Route = createFileRoute('/')({
  beforeLoad: async () => {
    const session = await getServerSession()
    if (!session) throw redirect({ to: '/sign-in' })
    return { session }
  },
  component: Home,
})

function Home() {
  const { session } = Route.useRouteContext()
  const navigate = useNavigate()
  const signOut = async () => {
    await authClient.signOut()
    await navigate({ to: '/sign-in' })
  }
  return (
    <div className="p-8">
      <h1 className="text-2xl font-semibold">Bonjour {session.user.name}</h1>
      <nav className="mt-6 flex gap-4 text-sm">
        <Link to="/settings" className="underline underline-offset-4">
          Réglages
        </Link>
        {session.user.role === 'admin' && (
          <Link to="/admin" className="underline underline-offset-4">
            Administration
          </Link>
        )}
        <button type="button" onClick={() => void signOut()} className="underline underline-offset-4">
          Se déconnecter
        </button>
      </nav>
    </div>
  )
}
