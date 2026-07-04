package internal

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
)

const (
	StatusPlacement = "placement"
	StatusActive    = "active"
	StatusFinished  = "finished"
)

type Coord struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Ship struct {
	Type      string  `json:"type"`
	Positions []Coord `json:"positions"`
	Hits      int     `json:"hits"`
	Sunk      bool    `json:"sunk"`
}

type PlayerState struct {
	Ships          []Ship  `json:"ships"`
	ShotsReceived  []Coord `json:"shots_received"`
	Ready          bool    `json:"ready"`
}

type ShotRecord struct {
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Shooter uint   `json:"shooter"`
	Result  string `json:"result"`
}

type GameState struct {
	Player1 PlayerState  `json:"player1"`
	Player2 PlayerState  `json:"player2"`
	Shots   []ShotRecord `json:"shots"`
}

type Game struct {
	ID          uint64         `gorm:"primaryKey"`
	Player1ID   uint           `gorm:"column:player1_id"`
	Player2ID   uint           `gorm:"column:player2_id"`
	Status      string         `gorm:"column:status"`
	CurrentTurn uint           `gorm:"column:current_turn"`
	WinnerID    *uint          `gorm:"column:winner_id"`
	State       datatypes.JSON `gorm:"column:state;type:jsonb"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
}

func (Game) TableName() string { return "games" }

func emptyGameState() GameState {
	return GameState{
		Player1: PlayerState{Ships: []Ship{}, ShotsReceived: []Coord{}},
		Player2: PlayerState{Ships: []Ship{}, ShotsReceived: []Coord{}},
		Shots:   []ShotRecord{},
	}
}

func (g *Game) ParsedState() GameState {
	if len(g.State) == 0 {
		return emptyGameState()
	}
	var s GameState
	_ = json.Unmarshal(g.State, &s)
	return s
}

func (g *Game) saveState(s GameState) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	g.State = datatypes.JSON(data)
	return nil
}

func playerSlot(g *Game, userID uint) (self *PlayerState, opponent *PlayerState, isP1 bool, err error) {
	s := g.ParsedState()
	if uint(g.Player1ID) == userID {
		return &s.Player1, &s.Player2, true, nil
	}
	if uint(g.Player2ID) == userID {
		return &s.Player2, &s.Player1, false, nil
	}
	return nil, nil, false, ErrNotInGame
}
