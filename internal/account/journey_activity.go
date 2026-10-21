package account

import (
	"maps"
	"math"
	"math/bits"
	"slices"

	"mmm-revived/internal/model"
)

func (a *Account) ClaimJourneySign(r *Rules, id model.ActivityID, days []int, now int64) (map[model.ItemID]int64, error) {
	rule, err := a.JourneyActivityRule(r, id, 24, now)
	if err != nil {
		return nil, err
	}
	activity := a.Journey.Activities[id]
	var rewards []model.RewardID
	claimed := slices.Clone(activity.SignRewards)
	seen := make(map[int]bool, len(days))
	for _, day := range days {
		reward, known := rule.Rewards[day]
		if !known || day < 1 || day > activity.Signed || seen[day] {
			return nil, rejected("sign reward unavailable")
		}
		seen[day] = true
		if !slices.Contains(claimed, day) {
			claimed = append(claimed, day)
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
	activity.SignRewards = claimed
	a.Journey.Activities[id] = activity
	return quantities, nil
}

func (a *Account) ClaimJourneyTrailers(r *Rules, id model.ActivityID, indices []int, now int64) (map[model.ItemID]int64, error) {
	if _, err := a.JourneyActivityRule(r, id, 30, now); err != nil {
		return nil, err
	}
	activity := a.Journey.Activities[id]
	claimed := maps.Clone(activity.TrailerRewards)
	var rewards []model.RewardID
	seen := make(map[int]bool, len(indices))
	for _, index := range indices {
		rule, known := r.Journey.Trailers[index]
		if _, unlocked := activity.TrailerRewards[index]; !known || seen[index] || !unlocked {
			return nil, rejected("gameplay preview reward unavailable")
		}
		seen[index] = true
		if claimed[index] != 1 {
			claimed[index] = 1
			rewards = append(rewards, rule.Reward)
		}
	}
	quantities, err := mergeJourneyRewards(r, rewards)
	if err != nil {
		return nil, err
	}
	if err = a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	activity.TrailerRewards = claimed
	a.Journey.Activities[id] = activity
	return quantities, nil
}

func (a *Account) StartDreamguard(r *Rules, id model.ActivityID, now int64) error {
	if _, err := a.JourneyActivityRule(r, id, 18, now); err != nil {
		return err
	}
	activity := a.Journey.Activities[id]
	if activity.TaskStart != 0 {
		return nil
	}
	activity.TaskStart = now
	a.Journey.Activities[id] = activity
	if a.Tasks[r.Journey.DreamguardTask].Status == 0 {
		a.Tasks[r.Journey.DreamguardTask] = model.Task{Status: 2}
	}
	return nil
}

func (a *Account) MakeDreamguardPermanent(r *Rules, id model.ActivityID, now int64) error {
	if _, err := a.JourneyActivityRule(r, id, 18, now); err != nil {
		return err
	}
	activity := a.Journey.Activities[id]
	if activity.TaskStart == 0 || activity.TaskClaimed {
		return rejected("dreamguard task is unavailable")
	}
	if activity.TaskForever == 1 {
		return nil
	}
	if err := a.Spend(r.Journey.DreamguardPrice); err != nil {
		return err
	}
	activity.TaskForever = 1
	a.Journey.Activities[id] = activity
	return nil
}

func (a *Account) ClaimDreamguard(r *Rules, id model.ActivityID, now int64) (map[model.ItemID]int64, error) {
	if _, err := a.JourneyActivityRule(r, id, 18, now); err != nil {
		return nil, err
	}
	activity := a.Journey.Activities[id]
	if activity.TaskClaimed {
		return map[model.ItemID]int64{}, nil
	}
	if activity.TaskStart == 0 || activity.TaskForever != 1 && now > activity.TaskStart+r.Journey.DreamguardDuration || a.Tasks[r.Journey.DreamguardTask].Status < 3 {
		return nil, rejected("dreamguard task is incomplete or expired")
	}
	quantities, err := rewardItems(r, r.Journey.DreamguardReward, 1)
	if err != nil {
		return nil, err
	}
	if err = a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	task := a.Tasks[r.Journey.DreamguardTask]
	task.Status = 4
	a.Tasks[r.Journey.DreamguardTask] = task
	activity.TaskClaimed = true
	a.Journey.Activities[id] = activity
	return quantities, nil
}

func (a *Account) ClaimJourneyStars(r *Rules, id model.ActivityID, stars []int, now int64) (map[model.ItemID]int64, error) {
	rule, err := a.JourneyActivityRule(r, id, 26, now)
	if err != nil {
		return nil, err
	}
	group, known := r.Journey.SoloGroups[rule.Parameter]
	if !known {
		return nil, rejected("solo challenge group is unavailable")
	}
	activity := a.Journey.Activities[id]
	total := 0
	for _, node := range group.Nodes {
		total += bits.OnesCount(uint(activity.NodeStars[node] & 7))
	}
	var rewards []model.RewardID
	claimed := slices.Clone(activity.StarRewards)
	seen := make(map[int]bool, len(stars))
	for _, star := range stars {
		reward, known := group.Rewards[star]
		if !known || star > total || seen[star] {
			return nil, rejected("solo star reward unavailable")
		}
		seen[star] = true
		if !slices.Contains(claimed, star) {
			claimed = append(claimed, star)
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
	activity.StarRewards = claimed
	a.Journey.Activities[id] = activity
	return quantities, nil
}

func (a *Account) ClaimJourneyBrawl(r *Rules, id model.ActivityID, index int, now int64) (map[model.ItemID]int64, error) {
	rule, err := a.JourneyActivityRule(r, id, 10, now)
	if err != nil {
		return nil, err
	}
	threshold, known := rule.Thresholds[index]
	reward, rewardKnown := rule.Rewards[index]
	activity := a.Journey.Activities[id]
	if !known || !rewardKnown || activity.Star < threshold {
		return nil, rejected("brawl reward unavailable")
	}
	if slices.Contains(activity.Rewards, index) {
		return map[model.ItemID]int64{}, nil
	}
	quantities, err := rewardItems(r, reward, 1)
	if err != nil {
		return nil, err
	}
	if err = a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	activity.Rewards = append(activity.Rewards, index)
	a.Journey.Activities[id] = activity
	return quantities, nil
}

func (a *Account) ClaimLimitedJourneyPass(r *Rules, id model.ActivityID, levels []int, active int, now int64) (map[model.ItemID]int64, error) {
	if _, err := a.JourneyActivityRule(r, id, 20, now); err != nil {
		return nil, err
	}
	activity := a.Journey.Activities[id]
	rule, known := r.Journey.LimitedPasses[activity.Pass.ID]
	if !known {
		return nil, ErrJourneyPassUnavailable
	}
	quantities, err := claimJourneyPass(a, r, &activity.Pass, rule, levels, active)
	if err != nil {
		return nil, err
	}
	a.Journey.Activities[id] = activity
	return quantities, nil
}

func (a *Account) FinishJourneyMiniGame(r *Rules, id model.ActivityID, level int, now int64) (map[model.ItemID]int64, error) {
	if _, err := a.JourneyActivityRule(r, id, 20, now); err != nil {
		return nil, err
	}
	rule, known := r.Journey.MiniGames[level]
	activity := a.Journey.Activities[id]
	_, previous := activity.MiniGames[rule.Predecessor]
	if !known || a.Level < rule.Level || now+r.Journey.CalendarOffset < rule.Start || r.Journey.MiniGameUnlockMission > 0 && activity.Missions[r.Journey.MiniGameUnlockMission].Finished != 1 || rule.Predecessor > 0 && !previous {
		return nil, rejected("minigame level is unavailable")
	}
	if _, finished := activity.MiniGames[level]; finished {
		return map[model.ItemID]int64{}, nil
	}
	quantities, err := rewardItems(r, rule.Reward, 1)
	if err != nil {
		return nil, err
	}
	if err = a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	if activity.MiniGames == nil {
		activity.MiniGames = map[int]int64{}
	}
	activity.MiniGames[level] = now
	a.Journey.Activities[id] = activity
	return quantities, nil
}

func (a *Account) StartJourneyMissionNode(r *Rules, id model.ActivityID, mission, node int, now int64) error {
	if _, err := a.JourneyActivityRule(r, id, 20, now); err != nil {
		return err
	}
	rule, known := r.Journey.Missions[mission]
	if !known || a.Level < rule.Level {
		return rejected("limited mission unavailable")
	}
	activity := a.Journey.Activities[id]
	state, started := activity.Missions[mission]
	if !started {
		for _, previous := range rule.Predecessors {
			if activity.Missions[previous].Finished != 1 {
				return rejected("limited mission predecessor is incomplete")
			}
		}
		state = model.JourneyMission{Node: rule.FirstNode}
	}
	if node != state.Node && node != rule.FirstNode {
		return rejected("invalid limited mission node")
	}
	enterJourneyMissionNode(&state, node)
	if activity.Missions == nil {
		activity.Missions = map[int]model.JourneyMission{}
	}
	activity.Missions[mission] = state
	a.Journey.Activities[id] = activity
	return nil
}

func enterJourneyMissionNode(state *model.JourneyMission, node int) {
	if state.Nodes == nil {
		state.Nodes = map[int]model.MissionNode{}
	}
	if state.Passed == nil {
		state.Passed = map[int]int{}
	}
	state.Node = node
	if current, exists := state.Nodes[node]; !exists || current.ID == 0 {
		current.ID = model.NodeID(node)
		state.Nodes[node] = current
	}
}

func (a *Account) FinishJourneyMissionNode(r *Rules, id model.ActivityID, mission, node int, now int64) (map[model.ItemID]int64, error) {
	if _, err := a.JourneyActivityRule(r, id, 20, now); err != nil {
		return nil, err
	}
	activity := a.Journey.Activities[id]
	state, started := activity.Missions[mission]
	rule, known := r.Journey.MissionNodes[node]
	if !known || !started || state.Node != node {
		return nil, rejected("limited mission node unavailable")
	}
	if rule.Type == 1 && !a.Journey.Novice.BattleWins[model.BattleID(rule.Parameter)] {
		return nil, rejected("limited mission battle is incomplete")
	}
	quantities := map[model.ItemID]int64{}
	state.Nodes, state.Passed = maps.Clone(state.Nodes), maps.Clone(state.Passed)
	enterJourneyMissionNode(&state, node)
	if rule.Type == 2 && state.Finished == 0 {
		var err error
		if quantities, err = rewardItems(r, r.Journey.Missions[mission].Reward, 1); err != nil {
			return nil, err
		}
		if err = a.GrantItems(r, quantities); err != nil {
			return nil, err
		}
		state.Finished = 1
		current := state.Nodes[node]
		current.Rewarded = 1
		state.Nodes[node] = current
	}
	state.Passed[node] = 1
	if rule.Next != 0 {
		enterJourneyMissionNode(&state, rule.Next)
	}
	activity.Missions = maps.Clone(activity.Missions)
	activity.Missions[mission] = state
	a.Journey.Activities[id] = activity
	return quantities, nil
}

func (a *Account) CompleteJourneyBattle(r *Rules, battle model.BattleID, won bool, bitmap int, now int64) {
	if !won {
		return
	}
	a.Journey.Novice.BattleWins[battle] = true
	for id, rule := range r.Journey.Activities {
		if rule.Type != 26 || !a.journeyActivityAvailable(rule, now, r.Journey.CalendarOffset) {
			continue
		}
		activity, known := a.Journey.Activities[id]
		if !known {
			continue
		}
		for _, node := range r.Journey.SoloGroups[rule.Parameter].Nodes {
			if r.Journey.SoloNodes[node] == battle {
				activity.NodeStars[node] |= bitmap & 7
			}
		}
		a.Journey.Activities[id] = activity
	}
}

func (a *Account) RecordJourneyBrawl(r *Rules, id model.ActivityID, battle model.BattleID, score int, now int64) error {
	rule, err := a.JourneyActivityRule(r, id, 10, now)
	if err != nil {
		return err
	}
	if !slices.Contains(rule.Battles, battle) || score < 0 || score > 100000 {
		return rejected("invalid brawl score")
	}
	activity := a.Journey.Activities[id]
	activity.Star = max(activity.Star, score)
	activity.BestScore = max(activity.BestScore, score)
	a.Journey.Activities[id] = activity
	return nil
}

func (a *Account) AddLimitedJourneyPassExperience(r *Rules, amount int64, now int64) error {
	if amount < 0 {
		return rejected("invalid limited pass experience")
	}
	for id, activity := range a.Journey.Activities {
		if _, err := a.JourneyActivityRule(r, id, 20, now); err != nil {
			continue
		}
		rule, known := r.Journey.LimitedPasses[activity.Pass.ID]
		if !known || rule.ExperiencePerLevel <= 0 {
			continue
		}
		if amount > math.MaxInt64-activity.Pass.Experience {
			return rejected("limited pass experience overflow")
		}
		activity.Pass.Experience += amount
		levelJourneyPass(&activity.Pass, rule)
		a.Journey.Activities[id] = activity
	}
	return nil
}
