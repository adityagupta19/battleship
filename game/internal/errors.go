package internal

import "errors"

var (
	ErrNotInGame      = errors.New("user is not in this game")
	ErrInvalidStatus  = errors.New("invalid game status for this action")
	ErrNotYourTurn    = errors.New("not your turn")
	ErrAlreadyShot    = errors.New("already fired at this cell")
	ErrOutOfBounds    = errors.New("coordinates out of bounds")
	ErrInvalidFleet   = errors.New("invalid ship placement")
	ErrAlreadyReady   = errors.New("ships already placed")
)
