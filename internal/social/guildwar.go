package social

import "mmm-revived/internal/calendar"

const (
	WarEmpty = iota
	WarPrepare
	WarBattle1
	WarBattle2
	WarBattle3
	WarBattle4
	WarEnd
)

type WarPeriod struct {
	ID      int
	Prepare int64
	Battle  int64
	End     int64
	Changes []int64
}

func (r *Repository) CurrentWar(now int64) (int, int) {
	t := calendar.ContentTime(now)
	for _, period := range r.wars {
		if t < period.Prepare || t >= period.End {
			continue
		}
		if t < period.Battle {
			return period.ID, WarPrepare
		}
		phase := WarBattle1
		for _, day := range period.Changes {
			if t >= period.Battle+day*daySeconds {
				phase++
			}
		}
		return period.ID, phase
	}
	return 0, WarEmpty
}

func WarBattling(phase int) bool { return phase >= WarBattle1 && phase <= WarBattle4 }
