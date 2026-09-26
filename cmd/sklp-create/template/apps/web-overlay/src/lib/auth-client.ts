import { createUrbangateAuthClient } from '@lalternative/auth/urbangate-client'

const baseURL =
  typeof window === 'undefined'
    ? (process.env.APP_URL ?? 'http://localhost:5273')
    : window.location.origin

export const authClient = createUrbangateAuthClient({ baseURL })
