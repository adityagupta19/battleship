import { useCallback, useEffect, useRef, useState } from 'react'
import type { ServerMessage } from './types'

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'

export function useGameSocket(userId: number | null) {
  const wsRef = useRef<WebSocket | null>(null)
  const [connected, setConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<ServerMessage | null>(null)
  const handlerRef = useRef<((msg: ServerMessage) => void) | null>(null)

  useEffect(() => {
    if (!userId) return

    const url = `${WS_URL}?user_id=${userId}`
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => setConnected(true)
    ws.onclose = () => setConnected(false)
    ws.onerror = () => setConnected(false)
    ws.onmessage = (ev) => {
      const msg: ServerMessage = JSON.parse(ev.data)
      setLastMessage(msg)
      handlerRef.current?.(msg)
    }

    return () => {
      ws.close()
      wsRef.current = null
    }
  }, [userId])

  const send = useCallback((payload: object) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(payload))
    }
  }, [])

  const onMessage = useCallback((handler: (msg: ServerMessage) => void) => {
    handlerRef.current = handler
  }, [])

  return { connected, lastMessage, send, onMessage }
}
