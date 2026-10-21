package account

import (
	"context"
	"database/sql"
	"maps"
	"slices"

	"mmm-revived/internal/database"
	"mmm-revived/internal/model"
)

func (a *Account) PrepareNetwork(r NetworkRules, now int64) {
	n := &a.Network
	if !n.Initialized {
		n.Initialized = true
		n.Score, n.MaxScore = 1000, 1000
		n.Activity = r.DefaultActivity()
		n.Day = -1
	}
	if n.Activity == 0 {
		n.Activity = r.DefaultActivity()
	}
	if day := now / 86400; n.Day != day {
		n.Day = day
		n.Attempts = r.Activities[n.Activity].Attempts
		n.Refreshed = 1
	}
	if n.Defense == nil {
		n.Defense = map[model.HeroID]int{}
	}
	if n.Interactions == nil {
		n.Interactions = map[int][]int{}
	}
	if n.Friends == nil {
		n.Friends = map[model.UserID]model.AsyncTarget{}
	}
	if n.Support.Day != n.Day {
		n.Support = model.SupportSelection{Day: n.Day}
	}
	if a.Tasks != nil {
		for _, id := range r.Activities[n.Activity].Tasks {
			if task := a.Tasks[id]; task.Status == 0 {
				task.Status = 2
				a.Tasks[id] = task
			}
		}
	}
}

func (a *Account) NetworkSnapshot(now int64) model.PVPComponent {
	n := &a.Network
	score, maxScore := n.Score, n.MaxScore
	if !n.Initialized {
		score, maxScore = 1000, 1000
	}
	return model.PVPComponent{Score: score, MaxScore: maxScore, Rank: int(n.Activity), Activity: n.Activity, Environment: n.Environment, Camp: n.Camp, Info: networkPositions(n.Defense), InfoList: n.DefenseOrder, Targets: n.Targets, Friends: n.Friends, Rewards: n.Rewarded, Interactions: networkInteractions(n.Interactions), Attempts: n.Attempts, Refreshed: n.Refreshed, LastRequest: n.LastRequest}
}

func (a *Account) NetworkDefense(r *Rules) model.AsyncDefense {
	n := &a.Network
	score := n.Score
	if !n.Initialized {
		score = 1000
	}
	level, _ := r.Network.Rank(score)
	info := model.AsyncHeroInfo{Heroes: make(map[model.HeroID]model.Hero, len(n.Defense)), EquippedGear: make(map[model.HeroID]map[int]model.Artifact, len(n.Defense)), Equipped: make(map[model.HeroID]model.Highlight, len(n.Defense)), Bonds: a.BondAttributes(r)}
	for id := range n.Defense {
		hero, owned := a.Heroes[id]
		if !owned {
			continue
		}
		artifacts := make(map[int]model.Artifact, len(hero.Artifacts))
		for slot, instance := range hero.Artifacts {
			if item, ok := a.Artifacts[instance]; ok && instance != "" {
				artifacts[slot] = item
			}
		}
		info.Heroes[id], info.EquippedGear[id] = hero, artifacts
		if item, ok := a.Highlights[hero.Highlights[1]]; ok && hero.Highlights[1] != "" {
			info.Equipped[id] = item
		}
	}
	return model.AsyncDefense{UID: int64(a.UID), PVPLevel: level, WinFix: 1, LoseFix: 1, Positions: networkPositions(n.Defense), Order: n.DefenseOrder, Interactions: networkInteractions(n.Interactions), Environment: n.Environment, Camp: n.Camp, CampDisplay: n.Camp, HeroInfo: info}
}

func (a *Account) SetNetworkDefense(r NetworkRules, heroes map[model.HeroID]int, heroOrder []model.HeroID, interactions map[int][]int, environment, camp int) error {
	if len(heroes) == 0 || len(heroes) > 3 || len(interactions) > 6 {
		return rejected("invalid defense size")
	}
	slots := make(map[int]bool, len(heroes))
	for hero, slot := range heroes {
		if _, ok := a.Heroes[hero]; !ok || slot < 0 || slot > 65535 || slots[slot] {
			return rejected("invalid defense hero or position")
		}
		slots[slot] = true
	}
	objects := 0
	for id, positions := range interactions {
		if id <= 0 || len(positions) > 6 {
			return rejected("invalid defense interaction")
		}
		for _, slot := range positions {
			if slot < 0 || slot > 65535 || slots[slot] {
				return rejected("invalid defense interaction position")
			}
			slots[slot] = true
			objects++
		}
	}
	if objects > 6 {
		return rejected("too many defense interactions")
	}
	if len(heroOrder) != len(heroes) {
		return rejected("invalid defense order")
	}
	seen := make(map[model.HeroID]bool, len(heroOrder))
	for _, id := range heroOrder {
		if _, exists := heroes[id]; !exists || seen[id] {
			return rejected("invalid defense order")
		}
		seen[id] = true
	}
	rule := r.Activities[a.Network.Activity]
	if environment != 0 && !slices.Contains(rule.Environments, environment) || camp != 0 && !slices.Contains(rule.Camps, camp) {
		return rejected("defense environment unavailable")
	}
	a.Network.Defense, a.Network.DefenseOrder = maps.Clone(heroes), slices.Clone(heroOrder)
	a.Network.Interactions = make(map[int][]int, len(interactions))
	for id, positions := range interactions {
		a.Network.Interactions[id] = slices.Clone(positions)
	}
	a.Network.Environment, a.Network.Camp = environment, camp
	return nil
}

