package account

import (
	"fmt"
	"maps"
	"math"
	"slices"

	"mmm-revived/internal/model"
)

var journeyActivityTypes = [...]int{1, 3, 6, 7, 10, 12, 16, 17, 18, 20, 22, 24, 25, 26, 28, 30, 32}

var (
	ErrJourneyPassUnavailable = fmt.Errorf("%w: battle pass is unavailable", ErrProgression)
	ErrJourneyPassLevelFull   = fmt.Errorf("%w: battle pass level is full", ErrProgression)
	ErrJourneyPassCurrency    = fmt.Errorf("%w: insufficient battle pass currency", ErrProgression)
)

func newJourneyPass(id int) model.JourneyPass {
	return model.JourneyPass{ID: id, Level: 1, Free: map[int]int{}, Paid: map[int]int{}}
}

func (a *Account) ensureTasks(lists ...[]model.TaskID) {
	for _, ids := range lists {
		for _, id := range ids {
			if _, exists := a.Tasks[id]; !exists {
				a.Tasks[id] = model.Task{Status: 2}
			}
		}
	}
}

func levelJourneyPass(pass *model.JourneyPass, rule JourneyPassRule) {
	for pass.Level < rule.MaximumLevel && pass.Experience >= rule.ExperiencePerLevel {
		pass.Experience -= rule.ExperiencePerLevel
		pass.Level++
	}
	if pass.Level == rule.MaximumLevel {
		pass.Experience = 0
	}
}

