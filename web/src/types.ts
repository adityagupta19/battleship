export type CellValue = 0 | 1 | 2 | 3

export interface BoardView {
  grid: number[]
  ships?: { type: string; positions?: { x: number; y: number }[] }[]
}

export interface ShipPlacement {
  type: string
  start_x: number
  start_y: number
  horizontal: boolean
}

export interface ServerMessage {
  type: string
  game_id?: number
  opponent_id?: number
  status?: string
  result?: string
  sunk_ship?: string
  game_over?: boolean
  winner_id?: number
  next_turn?: number
  your_turn?: boolean
  you_ready?: boolean
  opponent_ready?: boolean
  your_board?: BoardView
  opponent_view?: BoardView
  message?: string
}

export type GamePhase = 'lobby' | 'matching' | 'placement' | 'active' | 'finished'

export const DEFAULT_SHIPS: ShipPlacement[] = [
  { type: 'carrier', start_x: 0, start_y: 0, horizontal: true },
  { type: 'battleship', start_x: 2, start_y: 0, horizontal: true },
  { type: 'cruiser', start_x: 4, start_y: 0, horizontal: true },
  { type: 'submarine', start_x: 6, start_y: 0, horizontal: true },
  { type: 'destroyer', start_x: 8, start_y: 0, horizontal: true },
]
