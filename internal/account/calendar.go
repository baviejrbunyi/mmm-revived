package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"mmm-revived/internal/calendar"
	"mmm-revived/internal/database"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

const (
	calendarCycleMigration = "calendar-cycle-%d"
)

var calendarRotations = [...]int{10, 14}

var calendarRankModes = [...]model.RankMode{model.RankBoss, model.RankInfinite, model.RankAsync, model.RankRetailPVP, model.RankGuildWar}

type CalendarState struct {
	Cycle int64          `json:"cycle"`
	Mail  []CalendarMail `json:"mail,omitempty"`
}

type CalendarMail struct {
	Template int                    `json:"template"`
	Items    map[model.ItemID]int64 `json:"items"`
}

func CalendarLoopWindow(r *Rules) (int64, int64) {
	if r == nil {
		return 0, 0
	}
	var start, end int64
	for _, pass := range r.Journey.Passes {
		if pass.Start <= 0 || pass.End <= pass.Start {
			continue
		}
		if start == 0 || pass.Start < start {
			start = pass.Start
		}
		end = max(end, pass.End+1)
	}
	first := func(begin, candidate int64) int64 {
		if candidate >= start && (begin == 0 || candidate < begin) {
			return candidate
		}
		return begin
	}
	begin := start
	for _, kind := range calendarRotations {
		var rotation int64
		for _, rule := range r.Journey.Activities {
			if rule.Type == kind && rule.End > 0 {
				rotation = first(rotation, rule.Start)
			}
		}
		begin = max(begin, rotation)
	}
	var boss int64
	for _, open := range r.Events.BossOpens {
		boss = first(boss, open.Start)
	}
	if begin = max(begin, boss); end-begin < calendar.WeekSeconds {
		begin = start
	}
	period := end - begin
	return begin, period - period%calendar.WeekSeconds
}

func (s *Store) ConfigureCalendarLoop(enabled bool) {
	if !enabled {
		calendar.SetLoop(0, 0)
		return
	}
	calendar.SetLoop(CalendarLoopWindow(s.Rules))
}

func (s *Store) ConfigureCalendar(requested *int64) (int64, error) {
	ctx := context.Background()
	offset, err := s.Database.Queries.ReadClockOffset(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if requested != nil {
		offset = *requested
	}
	if err = calendar.SetOffset(offset); err != nil {
		return 0, err
	}
	if err = s.Database.Queries.SaveClockOffset(ctx, offset); err != nil {
		return 0, err
	}
	cycle, _ := calendar.Advance(time.Now().Unix())
	s.applyCalendarOffset()
	return offset, s.settleCalendarCycle(cycle)
}

func (s *Store) AdvanceCalendar(now int64) (int64, error) {
	cycle, changed := calendar.Advance(now)
	if !changed {
		return cycle, nil
	}
	s.applyCalendarOffset()
	return cycle, s.settleCalendarCycle(cycle)
}

func (s *Store) applyCalendarOffset() {
	if s.Rules == nil {
		return
	}
	offset := calendar.Offset()
	s.Rules.Draw.CalendarOffset = offset
	s.Rules.Journey.CalendarOffset = offset
	s.Rules.Events.CalendarOffset = offset
}

func (s *Store) settleCalendarCycle(cycle int64) error {
	if cycle <= 0 {
		return nil
	}
	ctx, name := context.Background(), fmt.Sprintf(calendarCycleMigration, cycle)
	if _, err := s.Database.Queries.GetMigration(ctx, name); !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	tx, err := s.Database.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	modes := make([]any, len(calendarRankModes))
	for i, mode := range calendarRankModes {
		modes[i] = string(mode)
	}
	if err = s.settleCutBossRanks(tx, time.Now().Unix()); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM rank_entries WHERE mode IN (?,?,?,?,?)", modes...); err != nil {
		return err
	}
	for _, table := range [...]string{"guild_war_progress", "guild_war_history", "guild_wars"} {
		if _, err = tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return err
		}
	}
	if err = s.Database.Queries.WithTx(tx).SaveMigration(ctx, name); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) advanceAccountCalendar(tx *sql.Tx, a *Account, now int64) error {
	a.AdvanceCalendar(s.Rules, now)
	if len(a.Calendar.Mail) == 0 {
		return nil
	}
	q, ctx := s.queries(tx), context.Background()
	count, err := q.MailboxCount(ctx, a.UID)
	if err != nil {
		return err
	}
	for len(a.Calendar.Mail) > 0 && count < s.mailboxLimit() {
		mail := a.Calendar.Mail[0]
		texts := s.Rules.MailTexts(mail.Template)
		var payloads [4][]byte
		for index, value := range [...]any{model.LocalizedText{"cn": texts[0], "en": texts[0]}, model.LocalizedText{"cn": texts[1], "en": texts[1]}, model.LocalizedText{"cn": texts[2], "en": texts[2]}, mail.Items} {
			if payloads[index], err = json.Marshal(value); err != nil {
				return err
			}
		}
		if _, err = s.deliverMail(q, ctx, database.DeliverMailParams{Owner: a.UID, Title: payloads[0], Content: payloads[1], Sender: payloads[2], Template: int64(mail.Template), Created: now, Expires: now + 30*DaySeconds, Items: payloads[3], HasReward: 1}); err != nil {
			return err
		}
		a.Calendar.Mail = a.Calendar.Mail[1:]
		count++
	}
	if len(a.Calendar.Mail) == 0 {
		a.Calendar.Mail = nil
	}
	return nil
}

