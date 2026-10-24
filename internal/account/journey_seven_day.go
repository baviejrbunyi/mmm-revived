package account

import (
	"slices"

	"mmm-revived/internal/model"
)

func (rule JourneySevenDayRule) Marks(id model.TaskID) int64 {
	if rule.Mark == 0 || !slices.Contains(rule.Tasks, id) {
		return 0
	}
	return 1
}

func (a *Account) backfillSevenDayMarks(r *Rules) {
	rule := r.Journey.SevenDay
	if rule.Mark == 0 {
		return
	}
	var claimed int64
	for _, id := range rule.Tasks {
		if a.Tasks[id].Status >= 4 {
			claimed += rule.Marks(id)
		}
	}
	if claimed <= a.Items[rule.Mark] {
		return
	}
	if a.Items == nil {
		a.Items = map[model.ItemID]int64{}
	}
	a.Items[rule.Mark] = claimed
}
