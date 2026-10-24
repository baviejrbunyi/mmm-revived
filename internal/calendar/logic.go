package calendar

import "time"

const (
	DaySeconds     int64 = 86400
	RefreshSeconds int64 = 18000
	epochWeekday   int64 = 3
)

func LogicDay(now int64) int64 {
	return floorDiv(Monotonic(now)-RefreshSeconds, DaySeconds)
}

func LogicWeek(now int64) int64 {
	return floorDiv(LogicDay(now)+epochWeekday, 7)
}

func LogicWeekday(now int64) int {
	day := LogicDay(now) + epochWeekday
	return int(day-floorDiv(day, 7)*7) + 1
}

func LogicMonth(now int64) int64 {
	date := time.Unix(LogicDay(now)*DaySeconds, 0).UTC()
	return int64(date.Year())*12 + int64(date.Month())
}

func floorDiv(value, divisor int64) int64 {
	quotient := value / divisor
	if value%divisor != 0 && value < 0 {
		quotient--
	}
	return quotient
}