func (a *Account) CalendarStale(r *Rules, now int64) bool {
	return r != nil && (a.Calendar.Cycle != calendar.Cycle() || a.Journey.Pass.ID != activeCalendarPass(r, now))
}

func (a *Account) AdvanceCalendar(r *Rules, now int64) {
	if r == nil {
		return
	}
	if a.Tasks == nil {
		a.Tasks = map[model.TaskID]model.Task{}
	}
	if a.Journey.TaskBaselines == nil {
		a.Journey.TaskBaselines = map[model.TaskID]int64{}
	}
	if cycle := calendar.Cycle(); a.Calendar.Cycle != cycle {
		a.Calendar.Cycle = cycle
		a.repeatCalendar(r)
	}
	active := activeCalendarPass(r, now)
	if a.Journey.Pass.ID == active {
		return
	}
	a.settlePass(r, a.Journey.Pass, r.Journey.Passes[a.Journey.Pass.ID], r.Settings.Mail.PassSettlement)
	a.Journey.Pass = model.JourneyPass{}
	if rule, known := r.Journey.Passes[active]; known {
		a.Journey.Pass = newJourneyPass(active)
		a.Journey.Pass.Tasks, a.Journey.Pass.WeeklyTasks = slices.Clone(rule.Tasks), slices.Clone(rule.WeeklyTasks)
		a.restartTasks(r, rule.Tasks, rule.WeeklyTasks)
	}
}

func activeCalendarPass(r *Rules, now int64) int {
	moment := now + r.Journey.CalendarOffset
	for _, id := range slices.Sorted(maps.Keys(r.Journey.Passes)) {
		if rule := r.Journey.Passes[id]; moment >= rule.Start && (rule.End <= 0 || moment <= rule.End) {
			return id
		}
	}
	return 0
}

func (a *Account) repeatCalendar(r *Rules) {
	a.settlePVPCycle(r)
	a.settlePass(r, a.Journey.Pass, r.Journey.Passes[a.Journey.Pass.ID], r.Settings.Mail.PassSettlement)
	a.Journey.Pass = model.JourneyPass{}
	for id, state := range a.Journey.Activities {
		rule, known := r.Journey.Activities[id]
		if !known || !calendar.Loops(rule.Start, rule.End) {
			continue
		}
		pass := r.Journey.LimitedPasses[state.Pass.ID]
		a.settlePass(r, state.Pass, pass, r.Settings.Mail.LimitedPassSettlement)
		a.restartTasks(r, pass.Tasks, pass.WeeklyTasks)
		if rule.Type == 22 {
			a.restartTasks(r, r.Journey.Hyperflows[rule.Parameter].Tasks)
		}
		a.restartTasks(r, rule.EventTasks()...)
		delete(a.Journey.Activities, id)
	}
	for id, state := range a.Activities {
		if rule, known := r.Journey.Activities[id]; known && calendar.Loops(rule.Start, rule.End) && state.Status == model.PVPInactive {
			a.Activities[id] = model.PVPActivity{Chance: state.Chance}
		}
	}
	if rule, known := r.Journey.Activities[a.Network.Activity]; known && calendar.Loops(rule.Start, rule.End) {
		a.restartTasks(r, r.Network.Activities[a.Network.Activity].Tasks)
		a.Network.Points, a.Network.Rewarded, a.Network.RewardedTasks = 0, nil, nil
	}
	a.settleBossScores(r)
	boss := &a.Events.Boss
	boss.Open, boss.TotalScore, boss.Rewards, boss.RankClaimed, boss.Pending = 0, 0, []int{}, nil, nil
	boss.Best, boss.Daily = map[int]int64{}, map[int]int{}
	for id := range a.ShopPurchases {
		if rule, known := r.Shops[id]; known && calendar.Loops(rule.Start, rule.End) {
			delete(a.ShopPurchases, id)
		} else if rule, known := r.System.Shops[id]; known && calendar.Loops(rule.Start, rule.End) {
			delete(a.ShopPurchases, id)
			delete(a.System.ShopWindows, id)
		}
	}
}

func (a *Account) restartTasks(r *Rules, lists ...[]model.TaskID) {
	for _, ids := range lists {
		for _, id := range ids {
			a.Tasks[id] = model.Task{Status: 2}
			a.Journey.TaskBaselines[id] = a.taskCumulativeValue(r, r.Tasks[id])
		}
	}
}

func (a *Account) settlePass(r *Rules, pass model.JourneyPass, rule JourneyPassRule, template int) {
	items := map[model.ItemID]int64{}
	for level := 1; level <= pass.Level; level++ {
		row, known := rule.Rewards[level]
		if !known {
			continue
		}
		var rewards []model.RewardID
		if pass.Free[level] != 1 {
			rewards = append(rewards, row.Free)
		}
		if pass.Active > 0 && pass.Paid[level] != 1 {
			rewards = append(rewards, row.Paid...)
		}
		for _, id := range rewards {
			if id == 0 {
				continue
			}
			if quantities, err := rewardItems(r, id, 1); err == nil {
				addItems(items, quantities)
			}
		}
	}
	if len(items) > 0 {
		a.Calendar.Mail = append(a.Calendar.Mail, CalendarMail{Template: template, Items: items})
	}
}
