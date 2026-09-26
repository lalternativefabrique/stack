import type { Auth } from '@/lib/auth'

type Guarded = Awaited<ReturnType<Auth['requireAdmin']>>

export type AdminAccess = 'granted' | 'sign_in' | 'unavailable'

export function adminAccess(guarded: Guarded): AdminAccess {
  if (!('response' in guarded)) return 'granted'
  return guarded.response.status === 503 ? 'unavailable' : 'sign_in'
}
