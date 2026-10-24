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
	{ID: 1, Prepare: 1760475600, Battle: 1761098400, End: 1762290000, Changes: [3]int64{3, 7, 10}},
	{ID: 2, Prepare: 1762290000, Battle: 1762912800, End: 1764104400, Changes: [3]int64{3, 7, 10}},
	{ID: 3, Prepare: 1764882000, Battle: 1765504800, End: 1766696400, Changes: [3]int64{3, 7, 10}},
	{ID: 4, Prepare: 1766696400, Battle: 1767319200, End: 1768510800, Changes: [3]int64{3, 7, 10}},
	{ID: 5, Prepare: 1768510800, Battle: 1769133600, End: 1770325200, Changes: [3]int64{3, 7, 10}},
	{ID: 6, Prepare: 1770325200, Battle: 1770948000, End: 1772139600, Changes: [3]int64{3, 7, 10}},
	{ID: 7, Prepare: 1772139600, Battle: 1772762400, End: 1773954000, Changes: [3]int64{3, 7, 10}},
	{ID: 8, Prepare: 1773954000, Battle: 1774576800, End: 1775768400, Changes: [3]int64{3, 7, 10}},
	{ID: 9, Prepare: 1775768400, Battle: 1776391200, End: 1777582800, Changes: [3]int64{3, 7, 10}},
	{ID: 10, Prepare: 1777582800, Battle: 1778205600, End: 1779397200, Changes: [3]int64{3, 7, 10}},
	{ID: 11, Prepare: 1779397200, Battle: 1780020000, End: 1781211600, Changes: [3]int64{3, 7, 10}},
	{ID: 12, Prepare: 1781211600, Battle: 1781834400, End: 1783026000, Changes: [3]int64{3, 7, 10}},
	{ID: 13, Prepare: 1783026000, Battle: 1783648800, End: 1784840400, Changes: [3]int64{3, 7, 10}},
	{ID: 14, Prepare: 1784840400, Battle: 1785463200, End: 1786654800, Changes: [3]int64{3, 7, 10}},
	{ID: 15, Prepare: 1786654800, Battle: 1787277600, End: 1788469200, Changes: [3]int64{3, 7, 10}},
	{ID: 16, Prepare: 1788469200, Battle: 1789092000, End: 1790283600, Changes: [3]int64{3, 7, 10}},
	{ID: 17, Prepare: 1790283600, Battle: 1790906400, End: 1792098000, Changes: [3]int64{3, 7, 10}},
	{ID: 18, Prepare: 1792098000, Battle: 1792720800, End: 1793912400, Changes: [3]int64{3, 7, 10}},
	{ID: 19, Prepare: 1793912400, Battle: 1794535200, End: 1795726800, Changes: [3]int64{3, 7, 10}},
	{ID: 20, Prepare: 1795726800, Battle: 1796349600, End: 1797541200, Changes: [3]int64{3, 7, 10}},
	{ID: 21, Prepare: 1797541200, Battle: 1798164000, End: 1799355600, Changes: [3]int64{3, 7, 10}},
}

func CurrentWar(now int64) (int, int) {
	t := calendar.ContentTime(now)
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
