package rogue

import (
	"errors"
	"fmt"
	"math"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type Pair struct{ First, Second int }

func DecodeIndex(c protocol.Call) (int, error) {
	if err := c.Arguments(1); err != nil {
		return 0, err
	}
	value, err := c.Number(0, 0, math.MaxInt32)
	return int(value), err
}

func DecodePair(c protocol.Call) (Pair, error) {
	if err := c.Arguments(2); err != nil {
		return Pair{}, err
	}
	first, err := c.Number(0, 0, math.MaxInt32)
	if err != nil {
		return Pair{}, err
	}
	second, err := c.Number(1, 0, math.MaxInt32)
	return Pair{First: int(first), Second: int(second)}, err
}

func DecodeChestOpen(c protocol.Call) (Pair, error) {
	if len(c.Args) != 1 && len(c.Args) != 2 {
		return Pair{}, fmt.Errorf("invalid %s argument count", c.Name)
	}
	count, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return Pair{}, err
	}
	if len(c.Args) == 1 {
		return Pair{First: int(count), Second: -1}, nil
	}
	index, err := c.Number(1, 0, math.MaxInt32)
	return Pair{First: int(count), Second: int(index)}, err
}

func rogueList(c protocol.Call) ([]int, error) {
	if len(c.Args) != 1 {
		return nil, errors.New("invalid rogue list arguments")
	}
	values, err := c.Integers(0, 20, 1, math.MaxInt32)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, errors.New("rogue list is empty")
	}
	return values, nil
}

func DecodeRogueTeam(c protocol.Call) ([]model.HeroID, error) {
	values, err := rogueList(c)
	if err != nil {
		return nil, err
	}
	if len(values) > 3 {
		return nil, errors.New("rogue team is too large")
	}
	heroes := make([]model.HeroID, len(values))
	for i, id := range values {
		heroes[i] = model.HeroID(id)
	}
	return heroes, nil
}

func DecodeRogueBuffs(c protocol.Call) ([]int, error) { return rogueList(c) }
