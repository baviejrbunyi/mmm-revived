package account

import (
	"maps"
	"slices"
	"time"

	"mmm-revived/internal/calendar"
	"mmm-revived/internal/model"
)

func (a *Account) journeyLoginUnlocked(rule JourneyActivityRule, now int64) bool {
	return rule.LoginDay > 0 && rule.LoginLevel > 0 && a.Level >= rule.LoginLevel && LogicDay(now)-LogicDay(a.CreatedAt)+1 >= int64(rule.LoginDay)
}

func journeyActivityDay(rule JourneyActivityRule, now, offset int64) int64 {
	return floorDays(now+offset-calendar.RefreshSeconds) - floorDays(rule.Start-calendar.RefreshSeconds) + 1
}

func floorDays(seconds int64) int64 {
	days := seconds / calendar.DaySeconds
	if seconds%calendar.DaySeconds != 0 && seconds < 0 {
		days--
	}
	return days
}

func journeyWeek(now int64, weekday int) int64 {
	day := LogicDay(now) + 3 - int64(max(weekday, 1)-1)
	week := day / 7
	if day%7 != 0 && day < 0 {
		week--
	}
	return week
}

func (rule JourneyActivityRule) EventTasks() [][]model.TaskID {
	lists := [][]model.TaskID{rule.Tasks, rule.Permanent}
	for _, condition := range rule.Conditions {
		lists = append(lists, condition.Tasks)
	}
	return lists
}

func (a *Account) refreshJourneyEvent(r *Rules, rule JourneyActivityRule, activity *model.JourneyActivity, now int64) {
	a.ensureTasks(rule.EventTasks()...)
	day := LogicDay(now)
	switch rule.Type {
	case 12:
		if activity.DailyDay != day {
			activity.DailyUsed = 0
		}
	case 25:
		a.refreshCommonActive(r, rule, activity, now)
	case 28:
		calendarNow := now + r.Journey.CalendarOffset
		for _, id := range rule.Tasks {
			task := r.Tasks[id]
			if task.Type == 122 && len(task.Params) > 0 && calendarNow >= int64(task.Params[0]) {
				a.UpdateTask(id, 1, task.Target)
			}
		}
	case 32:
		if week := journeyWeek(now, rule.Parameter); activity.Week != week {
			first := activity.Week == 0
			activity.Week = week
			for _, condition := range rule.Conditions {
				for _, id := range condition.Tasks {
					if !first && !slices.Contains(rule.Permanent, id) {
						a.Tasks[id] = model.Task{Status: 2}
					}
				}
			}
		}
	}
	activity.DailyDay = day
}

func (a *Account) refreshCommonActive(r *Rules, rule JourneyActivityRule, activity *model.JourneyActivity, now int64) {
	day, week := LogicDay(now), LogicWeek(now)
	if activity.TaskMarks == nil {
		activity.TaskMarks = map[model.TaskID]int64{}
	}
	if activity.Week != week {
		activity.Week = week
		activity.WeekCoins = 0
	}
	newDay := activity.DailyDay != day
	activity.DailyDay = day
	for _, condition := range rule.Conditions {
		for _, id := range condition.Tasks {
			task := a.Tasks[id]
			if task.Status < 4 {
				continue
			}
			if activity.TaskMarks[id] == 0 {
				if coins, err := rewardItems(r, r.Tasks[id].Reward, 1); err == nil {
					activity.WeekCoins += coins[rule.Coin]
				}
				activity.TaskMarks[id] = day
			}
			repeatable := !condition.Once && !condition.Daily && (rule.WeeklyCap <= 0 || activity.WeekCoins < int64(rule.WeeklyCap))
			if condition.Daily && newDay || repeatable {
				a.Tasks[id] = model.Task{Status: 2}
				delete(activity.TaskMarks, id)
			}
		}
	}
}

func (a *Account) journeyEventAutoClaim(r *Rules, id model.TaskID, now int64) bool {
	for activityID, state := range a.Journey.Activities {
		rule := r.Journey.Activities[activityID]
		if state.Open && rule.Type == 7 && slices.Contains(rule.Tasks, id) && a.journeyActivityAvailable(rule, now, r.Journey.CalendarOffset) {
			return true
		}
	}
	return false
}

func (a *Account) doubleMaterialGroup(r *Rules, group int) bool {
	for _, rule := range r.Journey.Activities {
		if rule.Type != 12 {
			continue
		}
		for _, kind := range rule.Rotation {
			if slices.Contains(r.Journey.ChallengeTypes[kind], group) {
				return true
			}
		}
	}
	return false
}

func (a *Account) doubleMaterialEvent(r *Rules, now int64) (model.ActivityID, int, int) {
	for _, id := range slices.Sorted(maps.Keys(r.Journey.Activities)) {
		rule := r.Journey.Activities[id]
		state, known := a.Journey.Activities[id]
		if rule.Type != 12 || len(rule.Rotation) == 0 || !known || !state.Open || !a.journeyActivityAvailable(rule, now, r.Journey.CalendarOffset) {
			continue
		}
		day := journeyActivityDay(rule, now, r.Journey.CalendarOffset)
		if day < 1 {
			continue
		}
		used := state.DailyUsed
		if state.DailyDay != LogicDay(now) {
			used = 0
		}
		return id, rule.Rotation[(day-1)%int64(len(rule.Rotation))], max(0, rule.Parameter-used)
	}
	return 0, 0, 0
}

func (a *Account) doubleMaterialChances(r *Rules, group, count int, now int64) (model.ActivityID, error) {
	id, kind, remaining := a.doubleMaterialEvent(r, now)
	if id == 0 || !slices.Contains(r.Journey.ChallengeTypes[kind], group) {
		return 0, rejected("double material challenge is not open today")
	}
	if remaining < count {
		return 0, rejected("double material chances are used up")
	}
	return id, nil
}

func (a *Account) consumeDoubleMaterial(r *Rules, group, count int, quantities map[model.ItemID]int64) (map[model.ItemID]int64, error) {
	if !a.doubleMaterialGroup(r, group) {
		return quantities, nil
	}
	now := time.Now().Unix()
	id, err := a.doubleMaterialChances(r, group, count, now)
	if err != nil {
		return nil, err
	}
	doubled, err := multiplied(quantities, r.Settings.DoubleMaterial)
	if err != nil {
		return nil, err
	}
	activity := a.Journey.Activities[id]
	if day := LogicDay(now); activity.DailyDay != day {
		activity.DailyDay = day
		activity.DailyUsed = 0
	}
	activity.DailyUsed += count
	a.Journey.Activities[id] = activity
	return doubled, nil
}

func (a *Account) DoubleMaterialSnapshot(r *Rules, challenges *model.ChallengeComponent) {
	challenges.Double = map[int]int{}
	if r == nil {
		return
	}
	if id, kind, remaining := a.doubleMaterialEvent(r, time.Now().Unix()); id != 0 {
		challenges.Double[kind] = remaining
	}
}
