package account

import (
	"maps"
	"math"
	"slices"
	"time"

	"mmm-revived/internal/model"
)

func (a *Account) claimJourneyTasksNow(r *Rules, ids []model.TaskID) error {
	return a.claimJourneyTasks(r, ids, time.Now().Unix())
}

func (a *Account) claimJourneyTasks(r *Rules, ids []model.TaskID, now int64) error {
	if len(ids) == 0 || len(ids) > 100 {
		return rejected("invalid task claim count")
	}
	seen := make(map[model.TaskID]bool, len(ids))
	quantities := map[model.ItemID]int64{}
	var weeklyXP, regularXP int64
	weeklyTasks := r.Journey.Passes[a.Journey.Pass.ID].WeeklyTasks
	for _, id := range ids {
		rule, known := r.Tasks[id]
		if !known || !a.journeyTaskEnabled(r, id, rule, now) || a.Tasks[id].Status != 3 || seen[id] {
			return rejected("task is not claimable")
		}
		seen[id] = true
		reward, err := mergeJourneyRewards(r, []model.RewardID{a.journeyTaskReward(id, rule)})
		if err != nil {
			return err
		}
		experience := reward[r.Settings.Items.PassExperience]
		if slices.Contains(weeklyTasks, id) {
			if weeklyXP > math.MaxInt64-experience {
				return rejected("weekly task experience overflow")
			}
			weeklyXP += experience
		} else {
			if regularXP > math.MaxInt64-experience {
				return rejected("task experience overflow")
			}
			regularXP += experience
		}
		delete(reward, r.Settings.Items.PassExperience)
		if marks := r.Journey.SevenDay.Marks(id); marks > 0 {
			reward[r.Journey.SevenDay.Mark] += marks
		}
		for item, count := range reward {
			if quantities[item] > math.MaxInt64-count {
				return rejected("task reward overflow")
			}
			quantities[item] += count
		}
	}
	next := *a
	next.Tasks = maps.Clone(a.Tasks)
	oldWeekly := next.Journey.Pass.WeeklyXP
	if next.Journey.Pass.Week != LogicWeek(now) {
		oldWeekly = 0
	}
	if err := next.AddJourneyPassExperience(r, weeklyXP, true, now); err != nil {
		return err
	}
	awardedWeekly := max(int64(0), next.Journey.Pass.WeeklyXP-oldWeekly)
	if err := next.AddJourneyPassExperience(r, regularXP, false, now); err != nil {
		return err
	}
	if regularXP > math.MaxInt64-awardedWeekly || next.Items[r.Settings.Items.PassExperience] > math.MaxInt64-regularXP-awardedWeekly {
		return rejected("task reward experience overflow")
	}
	for _, id := range ids {
		task := next.Tasks[id]
		task.Status = 4
		next.Tasks[id] = task
		if r.Tasks[id].Daily {
			next.Journey.DailyClaims += min(1, math.MaxInt64-next.Journey.DailyClaims)
		}
	}
	if err := next.GrantItems(r, quantities); err != nil {
		return err
	}
	next.Items[r.Settings.Items.PassExperience] += regularXP + awardedWeekly
	next.Journey.TaskStats.Doubled = maps.Clone(next.Journey.TaskStats.Doubled)
	for _, id := range ids {
		task := next.Tasks[id]
		task.Status = 4
		next.Tasks[id] = task
		delete(next.Journey.TaskStats.Doubled, id)
	}
	*a = next
	return nil
}

func (a *Account) ProcessAutomaticTasks(r *Rules, now int64) error {
	if r == nil || a.Journey.ProcessingTasks {
		return nil
	}
	next := *a
	next.Tasks = maps.Clone(a.Tasks)
	next.Journey.Activities = maps.Clone(a.Journey.Activities)
	next.Journey.ProcessingTasks = true
	for range len(r.Tasks) + 1 {
		next.refreshJourneyTaskValuesAt(r, now)
		var ready []model.TaskID
		for id, rule := range r.Tasks {
			if next.Tasks[id].Status == 3 && (rule.AutoClaim == 1 || next.journeyEventAutoClaim(r, id, now)) && next.journeyTaskEnabled(r, id, rule, now) {
				ready = append(ready, id)
			}
		}
		if len(ready) == 0 {
			next.Journey.ProcessingTasks = false
			*a = next
			return nil
		}
		next.Journey.AutomaticPending = true
		slices.Sort(ready)
		for chunk := range slices.Chunk(ready, 100) {
			if err := next.claimJourneyTasks(r, chunk, now); err != nil {
				return err
			}
		}
	}
	return rejected("automatic task dependency cycle")
}
