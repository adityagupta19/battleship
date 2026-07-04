package internal

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/adityagupta19/battleship/game/db"
	"gorm.io/datatypes"
)

type ShipPlacementInput struct {
	Type        string
	StartX      int
	StartY      int
	Horizontal  bool
}

func CreateGameFromPair(gameID uint64, player1, player2 uint) error {
	database := db.GetDB()

	var existing Game
	if err := database.First(&existing, gameID).Error; err == nil {
		return nil
	}

	first := player1
	if rand.Intn(2) == 1 {
		first = player2
	}

	state := emptyGameState()
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	game := Game{
		ID:          gameID,
		Player1ID:   player1,
		Player2ID:   player2,
		Status:      StatusPlacement,
		CurrentTurn: first,
		State:       datatypes.JSON(data),
		CreatedAt:   time.Now(),
	}
	return database.Create(&game).Error
}

func PlaceShips(gameID uint64, userID uint, placements []ShipPlacementInput) (*Game, error) {
	database := db.GetDB()
	var game Game
	if err := database.First(&game, gameID).Error; err != nil {
		return nil, fmt.Errorf("game not found")
	}
	if game.Status != StatusPlacement {
		return nil, ErrInvalidStatus
	}

	state := game.ParsedState()
	self, _, isP1, err := playerSlot(&game, userID)
	if err != nil {
		return nil, err
	}
	if self.Ready {
		return nil, ErrAlreadyReady
	}

	ships, err := buildShips(placements)
	if err != nil {
		return nil, err
	}
	self.Ships = ships
	self.Ready = true

	if isP1 {
		state.Player1 = *self
	} else {
		state.Player2 = *self
	}

	if state.Player1.Ready && state.Player2.Ready {
		game.Status = StatusActive
	}

	if err := game.saveState(state); err != nil {
		return nil, err
	}
	if err := database.Save(&game).Error; err != nil {
		return nil, err
	}
	return &game, nil
}

func buildShips(placements []ShipPlacementInput) ([]Ship, error) {
	if len(placements) != len(FleetSpec) {
		return nil, fmt.Errorf("%w: expected %d ships", ErrInvalidFleet, len(FleetSpec))
	}

	used := map[string]bool{}
	occupied := map[string]bool{}
	var ships []Ship

	for _, p := range placements {
		length, ok := FleetSpec[p.Type]
		if !ok {
			return nil, fmt.Errorf("%w: unknown ship type %s", ErrInvalidFleet, p.Type)
		}
		if used[p.Type] {
			return nil, fmt.Errorf("%w: duplicate ship %s", ErrInvalidFleet, p.Type)
		}
		used[p.Type] = true

		var positions []Coord
		for i := 0; i < length; i++ {
			x, y := p.StartX, p.StartY
			if p.Horizontal {
				y += i
			} else {
				x += i
			}
			if x < 0 || x >= BoardSize || y < 0 || y >= BoardSize {
				return nil, ErrOutOfBounds
			}
			key := fmt.Sprintf("%d,%d", x, y)
			if occupied[key] {
				return nil, fmt.Errorf("%w: overlapping ships", ErrInvalidFleet)
			}
			occupied[key] = true
			positions = append(positions, Coord{X: x, Y: y})
		}
		ships = append(ships, Ship{Type: p.Type, Positions: positions})
	}

	for t := range FleetSpec {
		if !used[t] {
			return nil, fmt.Errorf("%w: missing ship %s", ErrInvalidFleet, t)
		}
	}
	return ships, nil
}

type FireResult struct {
	Result   string
	SunkShip string
	GameOver bool
	WinnerID uint
	NextTurn uint
}

