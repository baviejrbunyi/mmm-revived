package multipvp

import (
	"errors"
	"math"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type Revenge struct {
	Activity model.ActivityID
	Index    int
}

type MineSelection struct {
	Activity model.ActivityID
	Index    int
	Mine     int
}

type Defense struct {
	Activity model.ActivityID
	Index    int
	Heroes   map[model.HeroID]int
}

func DecodeRevenge(c protocol.Call) (Revenge, error) {
	if err := c.Arguments(2); err != nil {
		return Revenge{}, err
	}
	activity, err := protocol.DecodeActivity(c)
	if err != nil {
		return Revenge{}, err
	}
	index, err := c.Number(1, 0, 99)
	return Revenge{Activity: activity, Index: int(index)}, err
}

func DecodeMineSelection(c protocol.Call) (MineSelection, error) {
	if err := c.Arguments(3); err != nil {
		return MineSelection{}, err
	}
	activity, err := protocol.DecodeActivity(c)
	if err != nil {
		return MineSelection{}, err
	}
	index, err := c.Number(1, 0, 2)
	if err != nil {
		return MineSelection{}, err
	}
	mine, err := c.Number(2, 1, math.MaxInt32)
	return MineSelection{Activity: activity, Index: int(index), Mine: int(mine)}, err
}

func DecodeDefense(c protocol.Call) (Defense, error) {
	if err := c.Arguments(3); err != nil {
		return Defense{}, err
	}
	activity, err := protocol.DecodeActivity(c)
	if err != nil {
		return Defense{}, err
	}
	index, err := c.Number(1, 0, 2)
	if err != nil {
		return Defense{}, err
	}
	heroes := map[model.HeroID]int{}
	if protocol.EmptyTable(c.Args[2]) {
		return Defense{Activity: activity, Index: int(index), Heroes: heroes}, nil
	}
	values, okay := c.Args[2].(map[any]any)
	if !okay || len(values) > 64 {
		return Defense{}, errors.New("invalid Halloween defense dictionary")
	}
	for key, value := range values {
		hero, err := protocol.Integer(key)
		if err != nil || hero < 1 || hero > math.MaxInt32 {
			return Defense{}, errors.New("invalid Halloween defense hero")
		}
		born, err := protocol.Integer(value)
		if err != nil || born < 0 || born > 65535 {
			return Defense{}, errors.New("invalid Halloween defense position")
		}
		heroes[model.HeroID(hero)] = int(born)
	}
	return Defense{Activity: activity, Index: int(index), Heroes: heroes}, nil
}
