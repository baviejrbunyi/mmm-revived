package account

import (
	"maps"
	"math"
	"slices"

	"mmm-revived/internal/model"
)

const ()

type EventRules struct {
	CalendarOffset int64                   `json:"calendar_offset"`
	Science        map[int]ScienceRule     `json:"science"`
	SeasonLevels   map[int]map[int]int64   `json:"season_levels"`
	BossOpens      map[int]BossOpenRule    `json:"boss_opens"`
	Bosses         map[int]BossRule        `json:"bosses"`
	BossMissions   map[int]BossMissionRule `json:"boss_missions"`
	BossScores     map[int]model.RewardID  `json:"boss_scores"`
	BossRanks      map[int]BossRankRule    `json:"boss_ranks"`
}

type ScienceRule struct {
	Next    []int `json:"next"`
	Choices []int `json:"choices"`
}
type BossOpenRule struct {
	Start  int64 `json:"start"`
	End    int64 `json:"end"`
	Settle int64 `json:"settle"`
	Event  int   `json:"event"`
}
type BossRule struct {
	Monster  model.HeroID `json:"monster"`
	Missions []int        `json:"missions"`
}
type BossMissionRule struct {
	Battle model.BattleID `json:"battle"`
	Base   int64          `json:"base"`
	Time   int64          `json:"time"`
	Limit  int64          `json:"limit"`
	Reward model.RewardID `json:"reward"`
}
type BossRankRule struct {
	Minimum    int64          `json:"minimum"`
	Maximum    int64          `json:"maximum"`
	MinPercent bool           `json:"min_percent"`
	MaxPercent bool           `json:"max_percent"`
	Reward     model.RewardID `json:"reward"`
}

func (b BossRankRule) Matches(position, total int64, settings BossSettings) bool {
	value, percent := position, false
	if position > settings.RankDisplay {
		value, percent = min(100, (position-1)*100/max(total, settings.RankBoardMinimum)), true
	}
	if b.MinPercent {
		if !percent || value < b.Minimum {
			return false
		}
	} else if !percent && value < b.Minimum {
		return false
	}
	if b.MaxPercent {
		return !percent || value <= b.Maximum
	}
	return !percent && value <= b.Maximum
}

func (a *Account) InitializeEvents(r *Rules, now int64) {
	state := &a.Events
	week, day := LogicWeek(now), LogicDay(now)
	if !state.Season.Initialized {
		state.Season = model.SeasonProgress{Initialized: true, ID: 1, Nodes: []int{}, Resets: 1, Week: week}
	}
	if state.Season.Week != week {
		state.Season.Resets = 1
		state.Season.Week = week
	}
	if state.Boss.Best == nil {
		state.Boss.Best = map[int]int64{}
	}
	if state.Boss.Daily == nil {
		state.Boss.Daily = map[int]int{}
	}
	if state.Boss.Day != day {
		state.Boss.Day = day
		state.Boss.Count = 0
		state.Boss.Daily = map[int]int{}
	}
	if r == nil {
		return
	}
	calendar := now + r.Events.CalendarOffset
	active := 0
	for id, open := range r.Events.BossOpens {
		if calendar >= open.Start && calendar < open.End && id > active {
			active = id
		}
	}
	if _, known := r.Events.BossOpens[state.Boss.Open]; active == 0 && known {
		active = state.Boss.Open
	}
	if state.Boss.Open != active {
		if _, known := r.Events.BossOpens[state.Boss.Open]; known {
			a.settleBossScores(r)
		}
		state.Boss.Open = active
		state.Boss.TotalScore = 0
		state.Boss.Rewards = []int{}
		state.Boss.Best = map[int]int64{}
		state.Boss.Daily = map[int]int{}
		state.Boss.Pending = nil
	}
}

