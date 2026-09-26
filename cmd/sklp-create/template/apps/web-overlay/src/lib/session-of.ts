import type { Guarded } from '@lalternative/auth/urbangate'

export interface ServerSession {
  user: {
    id: string
    email: string
    name: string
    role: 'admin' | 'user'
  }
}

export class IdentityProviderUnavailable extends Error {}

export function sessionOf(guarded: Guarded): ServerSession | null {
  if ('response' in guarded) {
    if (guarded.response.status === 503)
      throw new IdentityProviderUnavailable('urbangate cannot answer')
    return null
  }
  const { user } = guarded.session
  return { user: { id: user.id, email: user.email, name: user.name, role: user.role } }
}
