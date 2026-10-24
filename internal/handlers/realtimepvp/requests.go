package realtimepvp

import (
	"errors"
	"math"

	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type ActivityTasks struct {
	Activity model.ActivityID
	Tasks    []model.TaskID
}

func DecodeActivityTasks(c protocol.Call) (ActivityTasks, error) {
	activity, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return ActivityTasks{}, err
	}
	ids, err := c.Integers(1, 100, 1, math.MaxInt32)
	r := ActivityTasks{Activity: model.ActivityID(activity)}
	for _, id := range ids {
		r.Tasks = append(r.Tasks, model.TaskID(id))
	}
	return r, err
}

type MatchStart struct {
	Activity model.ActivityID
	Version  int
}

type ConfirmSelection struct {
	Round  int
	Heroes []model.Selection
}

type PVPChoice struct {
	Activity model.ActivityID
	Index    int
}

func DecodeMatchStart(c protocol.Call) (MatchStart, error) {
	activity, err := protocol.DecodeActivity(c)
	if err != nil {
		return MatchStart{}, err
	}
	version, err := c.Number(1, 0, math.MaxInt32)
	return MatchStart{Activity: activity, Version: int(version)}, err
}

func DecodePVPChoice(c protocol.Call) (PVPChoice, error) {
	activity, err := protocol.DecodeActivity(c)
	if err != nil {
		return PVPChoice{}, err
	}
	index, err := c.Number(1, 0, 100)
	return PVPChoice{Activity: activity, Index: int(index)}, err
}

func DecodeSelection(c protocol.Call) (ConfirmSelection, error) {
	round, err := c.Number(0, 1, 10)
	if err != nil {
		return ConfirmSelection{}, err
	}
	if len(c.Args) != 2 {
		return ConfirmSelection{}, errors.New("selection requires a hero list")
	}
	list, ok := c.Args[1].([]any)
	if !ok || len(list) == 0 || len(list) > 3 {
		return ConfirmSelection{}, errors.New("invalid selection size")
	}
	r := ConfirmSelection{Round: int(round), Heroes: make([]model.Selection, 0, len(list))}
	heroes, positions := make(map[model.HeroID]bool, len(list)), make(map[int]bool, len(list))
	for _, value := range list {
		var selected model.Selection
		if err := (protocol.Call{Name: c.Name, Args: []any{value}}).Record(0, &selected); err != nil {
			return ConfirmSelection{}, err
		}
		if selected.Hero <= 0 || selected.Born < 0 || selected.Born > 65535 || heroes[selected.Hero] || positions[selected.Born] {
			return ConfirmSelection{}, errors.New("invalid hero selection")
		}
		heroes[selected.Hero], positions[selected.Born] = true, true
		r.Heroes = append(r.Heroes, selected)
	}
	return r, nil
}

func DecodeMatchReport(c protocol.Call) (model.MatchReport, error) {
	uid, err := c.Number(0, 1, math.MaxInt64)
	if err != nil {
		return model.MatchReport{}, err
	}
	if len(c.Args) > 1 && protocol.EmptyTable(c.Args[1]) {
		return model.MatchReport{Player: model.UserID(uid)}, nil
	}
	var report struct {
		GID      string `json:"gid"`
		Winner   int64  `json:"winner"`
		Opponent int64  `json:"oppo"`
		Frame    int    `json:"frame"`
	}
	if err = c.Record(1, &report); err != nil {
		return model.MatchReport{}, err
	}
	if len(report.GID) == 0 || len(report.GID) > 128 || report.Frame < 0 || report.Frame > 18000 {
		return model.MatchReport{}, errors.New("invalid battle report")
	}
	return model.MatchReport{Player: model.UserID(uid), GID: report.GID, Winner: model.UserID(report.Winner), Opponent: model.UserID(report.Opponent), Frame: report.Frame}, nil
}

type PVPHeroPlacement struct {
	Round int
	Hero  model.HeroID
	Born  int
}

func DecodePVPHeroLeave(c protocol.Call) (PVPHeroPlacement, error) {
	if len(c.Args) != 2 {
		return PVPHeroPlacement{}, errors.New("heroLeaveMap requires round and hero")
	}
	return decodePVPHeroPlacement(c)
}

func DecodePVPHeroPosition(c protocol.Call) (PVPHeroPlacement, error) {
	if len(c.Args) != 3 {
		return PVPHeroPlacement{}, errors.New("heroChangePos requires round, hero and position")
	}
	r, err := decodePVPHeroPlacement(c)
	if err != nil {
		return PVPHeroPlacement{}, err
	}
	born, err := c.Number(2, 0, 65535)
	r.Born = int(born)
	return r, err
}

func decodePVPHeroPlacement(c protocol.Call) (PVPHeroPlacement, error) {
	round, err := c.Number(0, 1, 10)
	if err != nil {
		return PVPHeroPlacement{}, err
	}
	hero, err := c.Number(1, 1, math.MaxInt32)
	return PVPHeroPlacement{Round: int(round), Hero: model.HeroID(hero)}, err
}

const duelFieldLimit = 32

type DuelOperation struct {
	Operation int64
	Arguments map[any]any
}

func DecodeDuelOperation(c protocol.Call) (DuelOperation, error) {
	if err := c.Arguments(2); err != nil {
		return DuelOperation{}, err
	}
	operation, err := c.Number(0, 1, 1)
	if err != nil {
		return DuelOperation{}, err
	}
	input, ok := c.Args[1].(map[any]any)
	if !ok {
		return DuelOperation{}, errors.New("invalid duel operation arguments")
	}
	arguments := map[any]any{}
	for _, key := range [...]string{"coid", "slot", "toid", "choice"} {
		value, present := input[key]
		if !present || value == nil {
			continue
		}
		switch field := value.(type) {
		case string:
			if field == "" || len(field) > duelFieldLimit || !protocol.ValidText(field) {
				return DuelOperation{}, errors.New("invalid duel operation field " + key)
			}
			arguments[key] = field
		default:
			number, err := protocol.Integer(value)
			if err != nil || number < -(1<<31) || number >= 1<<31 {
				return DuelOperation{}, errors.New("invalid duel operation field " + key)
			}
			arguments[key] = number
		}
	}
	if _, ok := arguments["coid"]; !ok {
		return DuelOperation{}, errors.New("duel operation requires a card")
	}
	return DuelOperation{Operation: operation, Arguments: arguments}, nil
}