func (a *Account) InitializeJourney(r *Rules, now int64) {
	if a.Tasks == nil {
		a.Tasks = map[model.TaskID]model.Task{}
	}
	state := &a.Journey
	day := LogicDay(now)
	if state.SpentItems == nil {
		state.SpentItems = map[model.ItemID]int64{}
	}
	if state.TaskBaselines == nil {
		state.TaskBaselines = map[model.TaskID]int64{}
	}
	if r != nil && state.TaskDay != day {
		state.TaskDay = day
		for id, rule := range r.Tasks {
			if rule.Daily {
				a.Tasks[id] = model.Task{Status: 2}
				state.TaskBaselines[id] = a.taskCumulativeValue(r, rule)
			}
		}
		if a.Retail {
			a.resetRetailDaily(r)
		}
	}
	if state.Challenges == nil {
		state.Challenges = map[int]model.JourneyChallengeRecord{}
	}
	if state.Spiral == nil {
		state.Spiral = map[int]int{}
	}
	if state.SpiralRewards == nil {
		state.SpiralRewards = map[int][]int{}
	}
	if state.InviteRewards == nil {
		state.InviteRewards = map[model.UserID]map[int]bool{}
	}
	if !state.Initialized {
		state.Initialized = true
		state.Novice = model.JourneyNovice{Nodes: map[int]int{}, Completed: map[model.TaskID]bool{}, Rewards: map[int]bool{}, BattleWins: map[model.BattleID]bool{}}
		state.Activities = map[model.ActivityID]model.JourneyActivity{}
	}
	if state.LoginDays == 0 || state.LoginDay != day {
		state.LoginDays++
		state.LoginDay = day
	}
	if r == nil {
		return
	}
	a.backfillSevenDayMarks(r)
	calendar := now + r.Journey.CalendarOffset
	for _, id := range slices.Sorted(maps.Keys(r.Journey.Passes)) {
		rule := r.Journey.Passes[id]
		if calendar < rule.Start || rule.End > 0 && calendar > rule.End {
			continue
		}
		if state.Pass.ID != id {
			state.Pass = newJourneyPass(id)
		}
		state.Pass.Tasks = slices.Clone(rule.Tasks)
		state.Pass.WeeklyTasks = slices.Clone(rule.WeeklyTasks)
		if week := LogicWeek(now); state.Pass.Week != week {
			state.Pass.Week = week
			state.Pass.WeeklyXP = 0
			for _, taskID := range rule.WeeklyTasks {
				a.Tasks[taskID] = model.Task{Status: 2}
				state.TaskBaselines[taskID] = a.taskCumulativeValue(r, r.Tasks[taskID])
			}
		}
		a.ensureTasks(rule.Tasks, rule.WeeklyTasks)
		break
	}
	for id, activity := range state.Activities {
		activity.Open = false
		state.Activities[id] = activity
	}
	for id, rule := range r.Journey.Activities {
		if !a.journeyActivityAvailable(rule, now, r.Journey.CalendarOffset) {
			continue
		}
		if !slices.Contains(journeyActivityTypes[:], rule.Type) {
			continue
		}
		activity := state.Activities[id]
		if activity.Initialized == 0 {
			activity = model.JourneyActivity{Initialized: now, TrailerRewards: map[int]int{}, NodeStars: map[int]int{}, Missions: map[int]model.JourneyMission{}, MiniGames: map[int]int64{}, Pass: newJourneyPass(0)}
		}
		activity.Open = true
		if rule.Type == 20 {
			activity.BossGroup, activity.BossTickets = r.limitedBoss(calendar)
		}
		if rule.Type == 20 && activity.Pass.ID == 0 {
			passID := rule.PassID
			if passID == 0 && rule.Parameter > 0 && rule.Parameter <= len(r.Journey.LimitedEventPasses) {
				passID = r.Journey.LimitedEventPasses[rule.Parameter-1]
			}
			if pass, known := r.Journey.LimitedPasses[passID]; known {
				activity.Pass = newJourneyPass(passID)
				activity.Pass.Tasks = slices.Clone(pass.Tasks)
				activity.Pass.WeeklyTasks = slices.Clone(pass.WeeklyTasks)
				a.ensureTasks(pass.Tasks, pass.WeeklyTasks)
			}
		}
		if rule.Type == 20 {
			for mission, state := range activity.Missions {
				if _, exists := state.Nodes[state.Node]; state.Node == 0 || exists && state.Passed != nil {
					continue
				}
				if state.Passed == nil {
					state.Passed = map[int]int{}
					for node, entry := range state.Nodes {
						if node != state.Node || entry.Rewarded == 1 {
							state.Passed[node] = 1
						}
					}
				}
				enterJourneyMissionNode(&state, state.Node)
				activity.Missions[mission] = state
			}
		}
		if rule.Type == 24 && activity.LastSign != day {
			activity.Signed = min(activity.Signed+1, len(rule.Rewards))
			activity.LastSign = day
		}
		if rule.Type == 30 {
			for index, trailer := range r.Journey.Trailers {
				if value, exists := activity.TrailerRewards[index]; exists && value != 1 {
					activity.TrailerRewards[index] = 0
				} else if !exists && (trailer.Level > 0 && a.Level >= trailer.Level || trailer.Days > 0 && trailerUnlocked(activity.Initialized, trailer.Days, now)) {
					activity.TrailerRewards[index] = 0
				}
			}
		}
		if rule.Type == 22 && activity.DailyDay != day {
			activity.DailyDay = day
			for _, taskID := range r.Journey.Hyperflows[rule.Parameter].Tasks {
				a.Tasks[taskID] = model.Task{Status: 2}
			}
		}
		if rule.Type == 22 && len(activity.BuffNodes) == 0 {
			hyper := r.Journey.Hyperflows[rule.Parameter]
			activity.HyperMaximum = 0
			for _, node := range hyper.BuffNodes {
				activity.HyperMaximum = max(activity.HyperMaximum, r.Journey.HyperBuffs[node].Threshold)
			}
			for _, node := range hyper.BuffNodes {
				entry := r.Journey.HyperBuffs[node]
				choice := 0
				if entry.Type == 1 {
					choice = node
				}
				activity.BuffNodes = append(activity.BuffNodes, model.JourneyBuffNode{ID: node, Active: entry.Threshold <= activity.HyperPoint, Choice: choice})
			}
			for index, node := range hyper.BattleNodes {
				entry := r.Journey.HyperBattles[node]
				value := model.JourneyHyperNode{ID: node, Open: index == 0}
				if len(entry.Battles) > 0 {
					value.First.ID = entry.Battles[0]
				}
				if len(entry.Battles) > 1 {
					value.Second.ID = entry.Battles[1]
				}
				activity.BattleNodes = append(activity.BattleNodes, value)
			}
		}
		a.refreshJourneyEvent(r, rule, &activity, now)
		state.Activities[id] = activity
		if rule.Type == 20 {
			a.redeemLimitedPassItems(r, id)
		}
	}
	a.InitializeRetailPVP(r, now)
	a.InitializeRetailShare(r, now)
	a.InitializeRetailCamp(r, now)
}

