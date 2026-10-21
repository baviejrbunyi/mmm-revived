package draw

import (
	"math"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type Draw struct {
	Pool  model.DrawPoolID
	Count int
}

func DecodeDraw(c protocol.Call) (Draw, error) {
	id, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return Draw{}, err
	}
	count, err := c.Number(1, 1, 10)
	return Draw{Pool: model.DrawPoolID(id), Count: int(count)}, err
}
