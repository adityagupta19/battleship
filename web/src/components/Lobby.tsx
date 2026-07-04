interface LobbyProps {
  username: string
  onUsernameChange: (v: string) => void
  onJoin: () => void
  loading: boolean
  error: string | null
  userId: number | null
  connected: boolean
  onFindMatch: () => void
  matching: boolean
}

export function Lobby({
  username,
  onUsernameChange,
  onJoin,
  loading,
  error,
  userId,
  connected,
  onFindMatch,
  matching,
}: LobbyProps) {
  if (!userId) {
    return (
      <div className="panel">
        <h1>Battleship</h1>
        <p>Enter a username to join.</p>
        <input
          type="text"
          placeholder="Username"
          value={username}
          onChange={(e) => onUsernameChange(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && onJoin()}
        />
        <button type="button" onClick={onJoin} disabled={loading || !username.trim()}>
          {loading ? 'Joining…' : 'Join'}
        </button>
        {error && <p className="error">{error}</p>}
      </div>
    )
  }

  return (
    <div className="panel">
      <h1>Battleship</h1>
      <p>Logged in as <strong>{username}</strong> (id: {userId})</p>
      <p className={connected ? 'ok' : 'warn'}>
        {connected ? 'Connected' : 'Connecting…'}
      </p>
      <button type="button" onClick={onFindMatch} disabled={!connected || matching}>
        {matching ? 'Finding match…' : 'Find Match'}
      </button>
      {error && <p className="error">{error}</p>}
    </div>
  )
}
