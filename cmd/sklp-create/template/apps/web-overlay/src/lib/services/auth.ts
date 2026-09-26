import type { UserProfile } from '../types/auth'

export async function getProfile(): Promise<UserProfile> {
  const response = await fetch('/api/auth/profile', { credentials: 'include' })
  if (!response.ok) {
    throw new Error('Not authenticated')
  }
  return response.json() as Promise<UserProfile>
}