func FireShot(gameID uint64, userID uint, x, y int) (*Game, *FireResult, error) {
	if x < 0 || x >= BoardSize || y < 0 || y >= BoardSize {
		return nil, nil, ErrOutOfBounds
	}

	database := db.GetDB()
	var game Game
	if err := database.First(&game, gameID).Error; err != nil {
		return nil, nil, fmt.Errorf("game not found")
	}
	if game.Status != StatusActive {
		return nil, nil, ErrInvalidStatus
	}
	if uint(game.CurrentTurn) != userID {
		return nil, nil, ErrNotYourTurn
	}

	state := game.ParsedState()
	self, opponent, isP1, err := playerSlot(&game, userID)
	if err != nil {
		return nil, nil, err
	}

	for _, s := range state.Shots {
		if s.Shooter == userID && s.X == x && s.Y == y {
			return nil, nil, ErrAlreadyShot
		}
	}

	hit := markOpponentHit(opponent, x, y)
	result := "miss"
	var sunkShip string
	if hit {
		result = "hit"
		if ship := findShipAt(opponent, x, y); ship != nil && ship.Sunk {
			result = "sunk"
			sunkShip = ship.Type
		}
	}

	state.Shots = append(state.Shots, ShotRecord{X: x, Y: y, Shooter: userID, Result: result})
	opponent.ShotsReceived = append(opponent.ShotsReceived, Coord{X: x, Y: y})

	if isP1 {
		state.Player2 = *opponent
		state.Player1 = *self
	} else {
		state.Player1 = *opponent
		state.Player2 = *self
	}

	fr := &FireResult{Result: result, SunkShip: sunkShip, NextTurn: uint(game.Player1ID)}
	if isP1 {
		fr.NextTurn = uint(game.Player2ID)
	}

	if allSunk(opponent) {
		game.Status = StatusFinished
		w := userID
		game.WinnerID = &w
		fr.GameOver = true
		fr.WinnerID = userID
	} else {
		game.CurrentTurn = fr.NextTurn
	}

	if err := game.saveState(state); err != nil {
		return nil, nil, err
	}
	if err := database.Save(&game).Error; err != nil {
		return nil, nil, err
	}
	return &game, fr, nil
}

func markOpponentHit(opponent *PlayerState, x, y int) bool {
	hit := false
	for i := range opponent.Ships {
		for _, pos := range opponent.Ships[i].Positions {
			if pos.X == x && pos.Y == y {
				opponent.Ships[i].Hits++
				hit = true
				if opponent.Ships[i].Hits >= len(opponent.Ships[i].Positions) {
					opponent.Ships[i].Sunk = true
				}
			}
		}
	}
	return hit
}

func findShipAt(p *PlayerState, x, y int) *Ship {
	for i := range p.Ships {
		for _, pos := range p.Ships[i].Positions {
			if pos.X == x && pos.Y == y {
				return &p.Ships[i]
			}
		}
	}
	return nil
}

func allSunk(p *PlayerState) bool {
	if len(p.Ships) == 0 {
		return false
	}
	for _, s := range p.Ships {
		if !s.Sunk {
			return false
		}
	}
	return true
}

func BuildBoardView(ships []Ship, shotsReceived []Coord, showShips bool) []int32 {
	grid := make([]int32, BoardSize*BoardSize)
	if showShips {
		for _, ship := range ships {
			for _, pos := range ship.Positions {
				idx := pos.Y*BoardSize + pos.X
				if ship.Sunk {
					grid[idx] = CellHit
				} else {
					grid[idx] = CellShip
				}
			}
		}
	}
	for _, shot := range shotsReceived {
		idx := shot.Y*BoardSize + shot.X
		if grid[idx] == CellShip {
			grid[idx] = CellHit
		} else if grid[idx] == 0 {
			grid[idx] = CellMiss
		}
	}
	return grid
}

func BuildShipViews(ships []Ship, showPositions bool) []*ShipView {
	var views []*ShipView
	for _, s := range ships {
		v := &ShipView{Type: s.Type, Hits: int32(s.Hits), Sunk: s.Sunk}
		if showPositions {
			for _, p := range s.Positions {
				v.Positions = append(v.Positions, &CoordView{X: int32(p.X), Y: int32(p.Y)})
			}
		}
		views = append(views, v)
	}
	return views
}

type ShipView struct {
	Type      string
	Positions []*CoordView
	Hits      int32
	Sunk      bool
}

type CoordView struct {
	X, Y int32
}