func (a *Account) AddSeasonExperience(r *Rules, amount int64) error {
	if amount < 0 || amount > 1000000000 {
		return rejected("invalid season experience")
	}
	state := &a.Events.Season
	state.Experience += amount
	levels := r.Events.SeasonLevels[state.ID]
	for {
		cost, known := levels[state.Level]
		if _, next := levels[state.Level+1]; !known || !next || cost <= 0 || state.Experience < cost {
			break
		}
		state.Experience -= cost
		state.Level++
	}
	return nil
}

func (r EventRules) scienceNode(id int) int {
	for parent, row := range r.Science {
		if slices.Contains(row.Choices, id) {
			return parent
		}
	}
	return id
}

func (r EventRules) canActivateScience(nodes []int, id int) bool {
	if row, exists := r.Science[id]; !exists || len(row.Choices) > 0 {
		return false
	}
	node := r.scienceNode(id)
	active := make(map[int]bool, len(nodes))
	for _, existing := range nodes {
		active[r.scienceNode(existing)] = true
	}
	if active[node] {
		return false
	}
	root := true
	for parent, row := range r.Science {
		if slices.Contains(row.Next, node) {
			if active[parent] {
				return true
			}
			root = false
		}
	}
	return root
}

func (a *Account) SetScience(r *Rules, id int) error {
	state := &a.Events.Season
	if len(state.Nodes) >= state.Level || !r.Events.canActivateScience(state.Nodes, id) {
		return rejected("science node cannot be activated")
	}
	state.Nodes = append(slices.Clone(state.Nodes), id)
	return nil
}

func (a *Account) ResetScience() error {
	state := &a.Events.Season
	if state.Resets <= 0 || len(state.Nodes) == 0 {
		return rejected("science reset is unavailable")
	}
	state.Resets--
	state.Nodes = []int{}
	return nil
}

func (r EventRules) bossMissions(openID int, now int64) ([]int, bool) {
	open, exists := r.BossOpens[openID]
	calendar := now + r.CalendarOffset
	return r.Bosses[open.Event].Missions, exists && calendar >= open.Start && calendar < min(open.End, open.Settle)
}

func (a *Account) ValidateBossBattle(r *Rules, mission int, battle model.BattleID, now int64) error {
	probe := Account{Events: a.Events}
	probe.InitializeEvents(r, now)
	boss := probe.Events.Boss
	missions, open := r.Events.bossMissions(boss.Open, now)
	index := slices.Index(missions, mission)
	if !open || index < 0 || r.Events.BossMissions[mission].Battle != battle {
		return rejected("boss battle is unavailable")
	}
	if index > 0 && boss.Best[missions[index-1]] <= 0 {
		return rejected("previous boss difficulty is incomplete")
	}
	return nil
}

func (a *Account) FinishBossBattle(r *Rules, mission int, win bool, duration int64) error {
	if !win {
		a.Events.Boss.Pending = nil
		return nil
	}
	rule, exists := r.Events.BossMissions[mission]
	if !exists || duration < 0 || rule.Limit <= 0 {
		return rejected("invalid boss result")
	}
	remaining := max(0, rule.Limit-duration)
	score := rule.Base + int64(math.Floor(float64(remaining)/float64(rule.Limit)*float64(rule.Time)))
	a.Events.Boss.Pending = &model.BossResult{Mission: mission, Score: score, First: a.Events.Boss.Best[mission] == 0}
	a.RecordJourneyTaskEvent(r, 7, 1, mission)
	return nil
}

func (a *Account) BossBest() int64 {
	var best int64
	for _, score := range a.Events.Boss.Best {
		best = max(best, score)
	}
	return best
}

func (a *Account) DeclineBoss() {
	a.Events.Boss.Pending = nil
}

