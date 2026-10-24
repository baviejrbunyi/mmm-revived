package dispatch

import (
	"math"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type DispatchStart struct {
	Mission model.DispatchID
	Heroes  []model.HeroID
}

func DecodeDispatch(c protocol.Call) (model.DispatchID, error) {
	id, err := c.InstanceID(0)
	return model.DispatchID(id), err
}

func DecodeDispatches(c protocol.Call) ([]model.DispatchID, error) {
	ids, err := c.InstanceIDs(0, 20, false)
	missions := make([]model.DispatchID, 0, len(ids))
	for _, id := range ids {
		missions = append(missions, model.DispatchID(id))
	}
	return missions, err
}

func DecodeDispatchStart(c protocol.Call) (DispatchStart, error) {
	mission, err := DecodeDispatch(c)
	if err != nil {
		return DispatchStart{}, err
	}
	heroes, err := c.Integers(1, 10, 1, math.MaxInt32)
	r := DispatchStart{Mission: mission, Heroes: make([]model.HeroID, 0, len(heroes))}
	for _, id := range heroes {
		r.Heroes = append(r.Heroes, model.HeroID(id))
	}
	return r, err
}

func DecodeMonsters(c protocol.Call) ([]int, error) {
	if len(c.Args) == 0 {
		return nil, nil
	}
	list, _ := c.Args[0].([]any)
	ids := make([]int, 0, min(len(list), 200))
	for _, value := range list[:min(len(list), 200)] {
		if n, err := protocol.Integer(value); err == nil && n > 0 && n <= math.MaxInt32 {
			ids = append(ids, int(n))
		}
	}
	return ids, nil
}
