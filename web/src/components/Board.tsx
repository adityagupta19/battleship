import type { BoardView, CellValue } from '../types'

const SIZE = 10

interface BoardProps {
  title: string
  board?: BoardView
  clickable?: boolean
  onCellClick?: (x: number, y: number) => void
}

function cellClass(v: number): string {
  switch (v as CellValue) {
    case 1: return 'cell ship'
    case 2: return 'cell miss'
    case 3: return 'cell hit'
    default: return 'cell water'
  }
}

export function Board({ title, board, clickable, onCellClick }: BoardProps) {
  const grid = board?.grid ?? Array(SIZE * SIZE).fill(0)

  return (
    <div className="board-wrap">
      <h3>{title}</h3>
      <div className="board" role="grid">
        {Array.from({ length: SIZE }, (_, y) =>
          Array.from({ length: SIZE }, (_, x) => {
            const idx = y * SIZE + x
            const v = grid[idx] ?? 0
            return (
              <button
                key={`${x}-${y}`}
                type="button"
                className={cellClass(v)}
                disabled={!clickable || v === 2 || v === 3}
                onClick={() => onCellClick?.(x, y)}
                aria-label={`cell ${x},${y}`}
              />
            )
          })
        )}
      </div>
    </div>
  )
}
