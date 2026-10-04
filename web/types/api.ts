export interface APIChallenge {
  id: string
  challenger: string
  challengee: string
  state: ChallengeState
  createdBy: string
  createdAt: string
  gameId?: string
}
export type ChallengeState = 'pending' | 'active' | 'denied' | 'complete'

export interface APIUserStats {
  userId: string
  mmr: number
  wins: number
  losses: number
  updatedAt: string
}

export interface APILeaderboardEntry {
  userId: string
  name: string
  image: string
  mmr: number
  wins: number
  losses: number
}


export interface APIUser {
  id: string
  name: string
  image: string
}

export interface APIGetResponse<T> {
  data?: T
  error?: string
}
