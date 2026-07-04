package internal

const BoardSize = 10

// Cell values for grid display.
const (
	CellWater = 0
	CellShip  = 1
	CellMiss  = 2
	CellHit   = 3
)

// Standard fleet ship types and lengths.
var FleetSpec = map[string]int{
	"carrier":    5,
	"battleship": 4,
	"cruiser":    3,
	"submarine":  3,
	"destroyer":  2,
}
