import type { APIChallenge } from '~/types/api'

export function headToHead(
  challenges: APIChallenge[],
  myId: string
): Record<string, { wins: number; losses: number }> {
  const out: Record<string, { wins: number; losses: number }> = {}
  for (const c of challenges) {
    if (c.gameState !== 'completed' || !c.winnerId) continue
    const opponent = c.challenger === myId ? c.challengee : c.challenger
    const r = (out[opponent] ??= { wins: 0, losses: 0 })
    if (c.winnerId === myId) r.wins++
    else r.losses++
  }
  return out
}

export const kd = (wins: number, losses: number): string =>
  losses === 0 ? wins.toFixed(2) : (wins / losses).toFixed(2)
