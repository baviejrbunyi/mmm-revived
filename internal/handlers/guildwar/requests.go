package guildwar

import (
	"math"

	"mmm-revived/internal/protocol"
)

type WarOrder struct {
	Areas   []int
	Blocked int
}

type WarLevel struct {
	Area  int
	Level int
}

func DecodeWarOrder(c protocol.Call) (WarOrder, error) {
	areas, err := c.Integers(0, 4, 1, math.MaxInt32)
	if err != nil {
		return WarOrder{}, err
	}
	blocked, err := c.Number(1, 1, math.MaxInt32)
	return WarOrder{Areas: areas, Blocked: int(blocked)}, err
}

func DecodeWarLevel(c protocol.Call) (WarLevel, error) {
	area, err := c.Number(0, 0, math.MaxInt32)
	if err != nil {
		return WarLevel{}, err
	}
	level, err := c.Number(1, 1, math.MaxInt32)
	return WarLevel{Area: int(area), Level: int(level)}, err
}
