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

type warPeriod struct {
	ID      int
	Prepare int64
	Battle  int64
	End     int64
	Changes [3]int64
}

var warSchedule = []warPeriod{
	{ID: 1, Prepare: 1760475600, Battle: 1761080400, End: 1762290000, Changes: [3]int64{3, 7, 10}},
	{ID: 2, Prepare: 1762290000, Battle: 1762894800, End: 1764104400, Changes: [3]int64{3, 7, 10}},
	{ID: 3, Prepare: 1764104400, Battle: 1764709200, End: 1765918800, Changes: [3]int64{3, 7, 10}},
}

func CurrentWar(now int64) (int, int) {
	t := calendar.ClientTime(now)
	for _, period := range warSchedule {
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
