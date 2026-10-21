package journey

import (
	"math"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type ActivityLevels struct {
	Activity model.ActivityID
	Levels   []int
	Track    int
}

type ActivityNode struct {
	Activity model.ActivityID
	ID       int
	Node     int
}

type JourneyPair struct {
	ID   int
	Node int
}

func DecodeJourneyID(c protocol.Call) (int, error) {
	if err := c.Arguments(1); err != nil {
		return 0, err
	}
	id, err := c.Number(0, 0, math.MaxInt32)
	return int(id), err
}

func DecodeJourneyPair(c protocol.Call) (JourneyPair, error) {
	if err := c.Arguments(2); err != nil {
		return JourneyPair{}, err
	}
	id, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return JourneyPair{}, err
	}
	node, err := c.Number(1, 1, 100000)
	return JourneyPair{ID: int(id), Node: int(node)}, err
}

func ClaimList(c protocol.Call, index int, minimum int64) ([]int, error) {
	if index < len(c.Args) && protocol.EmptyTable(c.Args[index]) {
		return nil, nil
	}
	return c.Integers(index, 100, minimum, math.MaxInt32)
}

func DecodePassClaim(c protocol.Call) (ActivityLevels, error) {
	if err := c.Arguments(2); err != nil {
		return ActivityLevels{}, err
	}
	levels, err := ClaimList(c, 0, 1)
	if err != nil {
		return ActivityLevels{}, err
	}
	track, err := c.Number(1, -1, 1)
	return ActivityLevels{Levels: levels, Track: int(track)}, err
}

func DecodeLimitedPassClaim(c protocol.Call) (ActivityLevels, error) {
	if err := c.Arguments(3); err != nil {
		return ActivityLevels{}, err
	}
	activity, err := protocol.DecodeActivity(c)
	if err != nil {
		return ActivityLevels{}, err
	}
	levels, err := ClaimList(c, 1, 1)
	if err != nil {
		return ActivityLevels{}, err
	}
	track, err := c.Number(2, -1, 1)
	return ActivityLevels{Activity: activity, Levels: levels, Track: int(track)}, err
}

func decodeActivityList(c protocol.Call, minimum int64) (ActivityLevels, error) {
	if err := c.Arguments(2); err != nil {
		return ActivityLevels{}, err
	}
	activity, err := protocol.DecodeActivity(c)
	if err != nil {
		return ActivityLevels{}, err
	}
	levels, err := ClaimList(c, 1, minimum)
	return ActivityLevels{Activity: activity, Levels: levels}, err
}

func DecodeBrawlList(c protocol.Call) (ActivityLevels, error) { return decodeActivityList(c, 0) }

func DecodeActivityLevels(c protocol.Call) (ActivityLevels, error) { return decodeActivityList(c, 1) }

func DecodeJourneyActivity(c protocol.Call) (model.ActivityID, error) {
	if err := c.Arguments(1); err != nil {
		return 0, err
	}
	return protocol.DecodeActivity(c)
}

func DecodeActivityValue(c protocol.Call) (ActivityNode, error) {
	if err := c.Arguments(2); err != nil {
		return ActivityNode{}, err
	}
	activity, err := protocol.DecodeActivity(c)
	if err != nil {
		return ActivityNode{}, err
	}
	id, err := c.Number(1, 0, math.MaxInt32)
	return ActivityNode{Activity: activity, ID: int(id)}, err
}

func DecodeActivityNode(c protocol.Call) (ActivityNode, error) {
	if err := c.Arguments(3); err != nil {
		return ActivityNode{}, err
	}
	activity, err := protocol.DecodeActivity(c)
	if err != nil {
		return ActivityNode{}, err
	}
	id, err := c.Number(1, 1, math.MaxInt32)
	if err != nil {
		return ActivityNode{}, err
	}
	node, err := c.Number(2, 1, math.MaxInt32)
	return ActivityNode{Activity: activity, ID: int(id), Node: int(node)}, err
}
