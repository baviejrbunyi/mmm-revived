package season

import (
	"errors"
	"math"

	"mmm-revived/internal/protocol"
)

type EventCount struct {
	ID    int
	Count int
}

func DecodeEventID(c protocol.Call) (int, error) {
	id, err := c.Number(0, 1, math.MaxInt32)
	return int(id), err
}

func DecodeEventCount(c protocol.Call) (EventCount, error) {
	id, err := DecodeEventID(c)
	if err != nil {
		return EventCount{}, err
	}
	count, err := c.Number(1, 1, 100)
	return EventCount{ID: id, Count: int(count)}, err
}

func DecodeDecision(c protocol.Call) (bool, error) {
	if len(c.Args) > 1 {
		return false, errors.New("boss settlement has extra arguments")
	}
	if len(c.Args) == 0 || c.Args[0] == nil {
		return false, nil
	}
	accepted, ok := c.Args[0].(bool)
	if !ok {
		return false, errors.New("boss decision is invalid")
	}
	return accepted, nil
}
