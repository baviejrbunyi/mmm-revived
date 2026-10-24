package account

import (
	"maps"
	"math"
	"slices"

	"mmm-revived/internal/calendar"
	"mmm-revived/internal/model"
)

func (a *Account) InitializeRetailCamp(r *Rules, now int64) {
	if a.Journey.Activities == nil {
		a.Journey.Activities = map[model.ActivityID]model.JourneyActivity{}
	}
	for id, rule := range r.RetailActivity.Activities {
		if rule.Type != 35 {
			continue
		}
		activity, exists := a.Journey.Activities[id]
		available := a.journeyActivityAvailable(rule, now, r.Journey.CalendarOffset)
		if !exists && !available {
			continue
		}
		activity.Open = available
		if activity.Camp == nil {
			activity.Camp = &model.RetailCampState{BestScores: map[model.BattleID]int64{}}
		}
		if activity.Camp.BestScores == nil {
			activity.Camp.BestScores = map[model.BattleID]int64{}
		}
		a.Journey.Activities[id] = activity
	}
}

func (a *Account) retailCampRule(r *Rules, id model.ActivityID, node int, battle model.BattleID, now int64) (RetailCampNodeRule, RetailCampScoreRule, int, error) {
	activity, err := a.JourneyActivityRule(r, id, 35, now)
	if err != nil {
		return RetailCampNodeRule{}, RetailCampScoreRule{}, 0, err
	}
	rule, known := r.RetailActivity.CampNodes[node]
	index := slices.Index(rule.Battles, battle)
	if !known || index < 0 || index >= len(rule.Difficulties) || index >= len(rule.Rewards) || !slices.Contains(r.RetailActivity.CampGroups[activity.Parameter], node) {
		return rule, RetailCampScoreRule{}, 0, rejected("camp battle does not match activity node")
	}
	score, known := r.RetailActivity.CampScores[rule.Difficulties[index]]
	if !known || score.Item <= 0 || score.Base < 0 || score.PerSecond < 0 || score.Limit < 0 || score.Limit > 86400 || score.PerSecond > (math.MaxInt64-score.Base)/max(int64(1), score.Limit) || r.RetailActivity.CampScorePolicy != "base_plus_remaining_seconds" {
		return rule, score, index, rejected("camp score policy is unavailable")
	}
	return rule, score, index, nil
}

func (a *Account) BeginRetailCamp(r *Rules, id model.ActivityID, node int, battle model.BattleID, now int64) error {
	rule, _, _, err := a.retailCampRule(r, id, node, battle, now)
	if err != nil {
		return err
	}
	if !slices.Contains(rule.Days, calendar.LogicWeekday(now)) {
		return rejected("camp node is closed today")
	}
	activity := a.Journey.Activities[id]
	if activity.Camp == nil {
		return rejected("camp activity has not initialized")
	}
	if len(rule.Cost) == 0 {
		return rejected("camp entry cost is unavailable")
	}
	for item, quantity := range rule.Cost {
		if item == 0 || quantity <= 0 || a.Items[item] < quantity {
			return rejected("insufficient camp entry items")
		}
	}
	state := *activity.Camp
	state.Run = &model.RetailCampRun{Node: node, Battle: battle, Started: now}
	state.Pending, state.Settled = nil, nil
	activity.Camp = &state
	a.Journey.Activities[id] = activity
	return nil
}

func (a *Account) ValidateRetailCampHeroes(r *Rules, id model.ActivityID, node int, heroes []model.HeroID) error {
	rule, known := r.RetailActivity.CampNodes[node]
	state := a.Journey.Activities[id].Camp
	if !known || len(heroes) == 0 || state == nil || state.Run == nil || state.Run.Node != node {
		return rejected("camp deployment is unavailable")
	}
	for _, hero := range heroes {
		if _, owned := a.Heroes[hero]; !owned || rule.Camp != 0 && r.HeroTraits[hero].Camp != rule.Camp {
			return rejected("hero does not belong to the required camp")
		}
	}
	return nil
}

func (a *Account) FinishRetailCamp(r *Rules, id model.ActivityID, node int, battle model.BattleID, won bool, duration int64, now int64) error {
	_, scoreRule, _, err := a.retailCampRule(r, id, node, battle, now)
	if err != nil {
		return err
	}
	activity := a.Journey.Activities[id]
	if activity.Camp == nil || activity.Camp.Run == nil || activity.Camp.Run.Node != node || activity.Camp.Run.Battle != battle || duration < 0 || duration > 86400 {
		return rejected("camp result does not match an active battle")
	}
	state := *activity.Camp
	state.Run = nil
	if won {
		score := scoreRule.Base + max(int64(0), scoreRule.Limit-duration)*scoreRule.PerSecond
		state.Pending = &model.RetailCampPending{Node: node, Battle: battle, Duration: duration, Score: score}
	}
	activity.Camp = &state
	a.Journey.Activities[id] = activity
	return nil
}

func (a *Account) SettleRetailCamp(r *Rules, id model.ActivityID, now int64) (model.RetailCampPending, map[model.ItemID]int64, error) {
	if _, err := a.JourneyActivityRule(r, id, 35, now); err != nil {
		return model.RetailCampPending{}, nil, err
	}
	activity := a.Journey.Activities[id]
	if activity.Camp == nil {
		return model.RetailCampPending{}, nil, rejected("camp settlement is unavailable")
	}
	state := *activity.Camp
	if state.Pending == nil {
		if state.Settled != nil {
			return *state.Settled, map[model.ItemID]int64{}, nil
		}
		return model.RetailCampPending{}, nil, rejected("no validated camp victory awaits settlement")
	}
	pending := *state.Pending
	rule, scoreRule, index, err := a.retailCampRule(r, id, pending.Node, pending.Battle, now)
	if err != nil {
		return pending, nil, err
	}
	quantities, err := rewardItems(r, rule.Rewards[index], 1)
	if err != nil {
		return pending, nil, err
	}
	if !addQuantity(quantities, scoreRule.Item, pending.Score) {
		return pending, nil, rejected("camp reward overflow")
	}
	for item, amount := range rule.Cost {
		if amount < 0 || a.Items[item] < amount {
			return pending, nil, rejected("insufficient camp entry items")
		}
	}
	if err = a.GrantItems(r, quantities); err != nil {
		return pending, nil, err
	}
	if err = a.Spend(rule.Cost); err != nil {
		return pending, nil, err
	}
	state.BestScores = maps.Clone(state.BestScores)
	state.BestScores[pending.Battle] = max(state.BestScores[pending.Battle], pending.Score)
	state.Pending, state.Settled = nil, &pending
	activity.Camp = &state
	a.Journey.Activities[id] = activity
	return pending, quantities, nil
}
