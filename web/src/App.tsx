import { useCallback, useEffect, useState } from 'react'
import { registerUser } from './api'
import { Lobby } from './components/Lobby'
import { GameView } from './components/GameView'
import { useGameSocket } from './useGameSocket'
import type { BoardView, GamePhase, ServerMessage } from './types'
import { DEFAULT_SHIPS } from './types'
import './App.css'

export default function App() {
  const [username, setUsername] = useState('')
  const [userId, setUserId] = useState<number | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [phase, setPhase] = useState<GamePhase>('lobby')
  const [matching, setMatching] = useState(false)

  const [gameId, setGameId] = useState<number | null>(null)
  const [opponentId, setOpponentId] = useState<number | null>(null)
  const [status, setStatus] = useState('placement')
  const [yourTurn, setYourTurn] = useState(false)
  const [youReady, setYouReady] = useState(false)
  const [opponentReady, setOpponentReady] = useState(false)
  const [yourBoard, setYourBoard] = useState<BoardView | undefined>()
  const [opponentView, setOpponentView] = useState<BoardView | undefined>()
  const [lastResult, setLastResult] = useState<string | null>(null)
  const [winnerId, setWinnerId] = useState<number | null>(null)

  const { connected, send, onMessage } = useGameSocket(userId)

  const applyGameState = useCallback((msg: ServerMessage) => {
    if (msg.game_id) setGameId(msg.game_id)
    if (msg.status) {
      setStatus(msg.status)
      setPhase(msg.status === 'finished' ? 'finished' : msg.status === 'active' ? 'active' : 'placement')
    }
    if (msg.your_turn !== undefined) setYourTurn(msg.your_turn)
    if (msg.you_ready !== undefined) setYouReady(msg.you_ready)
    if (msg.opponent_ready !== undefined) setOpponentReady(msg.opponent_ready)
    if (msg.your_board) setYourBoard(msg.your_board)
    if (msg.opponent_view) setOpponentView(msg.opponent_view)
    if (msg.winner_id) setWinnerId(msg.winner_id)
  }, [])

  useEffect(() => {
    onMessage((msg) => {
      switch (msg.type) {
        case 'match_found':
          setMatching(false)
          setGameId(msg.game_id ?? null)
          setOpponentId(msg.opponent_id ?? null)
          setPhase('placement')
          setError(null)
          break
        case 'game_state':
          applyGameState(msg)
          break
        case 'shot_result': {
          const parts = [msg.result]
          if (msg.sunk_ship) parts.push(`sunk ${msg.sunk_ship}`)
          setLastResult(parts.join(', '))
          if (msg.game_over) {
            setPhase('finished')
            setWinnerId(msg.winner_id ?? null)
          }
          break
        }
        case 'game_over':
          setPhase('finished')
          setWinnerId(msg.winner_id ?? null)
          break
        case 'error':
          setError(msg.message ?? 'unknown error')
          setMatching(false)
          break
      }
    })
  }, [onMessage, applyGameState])

  const handleJoin = async () => {
    setLoading(true)
    setError(null)
    try {
      const id = await registerUser(username.trim())
      setUserId(id)
      setPhase('lobby')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'join failed')
    } finally {
      setLoading(false)
    }
  }

  const handleFindMatch = () => {
    setMatching(true)
    setError(null)
    send({ type: 'find_match' })
  }

  const handleAutoPlace = () => {
    if (!gameId) return
    send({ type: 'place_ships', game_id: gameId, ships: DEFAULT_SHIPS })
  }

  const handleFire = (x: number, y: number) => {
    if (!gameId) return
    send({ type: 'fire_shot', game_id: gameId, x, y })
  }

  if (phase === 'placement' || phase === 'active' || phase === 'finished') {
    if (gameId && opponentId && userId) {
      return (
        <GameView
          gameId={gameId}
          opponentId={opponentId}
          status={status}
          yourTurn={yourTurn}
          youReady={youReady}
          opponentReady={opponentReady}
          yourBoard={yourBoard}
          opponentView={opponentView}
          lastResult={lastResult}
          winnerId={winnerId}
          userId={userId}
          onAutoPlace={handleAutoPlace}
          onFire={handleFire}
        />
      )
    }
  }

  return (
    <Lobby
      username={username}
      onUsernameChange={setUsername}
      onJoin={handleJoin}
      loading={loading}
      error={error}
      userId={userId}
      connected={connected}
      onFindMatch={handleFindMatch}
      matching={matching}
    />
  )
}
