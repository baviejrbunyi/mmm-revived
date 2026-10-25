package account

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"math/bits"
	"slices"
	"strconv"
	"time"

	"mmm-revived/internal/model"
)

var ErrProgression = errors.New("progression request rejected")

func rejected(message string) error { return fmt.Errorf("%w: %s", ErrProgression, message) }

func highlightLevelKey(rarity, level int) string {
	return strconv.Itoa(rarity) + "_" + strconv.Itoa(level)
}

func (a *Account) Spend(cost map[model.ItemID]int64) error {
	for id, count := range cost {
		if count < 0 || a.Items[id] < count {
			return rejected("insufficient items")
		}
	}
	spent := maps.Clone(a.Journey.SpentItems)
	if spent == nil {
		spent = map[model.ItemID]int64{}
	}
	now := time.Now().Unix()
	day, week := LogicDay(now), LogicWeek(now)
	daily, weekly := maps.Clone(a.Journey.DailySpent), maps.Clone(a.Journey.WeeklySpent)
	if daily == nil || a.Journey.SpentDay != day {
		daily = map[model.ItemID]int64{}
	}
	if weekly == nil || a.Journey.SpentWeek != week {
		weekly = map[model.ItemID]int64{}
	}
	for id, count := range cost {
		a.Items[id] -= count
		if count > 0 {
			for _, counter := range [...]map[model.ItemID]int64{spent, daily, weekly} {
				counter[id] += min(count, math.MaxInt64-counter[id])
			}
		}
	}
	a.Journey.SpentItems, a.Journey.DailySpent, a.Journey.WeeklySpent = spent, daily, weekly
	a.Journey.SpentDay, a.Journey.SpentWeek = day, week
	return nil
}

func (a *Account) Grant(r *Rules, ids ...model.RewardID) error {
	quantities := map[model.ItemID]int64{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		items, err := rewardItems(r, id, 1)
		if err != nil {
			return err
		}
		for item, amount := range items {
			if !addQuantity(quantities, item, amount) {
				return rejected("reward overflow")
			}
		}
	}
	return a.GrantItems(r, quantities)
}

func (a *Account) UpgradeHero(r *Rules, id model.HeroID, count int) error {
	hero, ok := a.Heroes[id]
	if !ok || count < 1 || count > 100 || hero.Level+count > r.TeamLevels[a.Level].HeroLimit || hero.Trial == 1 && hero.Level+count > r.Constants.Constants.TrialHeroMaxLevel {
		return rejected("invalid hero level")
	}
	breaks := r.Breaks[r.HeroBreakSets[id]]
	cost := map[model.ItemID]int64{}
	for level := hero.Level; level < hero.Level+count; level++ {
		rule, ok := r.Levels[level]
		if !ok {
			return rejected("unknown hero level")
		}
		for tier := hero.Break + 1; tier <= rule.Break; tier++ {
			row, ok := breaks[tier]
			if !ok {
				return rejected("unknown hero breakthrough")
			}
			addItems(cost, row)
		}
		hero.Break = max(hero.Break, rule.Break)
		cost[r.Constants.Items.HeroExperience] += rule.Experience
		cost[r.Constants.Items.Gold] += rule.Money
	}
	if err := a.Spend(cost); err != nil {
		return err
	}
	hero.Level += count
	hero.Experience = 0
	a.Heroes[id] = hero
	a.RefreshTasks(r)
	return nil
}

func (a *Account) UpgradeHighlight(r *Rules, id model.InstanceID, count int) error {
	highlight, ok := a.Highlights[id]
	if !ok || count < 1 || count > 100 || highlight.Level+count > a.Level {
		return rejected("invalid highlight level")
	}
	template, ok := r.HighlightTemplates[highlight.ID]
	if !ok {
		return rejected("unknown highlight template")
	}
	breaks := r.HighlightBreaks[template.BreakSet]
	cost := map[model.ItemID]int64{}
	for level := highlight.Level; level < highlight.Level+count; level++ {
		rule, ok := r.HighlightLevels[highlightLevelKey(template.Rarity, level)]
		if !ok {
			return rejected("unknown highlight level")
		}
		for tier := highlight.Break + 1; tier <= rule.Break; tier++ {
			row, ok := breaks[tier]
			if !ok {
				return rejected("unknown highlight breakthrough")
			}
			addItems(cost, row)
		}
		highlight.Break = max(highlight.Break, rule.Break)
		cost[r.Constants.Items.Gold] += rule.Money
	}
	if err := a.Spend(cost); err != nil {
		return err
	}
	highlight.Level += count
	a.Highlights[id] = highlight
	return nil
}

func (a *Account) BeginMission(r *Rules, chapter model.ChapterID, id model.MissionID) error {
	rule, ok := r.Missions[id]
	if !ok || rule.Chapter != chapter || a.Level < rule.Level {
		return rejected("mission is locked")
	}
	for _, required := range rule.Prerequisites {
		if a.Passed[required].Stars == 0 {
			return rejected("prerequisite mission is incomplete")
		}
	}
	node, ok := r.Nodes[rule.FirstNode]
	if !ok {
		return rejected("missing initial mission node")
	}
	mission := model.Mission{ID: id, Node: model.MissionNode{ID: rule.FirstNode}, Heroes: map[model.HeroID]int{}, Visited: map[model.MementoChipID]int{}, LokiRecord: map[model.MementoChipID]int{}, TeamSteps: map[int]int{}}
	if node.Type == 5 {
		tiles := r.MementoMaps[node.Map]
		if len(tiles) == 0 {
			return rejected("Memento map is missing")
		}
		initializeMemento(&mission, rule, tiles)
	}
	if err := a.Spend(map[model.ItemID]int64{r.Constants.Items.Energy: rule.Energy}); err != nil {
		return err
	}
	if err := a.grantEnergyExperience(r, rule.Energy); err != nil {
		return err
	}
	a.Missions[id] = mission
	a.CurrentMission = id
	if rule.Memento {
		a.CurrentMemento = id
	} else {
		a.CurrentStory = id
	}
	return nil
}