func (a *Account) SettleBoss(r *Rules) (map[model.ItemID]int64, error) {
	state := &a.Events.Boss
	result := state.Pending
	state.Pending = nil
	rewards := map[model.ItemID]int64{}
	if result == nil || state.Count >= r.Constants.Constants.WorldBossDailyTime {
		return rewards, nil
	}
	if result.First {
		var err error
		if rewards, err = rewardItems(r, r.Events.BossMissions[result.Mission].Reward, 1); err != nil {
			return nil, err
		}
		if err = a.GrantItems(r, rewards); err != nil {
			return nil, err
		}
	}
	state.Best[result.Mission] = max(state.Best[result.Mission], result.Score)
	state.Daily[result.Mission] = 1
	state.Count++
	state.TotalScore += result.Score
	return rewards, nil
}

func (a *Account) SweepBoss(r *Rules, mission, count int, now int64) error {
	state := &a.Events.Boss
	missions, open := r.Events.bossMissions(state.Open, now)
	if !open || !slices.Contains(missions, mission) || count < 1 || state.Count+count > r.Constants.Constants.WorldBossDailyTime || state.Best[mission] <= 0 || state.Daily[mission] == 0 {
		return rejected("boss sweep is unavailable")
	}
	state.Count += count
	state.TotalScore += state.Best[mission] * int64(count)
	return nil
}

func (a *Account) ClaimBossScore(r *Rules) (map[model.ItemID]int64, error) {
	var ids []int
	rewards := map[model.ItemID]int64{}
	for score, reward := range r.Events.BossScores {
		if int64(score) > a.Events.Boss.TotalScore || slices.Contains(a.Events.Boss.Rewards, score) {
			continue
		}
		items, err := rewardItems(r, reward, 1)
		if err != nil {
			return nil, err
		}
		addItems(rewards, items)
		ids = append(ids, score)
	}
	if err := a.GrantItems(r, rewards); err != nil {
		return nil, err
	}
	slices.Sort(ids)
	a.Events.Boss.Rewards = append(a.Events.Boss.Rewards, ids...)
	return rewards, nil
}

func (a *Account) ClaimBossRank(r *Rules, position, total, now int64) (map[model.ItemID]int64, error) {
	state := &a.Events.Boss
	open, exists := r.Events.BossOpens[state.Open]
	calendar := now + r.Events.CalendarOffset
	if !exists || position <= 0 || total <= 0 || position > total || calendar <= open.Settle+r.Settings.Boss.SettleDelay || calendar >= open.End || slices.Contains(state.RankClaimed, state.Open) {
		return nil, rejected("boss rank reward is unavailable")
	}
	var reward model.RewardID
	for _, id := range slices.Sorted(maps.Keys(r.Events.BossRanks)) {
		if row := r.Events.BossRanks[id]; row.Matches(position, total, r.Settings.Boss) {
			reward = row.Reward
			break
		}
	}
	if reward == 0 {
		return nil, rejected("boss rank reward is not configured")
	}
	quantities, err := rewardItems(r, reward, 1)
	if err != nil {
		return nil, err
	}
	if err = a.GrantItems(r, quantities); err != nil {
		return nil, err
	}
	state.RankClaimed = append(state.RankClaimed, state.Open)
	a.RecordBenefits("boss_rank", "openId", int64(state.Open), quantities)
	return quantities, nil
}

func (a *Account) EventsSnapshot(profile *model.Avatar) {
	season, boss := a.Events.Season, a.Events.Boss
	profile.Season = model.SeasonComponent{Season: season.ID, Level: season.Level, Experience: season.Experience, Science: season.Nodes, ScienceReset: season.Resets, Missions: map[int]int{}}
	best := make(map[int]int, len(boss.Best))
	for mission, score := range boss.Best {
		best[mission] = int(score)
	}
	settled := 0
	if slices.Contains(boss.RankClaimed, boss.Open) {
		settled = boss.Open
	}
	profile.Boss = model.BossComponent{Passed: best, Daily: boss.Daily, Rewards: boss.Rewards, TotalScore: boss.TotalScore, DailyCount: boss.Count, OpenID: boss.Open, Settled: settled, Records: []model.BossRecord{}}
}
