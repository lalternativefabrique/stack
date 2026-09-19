import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { AdminSetupForm } from '@lalternative/admin'
import '@lalternative/admin/styles.css'

/**
 * First-admin bootstrap. Trailing underscore keeps it outside the `/admin`
 * layout guard (it must be reachable with no session). Redirects away once an
 * admin already exists. The account is created and the role granted server-side
 * in /api/admin/setup — this only collects the fields and the emailed code.
 */
export const Route = createFileRoute('/admin/setup_')({
  component: AdminSetupPage,
})

async function postSetup(body: unknown) {
  const res = await fetch('/api/admin/setup', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const data = (await res.json().catch(() => ({}))) as { error?: string }
  if (!res.ok) throw new Error(data.error ?? 'Setup failed')
}

function AdminSetupPage() {
  const navigate = useNavigate()
  const [ready, setReady] = useState(false)
  // The form hands onRequestCode the address alone, but signing the account up
  // is what makes Better Auth send the code — and that needs the name and the
  // password too. They are read off the form's own inputs as they are typed,
  // which is the only place they exist before submit.
  const details = useRef({ name: '', password: '' })

  useEffect(() => {
    fetch('/api/admin/setup')
      .then((res) => res.json())
      .then((data: { hasAdmin: boolean }) => {
        if (data.hasAdmin) navigate({ to: '/admin/login' })
        else setReady(true)
      })
      .catch(() => setReady(true))
  }, [navigate])

  if (!ready) return null

  return (
    <div
      onChange={(event) => {
        const input = event.target as HTMLInputElement
        if (input.type === 'password') details.current.password = input.value
        else if (input.type === 'text') details.current.name = input.value
      }}
    >
      <AdminSetupForm
        onRequestCode={async (email) => {
          await postSetup({
            name: details.current.name,
            email,
            password: details.current.password,
          })
        }}
        onSubmit={async ({ name, email, password, code }) => {
          await postSetup({ name, email, password, code })
        }}
        onSuccess={() => navigate({ to: '/admin/login' })}
      />
    </div>
  )
}