func (a *Account) JourneySnapshot(profile *model.Avatar, now int64) {
	profile.Challenges.Challenges = a.Journey.Challenges
	profile.Spiral.Missions = a.Journey.Spiral
	profile.Spiral.Groups = make(map[int]model.JourneySpiralGroup, len(a.Journey.SpiralRewards))
	for id, rewards := range a.Journey.SpiralRewards {
		profile.Spiral.Groups[id] = model.JourneySpiralGroup{Rewards: rewards}
	}
	novice := a.Journey.Novice
	profile.Novice = model.NoviceComponent{Task: novice.Task, Node: novice.Node, Nodes: novice.Nodes, Place: map[string]int{}}
	if novice.Task != 0 {
		profile.Novice.Status = 2
	}
	pass := a.Journey.Pass
	profile.BattlePass = model.BattlePassComponent{PassID: pass.ID, Level: pass.Level, Experience: pass.Experience, WeeklyXP: pass.WeeklyXP, ActiveType: pass.Active, FreeRewards: pass.Free, PaidRewards: pass.Paid, CurrentTasks: pass.Tasks, WeeklyTasks: pass.WeeklyTasks}
	profile.MonthlyPass = model.MonthlyPassComponent{ExpiresAt: a.Journey.Monthly.Expires, LastReward: a.Journey.Monthly.LastDaily, RewardReady: a.Journey.Monthly.Expires > now && LogicDay(a.Journey.Monthly.LastDaily) != LogicDay(now)}
	for id, activity := range a.Journey.Activities {
		if !activity.Open {
			continue
		}
		if activity.Camp != nil {
			profile.Activities.Info[id] = activity.Camp.Snapshot()
		} else if activity.RetailPVP != nil {
			profile.Activities.Info[id] = *activity.RetailPVP
		} else {
			profile.Activities.Info[id] = activity
		}
		profile.Activities.Status[id] = 1
	}
}

func (a *Account) journeyActivityAvailable(rule JourneyActivityRule, now int64, offset int64) bool {
	calendar := now + offset
	if a.Level < rule.Level && !a.journeyLoginUnlocked(rule, now) || calendar < rule.Start || rule.End > 0 && calendar > rule.End {
		return false
	}
	return rule.Duration <= 0 || LogicDay(now)-LogicDay(a.CreatedAt) <= int64(rule.Duration)
}

func (a *Account) JourneyActivityRule(r *Rules, id model.ActivityID, kind int, now int64) (JourneyActivityRule, error) {
	rule, known := r.Journey.Activities[id]
	if !known || rule.Type != kind || !a.journeyActivityAvailable(rule, now, r.Journey.CalendarOffset) {
		return rule, rejected("activity is unavailable")
	}
	if _, exists := a.Journey.Activities[id]; !exists {
		return rule, rejected("activity has not initialized")
	}
	return rule, nil
}

func trailerUnlocked(initialized int64, days int, now int64) bool {
	return LogicDay(now)-LogicDay(initialized) >= int64(days-1)
}

func (a *Account) openNoviceNodes(nodes []int) {
	for _, node := range nodes {
		if a.Journey.Novice.Nodes[node] < 1 {
			a.Journey.Novice.Nodes[node] = 1
		}
	}
}

func (a *Account) StartNoviceNode(r *Rules, id int) error {
	nodes := r.Journey.NoviceTasks[a.Journey.Novice.Task].Nodes
	if !slices.Contains(nodes, id) || a.Journey.Novice.Nodes[id] == 3 {
		return rejected("novice node is unavailable")
	}
	if id != nodes[0] && r.Journey.NoviceNodes[id].Type == 13 {
		if a.Journey.Novice.Nodes[id] < 2 {
			a.Journey.Novice.Nodes[id] = 2
		}
		return nil
	}
	a.Journey.Novice.Node = id
	a.Journey.Novice.Nodes[id] = 2
	return nil
}

func (a *Account) noviceBattleWon(r *Rules, rule JourneyNoviceNode) bool {
	if len(rule.Parameters) == 0 {
		return false
	}
	wins := a.Journey.Novice.BattleWins
	if wins[model.BattleID(rule.Parameters[0])] {
		return true
	}
	if len(rule.Parameters) < 2 {
		return false
	}
	scene, known := r.Journey.NoviceNodes[int(rule.Parameters[1])]
	return known && scene.Type == 13 && len(scene.Parameters) > 1 && scene.Parameters[0] == rule.Parameters[0] && wins[model.BattleID(scene.Parameters[1])]
}