func (a *Account) AdvanceMission(r *Rules, id model.MissionID) error {
	mission, ok := a.Missions[id]
	if !ok {
		return rejected("mission is not active")
	}
	progress := a.Passed[id]
	if mission.Node.ID == 0 && progress.Stars > 0 && progress.Rewarded == 1 {
		return nil
	}
	if finishedReward(r, id, mission) {
		mission.Node = model.MissionNode{}
		a.Missions[id] = mission
		a.leaveMission(id)
		return nil
	}
	if !a.MissionActive(r, id) {
		return rejected("mission is not active")
	}
	node, ok := r.Nodes[mission.Node.ID]
	if !ok || node.Type == 1 || node.Type == 5 {
		return rejected("mission node cannot advance")
	}
	if node.Type == 2 {
		if progress.Stars == 0 || progress.Rewarded != 1 {
			return rejected("mission reward is not claimed")
		}
	} else if node.Next == 0 {
		mission.Node = model.MissionNode{}
		a.Missions[id] = mission
		a.leaveMission(id)
		return nil
	}
	mission.Node = model.MissionNode{ID: node.Next}
	a.Missions[id] = mission
	if node.Next == 0 {
		a.leaveMission(id)
	}
	return nil
}

func (a *Account) leaveMission(id model.MissionID) {
	if a.CurrentStory == id {
		a.CurrentStory = 0
	}
	if a.CurrentMemento == id {
		a.CurrentMemento = 0
	}
	if a.CurrentMission == id {
		a.CurrentMission = 0
	}
}

func (a *Account) FinishMissionBattle(r *Rules, id model.MissionID, stars int) error {
	mission, ok := a.Missions[id]
	if !ok || !a.MissionActive(r, id) || stars < 1 || stars > 7 {
		return rejected("invalid mission battle result")
	}
	node, ok := r.Nodes[mission.Node.ID]
	if !ok || node.Type != 1 || node.Next == 0 {
		return rejected("mission has no active battle")
	}
	mission.Node = model.MissionNode{ID: node.Next}
	a.Missions[id] = mission
	progress := a.Passed[id]
	progress.Stars |= stars
	a.Passed[id] = progress
	return nil
}

func finishedReward(r *Rules, id model.MissionID, mission model.Mission) bool {
	node, ok := r.Nodes[mission.Node.ID]
	return ok && node.Type == 2 && node.Next == 0 && mission.Node.Rewarded == 1
}

func (a *Account) ClaimMission(r *Rules, id model.MissionID) error {
	mission, ok := a.Missions[id]
	if ok && finishedReward(r, id, mission) {
		return nil
	}
	if !ok || !a.MissionActive(r, id) {
		return rejected("mission is not active")
	}
	node, ok := r.Nodes[mission.Node.ID]
	if !ok || node.Type != 2 {
		return rejected("mission reward is not ready")
	}
	if mission.Node.Rewarded == 1 {
		return nil
	}
	progress := a.Passed[id]
	if progress.Rewarded == 0 {
		if err := a.Grant(r, r.Missions[id].Reward); err != nil {
			return err
		}
		progress.Rewarded = 1
	}
	if progress.Stars == 0 {
		progress.Stars = 1
	}
	mission.Node.Rewarded = 1
	a.Missions[id], a.Passed[id] = mission, progress
	if node.Next == 0 {
		a.leaveMission(id)
	}
	a.RefreshTasks(r)
	return nil
}

func (a *Account) RefreshTasks(r *Rules) {
	a.refreshJourneyTaskValues(r)
}

func (a *Account) ClaimTasks(r *Rules, ids []model.TaskID) error {
	return a.claimJourneyTasksNow(r, ids)
}

func (a *Account) ClaimLevels(r *Rules, levels []int) error {
	rewards := make([]model.RewardID, 0, len(levels))
	for _, level := range levels {
		rule, ok := r.TeamLevels[level]
		if !ok || rule.Reward == 0 || level > a.Level || slices.Contains(a.RewardedLevels, level) {
			return rejected("level reward unavailable")
		}
		rewards = append(rewards, rule.Reward)
	}
	if err := a.Grant(r, rewards...); err != nil {
		return err
	}
	a.RewardedLevels = append(a.RewardedLevels, levels...)
	return nil
}

func (a *Account) ClaimChapter(r *Rules, id model.ChapterID, stars []int) error {
	rule, ok := r.Chapters[id]
	if !ok {
		return rejected("unknown chapter")
	}
	total := 0
	for _, mission := range rule.Missions {
		if bitmap := a.Passed[mission].Stars; bitmap > 0 {
			total += bits.OnesCount(uint(bitmap))
		}
	}
	progress := a.Chapters[id]
	rewards := make([]model.RewardID, 0, len(stars))
	for _, star := range stars {
		reward, ok := rule.Rewards[star]
		if !ok || total < star || slices.Contains(progress.Rewards, star) {
			return rejected("chapter reward unavailable")
		}
		rewards = append(rewards, reward)
	}
	if err := a.Grant(r, rewards...); err != nil {
		return err
	}
	progress.Rewards = append(progress.Rewards, stars...)
	a.Chapters[id] = progress
	return nil
}
