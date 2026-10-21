package protocol

import (
	"math"

	"mmm-revived/internal/model"
)

func DecodeNone(c Call) (struct{}, error) { return struct{}{}, c.Arguments(0) }

func DecodeIgnore(Call) (struct{}, error) { return struct{}{}, nil }

func DecodePlayer(c Call) (model.UserID, error) { return c.Player(1) }

func DecodeActivity(c Call) (model.ActivityID, error) {
	id, err := c.Number(0, 1, math.MaxInt32)
	return model.ActivityID(id), err
}
