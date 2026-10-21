package account

import (
	"maps"
	"slices"

	"mmm-revived/internal/model"
)

func (a *Account) ClaimJourneyBrawlList(r *Rules, id model.ActivityID, indices []int, now int64) (map[model.ItemID]int64, error) {
	rule, err := a.JourneyActivityRule(r, id, 10, now)
	if err != nil {
		return nil, err
	}
	activity := a.Journey.Activities[id]
	seen := map[int]bool{}
	var rewards []model.RewardID
	claimed := slices.Clone(activity.Rewards)
	if len(indices) > 100 {
		return nil, rejected("invalid brawl reward list")
	}
	for _, index := range indices {
		threshold, known := rule.Thresholds[index]
		reward, rewardKnown := rule.Rewards[index]
		if !known || !rewardKnown || activity.Star < threshold || seen[index] {
			return nil, rejected("brawl reward unavailable")
		}
		seen[index] = true
		if !slices.Contains(claimed, index) {
			claimed = append(claimed, index)
			rewards = append(rewards, reward)
		}
	}
	quantities, err := mergeJourneyRewards(r, rewards)
	if err != nil {
		return nil, err
	}
	if err = a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	activity.Rewards = claimed
	a.Journey.Activities[id] = activity
	return quantities, nil
}

func (a *Account) InviteRewardCounts() map[int]int {
	counts := map[int]int{}
	for _, claimed := range a.Journey.InviteRewards {
		for index, rewarded := range claimed {
			if rewarded {
				counts[index]++
			}
		}
	}
	return counts
}

func (a *Account) ClaimJourneyInviteList(r *Rules, target model.UserID, indices []int, progress model.InviteProgress) (map[model.ItemID]int64, error) {
	if target == 0 || progress.UID != target || len(indices) > 100 {
		return nil, rejected("invalid invited player reward list")
	}
	counts := a.InviteRewardCounts()
	maximum := r.RetailActivity.InviteRewardMaximum
	if maximum <= 0 {
		maximum = 5
	}
	seen := map[int]bool{}
	claimed := maps.Clone(a.Journey.InviteRewards[target])
	if claimed == nil {
		claimed = map[int]bool{}
	}
	var rewards []model.RewardID
	for _, index := range indices {
		reward, known := r.Journey.InviteRewards[index]
		if !known || seen[index] || !journeyInviteEligible(index, progress) {
			return nil, rejected("invited player task is incomplete")
		}
		seen[index] = true
		if claimed[index] {
			continue
		}
		if counts[index] >= maximum {
			return nil, rejected("invitation task reward limit reached")
		}
		claimed[index] = true
		rewards = append(rewards, reward)
	}
	quantities, err := mergeJourneyRewards(r, rewards)
	if err != nil {
		return nil, err
	}
	if err = a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	if a.Journey.InviteRewards == nil {
		a.Journey.InviteRewards = map[model.UserID]map[int]bool{}
	}
	a.Journey.InviteRewards[target] = claimed
	return quantities, nil
}
