package account

import (
	"time"

	"mmm-revived/internal/model"
)

const LogicDayScheme = 1

func (a *Account) migrateLogicDays(now int64) {
	if a.DayScheme >= LogicDayScheme {
		return
	}
	a.DayScheme = LogicDayScheme
	date := time.Unix(now, 0).UTC()
	legacyDay, legacyWeek, legacyMonth := now/DaySeconds, (now+3*DaySeconds)/(7*DaySeconds), int64(date.Year()*12+int(date.Month()))
	day, week, month := LogicDay(now), LogicWeek(now), LogicMonth(now)
	carry := func(stored *int64, legacy, current int64) {
		if *stored == legacy {
			*stored = current
		}
	}
	for _, stored := range []*int64{&a.Dispatch.Day, &a.Events.Boss.Day, &a.Journey.GuildLoginDay, &a.Network.Day, &a.Network.Support.Day} {
		carry(stored, legacyDay, day)
	}
	for _, stored := range []*int64{&a.Journey.SpentWeek, &a.Journey.Pass.Week, &a.Events.Season.Week, &a.RogueProgress.Week, &a.RogueProgress.PVEWeek} {
		carry(stored, legacyWeek, week)
	}
	for _, activity := range a.Journey.Activities {
		if activity.RetailPVP != nil {
			carry(&activity.RetailPVP.Day, legacyDay, day)
		}
	}
	for id, purchase := range a.ShopPurchases {
		switch purchase.PeriodType {
		case 1:
			carry(&purchase.Period, legacyDay, day)
		case 2:
			carry(&purchase.Period, legacyWeek, week)
		case 3:
			carry(&purchase.Period, legacyMonth, month)
		}
		a.ShopPurchases[id] = purchase
	}
	if a.Journey.GuildTaskWeek == 0 {
		a.Journey.GuildTaskWeek = week
	}
}

func migrateGuildWarDays(state *model.GuildWarState, now int64) {
	if state.DayScheme >= LogicDayScheme {
		return
	}
	state.DayScheme = LogicDayScheme
	legacy, day := now/DaySeconds, LogicDay(now)
	for uid, member := range state.Members {
		if member.Day == legacy {
			member.Day = day
			state.Members[uid] = member
		}
	}
}

func (a *Account) refreshGuildTasks(r *Rules, now int64) {
	week := LogicWeek(now)
	if a.Journey.GuildTaskWeek == week {
		return
	}
	a.Journey.GuildTaskWeek = week
	a.Journey.GuildLoginDays, a.Journey.GuildLoginDay = 0, 0
	if a.Journey.TaskBaselines == nil {
		a.Journey.TaskBaselines = map[model.TaskID]int64{}
	}
	for id, rule := range r.Tasks {
		if rule.Scope == TaskGuild {
			a.Tasks[id] = model.Task{Status: 2}
			a.Journey.TaskBaselines[id] = a.taskCumulativeValue(r, rule)
		}
	}
}
