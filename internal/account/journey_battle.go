package account

import (
	"maps"
	"math"
	"math/bits"
	"slices"

	"mmm-revived/internal/model"
)

func (a *Account) ValidateJourneyActivityBattle(r *Rules, id model.ActivityID, mission, node int, battle model.BattleID, now int64) error {
	rule, known := r.Journey.Activities[id]
	if !known || rule.Type != 20 && rule.Type != 22 {
		return rejected("activity has no journey battle")
	}
	if _, err := a.JourneyActivityRule(r, id, rule.Type, now); err != nil {
		return err
	}
	activity := a.Journey.Activities[id]
	if rule.Type == 20 {
		current, started := activity.Missions[mission]
		entry, known := r.Journey.MissionNodes[node]
		if !started || current.Node != node || !known || entry.Type != 1 || model.BattleID(entry.Parameter) != battle {
			return rejected("limited event battle does not match active story node")
		}
		return nil
	}
	entry, known := r.Journey.HyperBattles[node]
	if !known || !slices.Contains(entry.Battles, battle) {
		return rejected("hyperflow battle does not match node")
	}
	for _, state := range activity.BattleNodes {
		if state.ID != node {
			continue
		}
		if !state.Open {
			return rejected("hyperflow node is locked")
		}
		if len(entry.Battles) > 1 {
			if battle == state.First.ID && state.First.Stars > 0 {
				return rejected("double battle must reset before repeating")
			}
			if battle == state.Second.ID && (state.First.Stars == 0 || state.Second.Stars > 0) {
				return rejected("double battle order is invalid")
			}
		}
		return nil
	}
	return rejected("hyperflow battle does not belong to activity")
}

func (a *Account) ValidateJourneyActivityHeroes(r *Rules, id model.ActivityID, node int, battle model.BattleID, heroes []model.HeroID) error {
	if r.Journey.Activities[id].Type != 22 {
		return nil
	}
	for _, state := range a.Journey.Activities[id].BattleNodes {
		if state.ID != node || state.Second.ID != battle {
			continue
		}
		for _, heroID := range heroes {
			if slices.Contains(state.First.Heroes, heroID) {
				return rejected("hero already deployed in first double battle")
			}
			hero := a.Heroes[heroID]
			for _, instance := range hero.Artifacts {
				if instance != "" && slices.Contains(state.BannedArtifacts, instance) {
					return rejected("artifact already deployed in first double battle")
				}
			}
			for _, instance := range hero.Highlights {
				if instance != "" && slices.Contains(state.BannedHighlights, instance) {
					return rejected("highlight already deployed in first double battle")
				}
			}
		}
	}
	return nil
}

func (a *Account) FinishJourneyActivityBattle(r *Rules, id model.ActivityID, mission, node int, battle model.BattleID, won bool, bitmap int, heroes []model.HeroID, now int64) (map[model.ItemID]int64, error) {
	if err := a.ValidateJourneyActivityBattle(r, id, mission, node, battle, now); err != nil {
		return nil, err
	}
	if bitmap < 0 || bitmap > 7 {
		return nil, rejected("invalid journey battle stars")
	}
	if !won {
		return map[model.ItemID]int64{}, nil
	}
	if err := a.ValidateJourneyActivityHeroes(r, id, node, battle, heroes); err != nil {
		return nil, err
	}
	activity := a.Journey.Activities[id]
	if r.Journey.Activities[id].Type == 20 {
		state := activity.Missions[mission]
		rule := r.Journey.MissionNodes[node]
		if rule.Next == 0 {
			return nil, rejected("limited story battle has no following node")
		}
		state.Nodes, state.Passed = maps.Clone(state.Nodes), maps.Clone(state.Passed)
		enterJourneyMissionNode(&state, node)
		state.Passed[node] = 1
		enterJourneyMissionNode(&state, rule.Next)
		activity.Missions = maps.Clone(activity.Missions)
		activity.Missions[mission] = state
		a.Journey.Activities[id] = activity
		a.Journey.Novice.BattleWins[battle] = true
		return map[model.ItemID]int64{}, nil
	}
	entry := r.Journey.HyperBattles[node]
	nodes := slices.Clone(activity.BattleNodes)
	var rewards []model.RewardID
	unlock := 0
	for index, state := range nodes {
		if state.ID != node {
			continue
		}
		result := model.JourneyHyperBattle{ID: battle, Stars: bitmap, Heroes: slices.Clone(heroes), Died: []model.HeroID{}}
		current := bitmap
		if state.First.ID == battle {
			state.First = result
		} else {
			state.Second = result
		}
		if len(entry.Battles) > 1 {
			current = state.First.Stars & state.Second.Stars
		}
		state.Stars |= current
		for star, items := range entry.Rewards {
			if current&(1<<star) != 0 && state.Rewarded&(1<<star) == 0 {
				rewards = append(rewards, items...)
				state.Rewarded |= 1 << star
			}
		}
		state.RewardedStars = bits.OnesCount(uint(state.Rewarded))
		if current&1 != 0 {
			unlock = entry.Next
		}
		if len(entry.Battles) > 1 && state.Second.Stars == 0 {
			artifacts, highlights := map[model.InstanceID]bool{}, map[model.InstanceID]bool{}
			for _, heroID := range heroes {
				hero := a.Heroes[heroID]
				for _, instance := range hero.Artifacts {
					if instance != "" {
						artifacts[instance] = true
					}
				}
				for _, instance := range hero.Highlights {
					if instance != "" {
						highlights[instance] = true
					}
				}
			}
			state.BannedArtifacts = slices.Sorted(maps.Keys(artifacts))
			state.BannedHighlights = slices.Sorted(maps.Keys(highlights))
		}
		nodes[index] = state
	}
	if unlock > 0 {
		for index := range nodes {
			if nodes[index].ID == unlock {
				nodes[index].Open = true
			}
		}
	}
	quantities, err := mergeJourneyRewards(r, rewards)
	if err != nil {
		return nil, err
	}
	if err = a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	activity = a.Journey.Activities[id]
	activity.BattleNodes = nodes
	a.Journey.Activities[id] = activity
	return quantities, nil
}

func (a *Account) AddJourneyHyperPoints(r *Rules, amount int64, now int64) error {
	if amount < 0 || amount > math.MaxInt32 {
		return rejected("invalid hyperflow points")
	}
	for id, activity := range a.Journey.Activities {
		if _, err := a.JourneyActivityRule(r, id, 22, now); err != nil {
			continue
		}
		activity.HyperPoint = min(activity.HyperMaximum, activity.HyperPoint+int(amount))
		nodes := slices.Clone(activity.BuffNodes)
		for index := range nodes {
			nodes[index].Active = r.Journey.HyperBuffs[nodes[index].ID].Threshold <= activity.HyperPoint
		}
		activity.BuffNodes = nodes
		a.Journey.Activities[id] = activity
	}
	return nil
}

func (a *Account) RecordJourneyHyperBattleDetails(id model.ActivityID, node int, battle model.BattleID, died []model.HeroID, useTime int64) {
	activity, known := a.Journey.Activities[id]
	if !known {
		return
	}
	nodes := slices.Clone(activity.BattleNodes)
	for index := range nodes {
		if nodes[index].ID != node {
			continue
		}
		switch battle {
		case nodes[index].First.ID:
			nodes[index].First.Died, nodes[index].First.UseTime = slices.Clone(died), useTime
		case nodes[index].Second.ID:
			nodes[index].Second.Died, nodes[index].Second.UseTime = slices.Clone(died), useTime
		default:
			return
		}
		activity.BattleNodes = nodes
		a.Journey.Activities[id] = activity
		return
	}
}
