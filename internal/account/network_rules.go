package account

import (
	"slices"

	"mmm-revived/internal/model"
)

type NetworkRules struct {
	Bots       map[int]AsyncAIRule            `json:"bots"`
	Ranks      map[int]AsyncRankRule          `json:"ranks"`
	Activities map[model.ActivityID]AsyncRule `json:"activities"`
	Wars       map[int]GuildWarRule           `json:"wars"`
	Areas      map[int]GuildWarAreaRule       `json:"areas"`
	WarLevels  map[int]GuildWarLevelRule      `json:"war_levels"`
	Seasons    map[int]AsyncRankSeason        `json:"rank_seasons"`
	Rankings   map[model.ActivityID]Ranking   `json:"season_rankings"`
	Initial    int                            `json:"initial_score"`
}

type AsyncRankSeason struct {
	Reset        int `json:"reset"`
	Environments int `json:"environments"`
}

type Ranking struct {
	Reward model.RewardID `json:"reward"`
	Limit  int64          `json:"limit"`
}

type AsyncRankRule struct {
	Minimum   int            `json:"minimum"`
	Maximum   int            `json:"maximum"`
	Reward    model.RewardID `json:"reward"`
	Win       int            `json:"win"`
	Lose      int            `json:"lose"`
	Kill      int            `json:"kill"`
	Killed    int            `json:"killed"`
	AllKill   int            `json:"all_kill"`
	AllKilled int            `json:"all_killed"`
	AILevel   int            `json:"ai_level"`
}

type AsyncRule struct {
	Battles      []model.BattleID `json:"battles"`
	Attempts     int              `json:"attempts"`
	Refreshes    int              `json:"refreshes"`
	Tasks        []model.TaskID   `json:"tasks"`
	Environments []int            `json:"environments"`
	Camps        []int            `json:"camps"`
}

type GuildWarRule struct {
	Areas       []int                  `json:"areas"`
	BossGroup   int                    `json:"boss_group"`
	BossHP      int                    `json:"boss_hp"`
	BossReward  model.RewardID         `json:"boss_reward"`
	AreaRewards map[int]model.RewardID `json:"area_rewards"`
}

type GuildWarAreaRule struct {
	HP   int     `json:"hp"`
	Sets [][]int `json:"sets"`
}

type GuildWarLevelRule struct {
	Group        int            `json:"group"`
	Battle       model.BattleID `json:"battle"`
	MinimumLevel int            `json:"minimum_level"`
	MaximumLevel int            `json:"maximum_level"`
	Reduction    int            `json:"reduction"`
	Medals       int64          `json:"medals"`
	Assist       int            `json:"assist"`
	Reward       model.RewardID `json:"reward"`
}

func (r NetworkRules) Rank(score int) (int, AsyncRankRule) {
	best := 0
	for id, rank := range r.Ranks {
		if rank.Minimum <= score && id > best {
			best = id
		}
	}
	return best, r.Ranks[best]
}

func (r NetworkRules) DefaultActivity() model.ActivityID {
	var best model.ActivityID
	for id := range r.Activities {
		best = max(best, id)
	}
	return best
}

func (r NetworkRules) DefaultWar() int {
	best := 0
	for id := range r.Wars {
		best = max(best, id)
	}
	return best
}

func (r NetworkRules) WarSet(war, area, group int) (int, bool) {
	if area == 0 {
		return 0, r.Wars[war].BossGroup == group
	}
	if !slices.Contains(r.Wars[war].Areas, area) {
		return 0, false
	}
	for index, set := range r.Areas[area].Sets {
		if slices.Contains(set, group) {
			return index + 1, true
		}
	}
	return 0, false
}

type AsyncAIRule struct {
	Heroes      map[model.HeroID]int `json:"heroes"`
	Environment int                  `json:"environment"`
	Camp        int                  `json:"camp"`
}
