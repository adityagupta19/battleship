import { Board } from './Board'
import type { BoardView } from '../types'

interface GameViewProps {
  gameId: number
  opponentId: number
  status: string
  yourTurn: boolean
  youReady: boolean
  opponentReady: boolean
  yourBoard?: BoardView
  opponentView?: BoardView
  lastResult: string | null
  winnerId: number | null
  userId: number
  onAutoPlace: () => void
  onFire: (x: number, y: number) => void
}

export function GameView({
  gameId,
  opponentId,
  status,
  yourTurn,
  youReady,
  opponentReady,
  yourBoard,
  opponentView,
  lastResult,
  winnerId,
  userId,
  onAutoPlace,
  onFire,
}: GameViewProps) {
  const finished = status === 'finished'

  return (
    <div className="game">
      <header className="game-header">
        <h1>Game #{gameId}</h1>
        <p>Opponent: {opponentId}</p>
        <p>Status: <strong>{status}</strong></p>
        {!finished && status === 'active' && (
          <p className={yourTurn ? 'ok' : 'warn'}>
            {yourTurn ? 'Your turn — fire on enemy waters' : "Opponent's turn"}
          </p>
        )}
        {status === 'placement' && (
          <p>
            You: {youReady ? 'ready' : 'placing ships'} · Opponent:{' '}
            {opponentReady ? 'ready' : 'placing ships'}
          </p>
        )}
        {lastResult && <p className="result">Last shot: {lastResult}</p>}
        {finished && (
          <p className="banner">
            {winnerId === userId ? 'You won!' : 'You lost.'}
          </p>
        )}
      </header>

      {status === 'placement' && !youReady && (
        <button type="button" className="primary" onClick={onAutoPlace}>
          Auto Place Ships
        </button>
      )}

      <div className="boards">
        <Board title="Your Fleet" board={yourBoard} />
        <Board
          title="Enemy Waters"
          board={opponentView}
          clickable={status === 'active' && yourTurn && !finished}
          onCellClick={onFire}
        />
      </div>
    </div>
  )
}
