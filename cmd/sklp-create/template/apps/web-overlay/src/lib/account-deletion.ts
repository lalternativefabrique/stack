import type { AccountDeletionSteps } from '@lalternative/auth/urbangate'
import { CORE_URL } from '@/lib/core-url'

export function accountDeletion(accessToken: string): AccountDeletionSteps {
  return {
    deleteData: async () => {
      const res = await fetch(`${CORE_URL}/api/v1/account`, {
        method: 'DELETE',
        headers: { authorization: `Bearer ${accessToken}` },
        signal: AbortSignal.timeout(30_000),
      })
      if (!res.ok) throw new Error(`core account deletion answered ${res.status}`)
    },
    onError: (step, error) => console.error(`[account] ${step} step failed`, error),
  }
}