func (a *Account) FinishNoviceNode(r *Rules, id int) error {
	if a.Journey.Novice.Nodes[id] == 3 {
		return nil
	}
	if a.Journey.Novice.Node != id || a.Journey.Novice.Nodes[id] != 2 {
		return rejected("novice node has not started")
	}
	rule, known := r.Journey.NoviceNodes[id]
	if !known {
		return rejected("unknown novice node")
	}
	if rule.Type == 10 && !a.Journey.Novice.Rewards[id] {
		return rejected("novice node reward has not been claimed")
	}
	if rule.Type == 3 && !a.noviceBattleWon(r, rule) {
		return rejected("novice battle is incomplete")
	}
	a.Journey.Novice.Nodes[id] = 3
	task := a.Journey.Novice.Task
	nodes := r.Journey.NoviceTasks[task].Nodes
	if len(nodes) == 0 || id != nodes[0] {
		a.Journey.Novice.Node = 0
		return nil
	}
	for _, node := range nodes {
		a.Journey.Novice.Nodes[node] = 3
	}
	a.Journey.Novice.Completed[task] = true
	a.Journey.Novice.Task = 0
	a.Journey.Novice.Node = 0
	if next := r.Journey.NoviceTasks[task].Next; len(next) > 0 {
		a.Journey.Novice.Task = next[0]
		a.openNoviceNodes(r.Journey.NoviceTasks[next[0]].Nodes)
	}
	return nil
}

func (a *Account) ClaimNoviceReward(r *Rules, id int) (map[model.ItemID]int64, error) {
	rule, known := r.Journey.NoviceNodes[id]
	if !known || rule.Type != 10 || len(rule.Parameters) == 0 {
		return nil, rejected("novice reward is unavailable")
	}
	if a.Journey.Novice.Rewards[id] {
		if a.Journey.Novice.Node == id && a.Journey.Novice.Nodes[id] == 2 {
			if err := a.FinishNoviceNode(r, id); err != nil {
				return nil, err
			}
		}
		return map[model.ItemID]int64{}, nil
	}
	if a.Journey.Novice.Node != id {
		return nil, rejected("novice reward is unavailable")
	}
	reward := model.RewardID(rule.Parameters[0])
	quantities := map[model.ItemID]int64{}
	if _, known := r.Rewards[reward]; known {
		var err error
		if quantities, err = rewardItems(r, reward, 1); err != nil {
			return nil, err
		}
	}
	if err := a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	a.Journey.Novice.Rewards[id] = true
	visible := false
	for _, count := range quantities {
		if count > 0 {
			visible = true
			break
		}
	}
	if !visible {
		if err := a.FinishNoviceNode(r, id); err != nil {
			return nil, err
		}
	}
	return quantities, nil
}

func (a *Account) passRule(r *Rules, now int64) (JourneyPassRule, error) {
	rule, known := r.Journey.Passes[a.Journey.Pass.ID]
	calendar := now + r.Journey.CalendarOffset
	if !known || calendar < rule.Start || rule.End > 0 && calendar > rule.End {
		return rule, ErrJourneyPassUnavailable
	}
	return rule, nil
}

func (a *Account) BuyJourneyPassLevels(r *Rules, target int, now int64) error {
	rule, err := a.passRule(r, now)
	if err != nil {
		return err
	}
	if rule.Currency <= 0 || rule.LevelPrice <= 0 {
		return ErrJourneyPassUnavailable
	}
	if a.Journey.Pass.Level >= rule.MaximumLevel || target > rule.MaximumLevel {
		return ErrJourneyPassLevelFull
	}
	if target <= a.Journey.Pass.Level {
		return rejected("invalid pass level purchase")
	}
	levels := int64(target - a.Journey.Pass.Level)
	if levels > math.MaxInt64/rule.LevelPrice {
		return rejected("pass price overflow")
	}
	cost := levels * rule.LevelPrice
	if a.Items[rule.Currency] < cost {
		return ErrJourneyPassCurrency
	}
	if err = a.Spend(map[model.ItemID]int64{rule.Currency: cost}); err != nil {
		return err
	}
	a.Journey.Pass.Level = target
	return nil
}

func mergeJourneyRewards(r *Rules, ids []model.RewardID) (map[model.ItemID]int64, error) {
	result := map[model.ItemID]int64{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		quantities, err := rewardItems(r, id, 1)
		if err != nil {
			return nil, err
		}
		for item, count := range quantities {
			if result[item] > math.MaxInt64-count {
				return nil, rejected("reward overflow")
			}
			result[item] += count
		}
	}
	return result, nil
}

