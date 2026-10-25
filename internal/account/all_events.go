package account

import (
	"cmp"
	"maps"
	"math"
	"slices"

	"mmm-revived/internal/calendar"
	"mmm-revived/internal/model"
)

type AllEventSettings struct {
	Overlap int64 `json:"overlap"`
	Horizon int64 `json:"horizon"`
}

type EventWindows struct {
	Activities map[model.ActivityID][2]int64
	Shops      map[model.ShopID][2]int64
}

func (s *Store) ConfigureEvents(enabled bool) {
	if s.Rules == nil {
		return
	}
	s.Rules.AllEvents = EventWindows{}
	if enabled {
		s.Rules.openAllEvents()
	}
}

func (r *Rules) openAllEvents() {
	original := maps.Clone(r.Journey.Activities)
	earliest, latest := int64(math.MaxInt64), int64(0)
	kinds := map[int][]model.ActivityID{}
	for id, rule := range original {
		if rule.End > 0 {
			earliest = min(earliest, rule.Start)
			latest = max(latest, rule.End)
			kinds[rule.Type] = append(kinds[rule.Type], id)
		}
	}
	if latest == 0 {
		return
	}
	first, period := calendar.Loop()
	start := earliest
	if period > 0 {
		start = first
	}
	end := max(latest, first+period) + r.Settings.AllEvents.Horizon
	windows := map[model.ActivityID][2]int64{}
	for kind, timed := range kinds {
		if r.concurrentEvents(original, timed) {
			for _, id := range timed {
				windows[id] = [2]int64{start, end}
			}
			continue
		}
		var sequence []model.ActivityID
		for id, rule := range original {
			if rule.Type == kind && rule.Start < end {
				sequence = append(sequence, id)
			}
		}
		slices.SortFunc(sequence, func(a, b model.ActivityID) int {
			return cmp.Or(cmp.Compare(original[a].Start, original[b].Start), cmp.Compare(a, b))
		})
		for index, id := range sequence {
			rule := original[id]
			window := [2]int64{rule.Start, end}
			if index == 0 {
				window[0] = min(rule.Start, start)
			}
			if index+1 < len(sequence) {
				window[1] = original[sequence[index+1]].Start - 1
			} else if rule.End <= 0 {
				window[1] = 0
			}
			windows[id] = window
		}
	}
	for id, window := range windows {
		r.Journey.Activities[id] = withWindow(r.Journey.Activities[id], window)
		if rule, known := r.RetailPVP.Activities[id]; known {
			r.RetailPVP.Activities[id] = withWindow(rule, window)
		}
		if rule, known := r.RetailActivity.Activities[id]; known {
			r.RetailActivity.Activities[id] = withWindow(rule, window)
		}
	}
	shops := map[model.ShopID][2]int64{}
	for id, rule := range r.Shops {
		if window, tied := eventShopWindow(original, windows, rule.Start, rule.End); tied {
			rule.Start, rule.End = window[0], window[1]
			r.Shops[id], shops[id] = rule, window
		}
	}
	for id, rule := range r.System.Shops {
		if window, tied := eventShopWindow(original, windows, rule.Start, rule.End); tied {
			rule.Start, rule.End = window[0], window[1]
			r.System.Shops[id], shops[id] = rule, window
		}
	}
	r.AllEvents = EventWindows{Activities: windows, Shops: shops}
}

func (r *Rules) concurrentEvents(rules map[model.ActivityID]JourneyActivityRule, ids []model.ActivityID) bool {
	for index, id := range ids {
		for _, other := range ids[index+1:] {
			if min(rules[id].End, rules[other].End)-max(rules[id].Start, rules[other].Start) > r.Settings.AllEvents.Overlap {
				return true
			}
		}
	}
	return false
}

func withWindow(rule JourneyActivityRule, window [2]int64) JourneyActivityRule {
	rule.Start, rule.End = window[0], window[1]
	return rule
}

func eventShopWindow(rules map[model.ActivityID]JourneyActivityRule, windows map[model.ActivityID][2]int64, start, end int64) ([2]int64, bool) {
	if end <= 0 {
		return [2]int64{}, false
	}
	var best model.ActivityID
	var score [3]int64
	for id := range windows {
		rule := rules[id]
		if rule.End <= 0 || start > rule.End || max(end-rule.End, rule.End-end) > DaySeconds {
			continue
		}
		candidate := [3]int64{max(start-rule.Start, rule.Start-start), max(end-rule.End, rule.End-end), int64(id)}
		if best == 0 || slices.Compare(candidate[:], score[:]) < 0 {
			best, score = id, candidate
		}
	}
	if best == 0 {
		return [2]int64{}, false
	}
	window := windows[best]
	return [2]int64{min(start, window[0]), window[1]}, true
}

func (r *Rules) EventWindowSnapshot(activities *model.ActivityComponent) {
	if r == nil || len(r.AllEvents.Activities) == 0 {
		return
	}
	activities.EventStart = make(map[model.ActivityID]int64, len(r.AllEvents.Activities))
	activities.EventEnd = make(map[model.ActivityID]int64, len(r.AllEvents.Activities))
	for id, window := range r.AllEvents.Activities {
		activities.EventStart[id], activities.EventEnd[id] = window[0], window[1]
	}
	activities.ShopStart = make(map[model.ShopID]int64, len(r.AllEvents.Shops))
	activities.ShopEnd = make(map[model.ShopID]int64, len(r.AllEvents.Shops))
	for id, window := range r.AllEvents.Shops {
		activities.ShopStart[id], activities.ShopEnd[id] = window[0], window[1]
	}
}
