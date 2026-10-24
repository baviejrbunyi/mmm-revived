package account

import (
	"maps"
	"slices"
	"strconv"

	"mmm-revived/internal/model"
)

const DefaultDispatchDuration int64 = 72000

const (
	DefaultDispatchOffers       = 8
	DefaultDispatchRefreshLimit = 5
)

func (a *Account) dispatchOffers(r *Rules, state model.DispatchState) (model.DispatchState, error) {
	pool := make([]model.DispatchRuleID, 0, len(r.DispatchMissions))
	for id, rule := range r.DispatchMissions {
		if a.Level >= rule.TeamLevel && (rule.MaximumLevel == 0 || a.Level <= rule.MaximumLevel) {
			pool = append(pool, id)
		}
	}
	slices.Sort(pool)
	if len(pool) == 0 || len(r.DispatchMaps) == 0 {
		return state, rejected("dispatch missions are unavailable at this level")
	}
	selected, err := sample(pool, DefaultDispatchOffers)
	if err != nil {
		return state, err
	}
	missions := make(map[model.DispatchID]model.DispatchMission, len(state.Missions)+len(selected))
	for id, mission := range state.Missions {
		if mission.Started != 0 {
			missions[id] = mission
		}
	}
	for _, rule := range selected {
		location, err := randomChoice(r.DispatchMaps)
		if err != nil {
			return state, err
		}
		state.NextID++
		id := model.DispatchID(strconv.FormatInt(state.NextID, 10))
		missions[id] = model.DispatchMission{Type: rule, Map: location, Heroes: []model.HeroID{}}
	}
	state.Missions = missions
	return state, nil
}

func (a *Account) PrepareDispatch(r *Rules, now int64) error {
	day := LogicDay(now)
	if r == nil || len(r.DispatchMissions) == 0 || a.Level < 12 || a.Dispatch.Initialized && a.Dispatch.Day == day {
		return nil
	}
	state := a.Dispatch
	state.Initialized, state.Day, state.Refreshes, state.Started = true, day, 0, 0
	state, err := a.dispatchOffers(r, state)
	if err != nil {
		return err
	}
	a.Dispatch = state
	return nil
}

func (a *Account) RefreshDispatch(r *Rules, now int64) error {
	if err := a.PrepareDispatch(r, now); err != nil {
		return err
	}
	empty := len(a.Dispatch.Missions) == 0
	if !a.Dispatch.Initialized || !empty && a.Dispatch.Refreshes >= DefaultDispatchRefreshLimit {
		return rejected("dispatch refresh limit reached")
	}
	state, err := a.dispatchOffers(r, a.Dispatch)
	if err != nil {
		return err
	}
	if !empty {
		state.Refreshes++
	}
	a.Dispatch = state
	return nil
}

func (a *Account) StartDispatch(r *Rules, id model.DispatchID, heroes []model.HeroID, now int64) error {
	mission, exists := a.Dispatch.Missions[id]
	rule, known := r.DispatchMissions[mission.Type]
	if !exists || !known || mission.Started != 0 || len(heroes) != rule.Heroes || a.Level < rule.TeamLevel {
		return rejected("invalid dispatch mission")
	}
	active := 0
	used := map[model.HeroID]bool{}
	for _, entry := range a.Dispatch.Missions {
		if entry.Started == 0 {
			continue
		}
		active++
		for _, hero := range entry.Heroes {
			used[hero] = true
		}
	}
	if active >= r.DispatchLimits[max(1, a.TeamBreak)] {
		return rejected("all dispatch slots are occupied")
	}
	for _, id := range heroes {
		hero, owned := a.Heroes[id]
		if !owned || used[id] || hero.Trial != 0 || hero.Level < rule.HeroLevel {
			return rejected("hero is unavailable for dispatch")
		}
		used[id] = true
	}
	mission.Started, mission.Ends, mission.Heroes = now, now+DefaultDispatchDuration, slices.Clone(heroes)
	a.Dispatch.Missions[id] = mission
	a.Dispatch.Started++
	a.RecordJourneyTaskEvent(r, 67, 1, 0)
	return nil
}

func (a *Account) CancelDispatch(id model.DispatchID, now int64) error {
	mission, exists := a.Dispatch.Missions[id]
	if !exists || mission.Started == 0 || now >= mission.Ends {
		return rejected("dispatch cannot be cancelled")
	}
	mission.Started, mission.Ends, mission.Heroes = 0, 0, []model.HeroID{}
	a.Dispatch.Missions[id] = mission
	return nil
}

func (a *Account) ClaimDispatch(r *Rules, ids []model.DispatchID, now int64) (map[model.ItemID]int64, model.ItemID, error) {
	if len(ids) == 0 || len(ids) > 20 {
		return nil, 0, rejected("invalid dispatch reward selection")
	}
	rewards := map[model.ItemID]int64{}
	var heroes []model.HeroID
	seen := make(map[model.DispatchID]bool, len(ids))
	for _, id := range ids {
		mission, exists := a.Dispatch.Missions[id]
		rule, known := r.DispatchMissions[mission.Type]
		if !exists || !known || seen[id] || mission.Started == 0 || now < mission.Ends {
			return nil, 0, rejected("dispatch reward is not available")
		}
		seen[id] = true
		items, err := rewardItems(r, rule.Reward, 1)
		if err != nil {
			return nil, 0, err
		}
		addItems(rewards, items)
		bonus := int64(0)
		for _, hero := range mission.Heroes {
			if slices.Contains(rule.Camps, r.HeroTraits[hero].Camp) {
				bonus++
			}
		}
		if rule.ExtraReward != 0 && bonus > 0 {
			items, err = rewardItems(r, rule.ExtraReward, bonus)
			if err != nil {
				return nil, 0, err
			}
			addItems(rewards, items)
		}
		heroes = append(heroes, mission.Heroes...)
	}
	if err := a.GrantItems(r, rewards); err != nil {
		return nil, 0, err
	}
	egg, err := a.rollDispatchEgg(r, len(ids), now)
	if err != nil {
		return nil, 0, err
	}
	a.Dispatch.Missions = maps.Clone(a.Dispatch.Missions)
	for _, id := range ids {
		delete(a.Dispatch.Missions, id)
	}
	a.AddProficiency(r, heroes, 200)
	a.RecordJourneyTaskEvent(r, 45, len(ids), 0)
	return rewards, egg, nil
}
