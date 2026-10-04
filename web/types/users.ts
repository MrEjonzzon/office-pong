import type { ChallengeState } from "./api"

export interface ListUser {
  id: string
  name: string
  image: string
  state?: ChallengeState
}
