import { useWebSocket, type UseWebSocketOptions } from '@vueuse/core'
import { toast } from '~/components/ui/toast/use-toast'

interface Player {
  uid: string
  score: number
  name: string
  image: string
}
interface GameState {
  id: string // session id
  you: Player
  opponent?: Player
  // opponentScore: number
  serving?: string
  status: string
  winner?: string
}

interface GameResponseState {
  type: 'state'
  payload: {
    id: string
    serving?: string
    status: string
    version: number
    players: Record<string, Player>
    updatedAt: string
    meta?: { [key: string]: string }
  }
}

interface GameResponseParticipants {
  type: 'participants'
  count: number
}

const initializeWebsocketConnection = async (
  gameID: string,
  sessionData: ReturnType<typeof authClient.getSession>,
  options?: UseWebSocketOptions
) => {
  const config = useRuntimeConfig()
  const token = sessionData?.session.token
  // Websocket
  // From what i can tell, websocket conns happen before headers, so can't send
  // as Auth header like normal requests at least
  const socket = useWebSocket(
    `${config.public.apiBasedUrlWs}/v1/game/${gameID}/ws?token=${token}`,
    options
  )

  return socket
}

export async function useGameState(gameID: string) {
  const { data: sessionData } = await authClient.getSession()
  // State
  const state = ref<GameState>()
  const connected = ref(false)
  const websocket = ref<ReturnType<typeof useWebSocket>>()
  if(!gameID) throw new Error('gameID is required')

  initializeWebsocketConnection(gameID, sessionData, {
    onMessage: async (ws, event) => {
      console.log('Message from server: ', event.data)
      await messageResponseHandler(event)
    },
    onConnected(ws) {
      console.log('Connected to server ', ws)
      connected.value = true
    },
    onDisconnected: (ws, event) => {
      console.log('Disconnected from server ', event)
      connected.value = false
      websocket.value?.close()
      if (event.code !== 1000) {
        toast({ title: 'Disconnected', description: 'Lost connection to the game server.', variant: 'destructive' })
      }
    },
  }).then(value => (websocket.value = value))

  const messageResponseHandler = async (event: MessageEvent) => {
    // TODO: Can do some kind of zod parsing here mayhaps to ensure the type of the message
    const data = await JSON.parse(event.data)
    if (data.type === 'state') stateResponseHandler(data as GameResponseState)
    if (data.type === 'participants') participantsResponseHandler(data as GameResponseParticipants)
  }

  const getPlayers = (data: GameResponseState): { you?: Player; opponent?: Player } => {
    // 🚧 insane cook below 🚧
    const yourID = sessionData?.user.id
    if (!yourID) return { you: undefined, opponent: undefined }

    // 🖕🏻🤓🖕🏻
    const you = data.payload.players[yourID]
    let opponent

    // ♿ ♿ ♿ ♿ ♿ ♿ ♿ ♿ ♿ ♿
    for (const [_, player] of Object.entries(data.payload.players)) {
      if (player.uid !== yourID) opponent = player
    }
    //♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿♿
    //hejdå :3
    return { you, opponent }
  }

  const stateResponseHandler = (data: GameResponseState) => {
    console.log('Data: ', data)
    if (!sessionData?.user) return

    const { you, opponent } = getPlayers(data)
    if (!you) return // smthin very wong

    state.value = {
      id: data.payload.id,
      you: you,
      opponent: opponent,
      serving: data.payload.serving,
      status: data.payload.status,
      winner: data.payload.meta?.winner,
    }
  }

  const participantsResponseHandler = (data: GameResponseParticipants) => {
    console.log('participants', data)
  }

  // Set Score
  const incrementScore = () => {
    if (!connected.value) {
      console.warn('Not connected to websocket, cannot send score update')
      return
    }

    if (state.value?.status === 'finished') {
      console.warn("can't update a finished game bozo")
      return
    }

    if (!state.value?.you) {
      console.warn("You don't exist 👹")
      return
    }

    if (!websocket.value) {
      console.warn("Bruv ain't got no websocket 😂")
      return
    }

    // eager update
    console.log('incrementing score')
    state.value.you.score += 1
    // send message here
    const message = {
      type: 'score:update',
      payload: {
        userId: state.value.you.uid,
        delta: 1,
      },
    }

    websocket.value.send(JSON.stringify(message))
  }

  return {
    connected,
    state,
    incrementScore,
  }
}