func (a *Account) ClaimNetworkRanks(r *Rules, ranks []int) (map[model.ItemID]int64, error) {
	if len(ranks) == 0 {
		return nil, rejected("empty rank reward selection")
	}
	seen := make(map[int]bool, len(ranks))
	quantities := map[model.ItemID]int64{}
	for _, id := range ranks {
		rule, ok := r.Network.Ranks[id]
		if !ok || seen[id] || a.Network.MaxScore < rule.Minimum {
			return nil, rejected("rank reward unavailable")
		}
		seen[id] = true
		if rule.Reward == 0 || slices.Contains(a.Network.Rewarded, id) {
			continue
		}
		items, err := rewardItems(r, rule.Reward, 1)
		if err != nil {
			return nil, err
		}
		addItems(quantities, items)
	}
	if err := a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	for _, id := range ranks {
		if !slices.Contains(a.Network.Rewarded, id) {
			a.Network.Rewarded = append(a.Network.Rewarded, id)
		}
	}
	return quantities, nil
}

func (a *Account) ClaimNetworkTasks(r *Rules, activity model.ActivityID, tasks []int) (map[model.ItemID]int64, error) {
	if activity != a.Network.Activity || len(tasks) == 0 {
		return nil, rejected("async activity unavailable")
	}
	allowed := r.Network.Activities[activity].Tasks
	quantities := map[model.ItemID]int64{}
	seen := make(map[model.TaskID]bool, len(tasks))
	for _, value := range tasks {
		id := model.TaskID(value)
		rule, ok := r.Tasks[id]
		if !ok || !slices.Contains(allowed, id) || seen[id] || a.Network.Points < int64(rule.Target) {
			return nil, rejected("async point reward unavailable")
		}
		seen[id] = true
		if slices.Contains(a.Network.RewardedTasks, id) {
			continue
		}
		items, err := rewardItems(r, rule.Reward, 1)
		if err != nil {
			return nil, err
		}
		addItems(quantities, items)
	}
	if err := a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	for _, value := range tasks {
		id := model.TaskID(value)
		if !slices.Contains(a.Network.RewardedTasks, id) {
			a.Network.RewardedTasks = append(a.Network.RewardedTasks, id)
		}
		task := a.Tasks[id]
		task.Status = 4
		a.Tasks[id] = task
	}
	return quantities, nil
}

func (s *Store) ClaimSupport(a *Account, tx *sql.Tx) (int64, error) {
	q, ctx := s.Database.Queries.WithTx(tx), context.Background()
	rows, err := q.ListSupportUsage(ctx, a.UID)
	if err != nil {
		return 0, err
	}
	claimed := map[int64]int64{}
	for _, row := range rows {
		claimed[row.Day] += row.Claimed
	}
	count := int64(0)
	for _, row := range rows {
		amount := min(max(0, 100-claimed[row.Day]), row.Uses-row.Claimed)
		count += amount
		claimed[row.Day] += amount
	}
	count = min(count, 100) * 300
	if err = a.GrantItems(s.Rules, map[model.ItemID]int64{1: count}); err != nil {
		return 0, err
	}
	_, err = q.ClaimSupportUsage(ctx, a.UID)
	return count, err
}

func supportUsage(q *database.Queries, borrower, owner model.UserID, day int64) (int64, error) {
	ctx := context.Background()
	total, err := q.SupportUsageCount(ctx, database.SupportUsageCountParams{Borrower: borrower, Day: day})
	if err != nil {
		return 0, err
	}
	used, err := q.SupportOwnerUsageCount(ctx, database.SupportOwnerUsageCountParams{Owner: owner, Borrower: borrower, Day: day})
	if err != nil {
		return 0, err
	}
	if total >= 10 || used >= 3 {
		return total, rejected("daily support limit reached")
	}
	return total, nil
}

func (s *Store) SupportAvailable(a *Account, owner model.UserID, now int64) error {
	_, err := supportUsage(s.Database.Queries, a.UID, owner, now/86400)
	return err
}

func (s *Store) RecordSupport(a *Account, tx *sql.Tx, now int64) error {
	selected := a.Network.Support
	if selected.Owner == 0 || selected.Guild {
		return nil
	}
	q, day := s.Database.Queries.WithTx(tx), now/86400
	total, err := supportUsage(q, a.UID, selected.Owner, day)
	if err != nil {
		return err
	}
	if err = q.AddSupportUsage(context.Background(), database.AddSupportUsageParams{Owner: selected.Owner, Borrower: a.UID, Hero: selected.Hero, Day: day}); err != nil {
		return err
	}
	a.Network.Support.Used = int(total) + 1
	return nil
}

func networkPositions(values map[model.HeroID]int) map[model.HeroID]model.AsyncPosition {
	result := make(map[model.HeroID]model.AsyncPosition, len(values))
	for id, born := range values {
		result[id] = model.AsyncPosition{Born: born}
	}
	return result
}

func networkInteractions(values map[int][]int) map[int]model.AsyncInteraction {
	result := make(map[int]model.AsyncInteraction, len(values))
	for id, slots := range values {
		result[id] = model.AsyncInteraction{Slots: slots}
	}
	return result
}

func (s *Store) AbandonAsyncBattle(a *Account) {
	pending := a.Network.Pending
	if pending == nil || pending.Guild != 0 {
		return
	}
	if !pending.Friend {
		a.Network.Attempts++
		if maximum := s.Rules.Network.Activities[pending.Activity].Attempts; maximum > 0 {
			a.Network.Attempts = min(a.Network.Attempts, maximum)
		}
	}
	a.Network.Pending = nil
}
