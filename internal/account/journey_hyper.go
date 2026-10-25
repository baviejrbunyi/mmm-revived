package account

import (
	"slices"

	"mmm-revived/internal/model"
)

func (a *Account) ChooseJourneyBuff(r *Rules, id model.ActivityID, node, choice int, now int64) error {
	if _, err := a.JourneyActivityRule(r, id, 22, now); err != nil {
		return err
	}
	rule, known := r.Journey.HyperBuffs[node]
	if !known || rule.Type != 2 || !slices.Contains(rule.Choices, choice) {
		return rejected("invalid hyperflow buff choice")
	}
	activity := a.Journey.Activities[id]
	nodes := slices.Clone(activity.BuffNodes)
	for index, entry := range nodes {
		if entry.ID != node {
			continue
		}
		if !entry.Active || activity.HyperPoint < rule.Threshold || entry.Choice != 0 && entry.Choice != choice {
			return rejected("hyperflow buff is locked or already selected")
		}
		entry.Choice = choice
		nodes[index] = entry
		activity.BuffNodes = nodes
		a.Journey.Activities[id] = activity
		return nil
	}
	return rejected("hyperflow buff does not belong to activity")
}

func (a *Account) ResetJourneyBuffs(r *Rules, id model.ActivityID, now int64) error {
	if _, err := a.JourneyActivityRule(r, id, 22, now); err != nil {
		return err
	}
	activity := a.Journey.Activities[id]
	changed := false
	nodes := slices.Clone(activity.BuffNodes)
	for index, entry := range nodes {
		if r.Journey.HyperBuffs[entry.ID].Type == 2 && entry.Choice != 0 {
			entry.Choice = 0
			nodes[index] = entry
			changed = true
		}
	}
	if !changed {
		return nil
	}
	limits := r.Constants.Constants
	cost := min(limits.HyperflowResetBase+limits.HyperflowResetIncrement*int64(activity.BuffReset), limits.HyperflowResetMaximum)
	if err := a.Spend(map[model.ItemID]int64{r.Constants.Items.Gold: cost}); err != nil {
		return err
	}
	activity.BuffReset++
	activity.BuffNodes = nodes
	a.Journey.Activities[id] = activity
	return nil
}

func (a *Account) ResetJourneyHyperBattle(r *Rules, id model.ActivityID, node int, now int64) error {
	if _, err := a.JourneyActivityRule(r, id, 22, now); err != nil {
		return err
	}
	rule, known := r.Journey.HyperBattles[node]
	if !known || len(rule.Battles) != 2 {
		return rejected("hyperflow node is not a double battle")
	}
	activity := a.Journey.Activities[id]
	nodes := slices.Clone(activity.BattleNodes)
	for index, entry := range nodes {
		if entry.ID != node {
			continue
		}
		if !entry.Open {
			return rejected("hyperflow battle node is locked")
		}
		entry.First = model.JourneyHyperBattle{ID: rule.Battles[0]}
		entry.Second = model.JourneyHyperBattle{ID: rule.Battles[1]}
		entry.BannedArtifacts = nil
		entry.BannedHighlights = nil
		nodes[index] = entry
		activity.BattleNodes = nodes
		a.Journey.Activities[id] = activity
		return nil
	}
	return rejected("hyperflow node does not belong to activity")
}