func claimJourneyPass(a *Account, r *Rules, pass *model.JourneyPass, rule JourneyPassRule, levels []int, active int) (map[model.ItemID]int64, error) {
	if active < -1 || active > 1 || active == 1 && pass.Active == 0 || len(levels) > rule.MaximumLevel {
		return nil, rejected("pass reward track unavailable")
	}
	var rewards []model.RewardID
	free, paid := maps.Clone(pass.Free), maps.Clone(pass.Paid)
	if free == nil {
		free = map[int]int{}
	}
	if paid == nil {
		paid = map[int]int{}
	}
	seen := make(map[int]bool, len(levels))
	for _, level := range levels {
		row, known := rule.Rewards[level]
		if !known || level < 1 || level > pass.Level || seen[level] {
			return nil, rejected("pass reward level unavailable")
		}
		seen[level] = true
		if active != 1 && free[level] != 1 {
			rewards = append(rewards, row.Free)
			free[level] = 1
		}
		if active != 0 && pass.Active > 0 && paid[level] != 1 {
			rewards = append(rewards, row.Paid...)
			paid[level] = 1
		}
	}
	quantities, err := mergeJourneyRewards(r, rewards)
	if err != nil {
		return nil, err
	}
	if err = a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	pass.Free, pass.Paid = free, paid
	return quantities, nil
}

func (a *Account) ClaimJourneyPass(r *Rules, levels []int, active int, now int64) (map[model.ItemID]int64, error) {
	rule, err := a.passRule(r, now)
	if err != nil {
		return nil, err
	}
	return claimJourneyPass(a, r, &a.Journey.Pass, rule, levels, active)
}

func (a *Account) AddJourneyPassExperience(r *Rules, amount int64, weekly bool, now int64) error {
	rule, err := a.passRule(r, now)
	if err != nil {
		return nil
	}
	if amount < 0 || rule.ExperiencePerLevel <= 0 {
		return rejected("invalid pass experience")
	}
	pass := &a.Journey.Pass
	if week := LogicWeek(now); pass.Week != week {
		pass.Week = week
		pass.WeeklyXP = 0
	}
	if weekly && rule.WeeklyCap > 0 {
		amount = min(amount, max(int64(0), rule.WeeklyCap-pass.WeeklyXP))
		pass.WeeklyXP += amount
	}
	if pass.Experience > math.MaxInt64-amount {
		return rejected("pass experience overflow")
	}
	pass.Experience += amount
	levelJourneyPass(pass, rule)
	return nil
}

func (a *Account) ActivateMonthlyPass(r *Rules, now int64) error {
	monthly := &a.Journey.Monthly
	duration := int64(r.Journey.MonthlyDuration) * 86400
	expires := max(now, monthly.Expires) + duration
	if duration <= 0 || expires > now+int64(r.Journey.MonthlyMaximumDuration)*86400 {
		return rejected("monthly pass duration limit reached")
	}
	if monthly.PendingPurchase == math.MaxInt32 {
		return rejected("monthly reward overflow")
	}
	monthly.Expires = expires
	monthly.PendingPurchase++
	return nil
}

func (a *Account) ClaimMonthlyPass(r *Rules, daily bool, now int64) (map[model.ItemID]int64, error) {
	monthly := &a.Journey.Monthly
	reward, count := r.Journey.MonthlyPurchaseReward, int64(monthly.PendingPurchase)
	if daily {
		if monthly.Expires <= now || monthly.LastDaily > 0 && LogicDay(monthly.LastDaily) == LogicDay(now) {
			return map[model.ItemID]int64{}, nil
		}
		reward, count = r.Journey.MonthlyDailyReward, 1
	}
	if count == 0 {
		return map[model.ItemID]int64{}, nil
	}
	quantities, err := rewardItems(r, reward, count)
	if err != nil {
		return nil, err
	}
	if err = a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	if daily {
		monthly.LastDaily = now
	} else {
		monthly.PendingPurchase = 0
	}
	return quantities, nil
}

func (a *Account) ActivateJourneyPass(r *Rules, kind int, now int64) error {
	if _, err := a.passRule(r, now); err != nil {
		return err
	}
	if kind < 1 || kind > 3 {
		return rejected("invalid pass activation")
	}
	if kind == 2 && a.Journey.Pass.Active == 0 {
		return rejected("pass upgrade requires basic activation")
	}
	target := 1
	if kind > 1 {
		target = 2
	}
	if a.Journey.Pass.Active >= target {
		return rejected("pass entitlement is already active")
	}
	a.Journey.Pass.Active = target
	return nil
}
