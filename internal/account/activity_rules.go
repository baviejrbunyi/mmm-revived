package account

import "mmm-revived/internal/model"

type RetailActivityRules struct {
	Rewards             map[model.RewardID][][2]int64            `json:"rewards"`
	Items               map[model.ItemID]ItemRule                `json:"items"`
	Activities          map[model.ActivityID]JourneyActivityRule `json:"activities"`
	CampGroups          map[int][]int                            `json:"camp_groups"`
	CampNodes           map[int]RetailCampNodeRule               `json:"camp_nodes"`
	CampScores          map[int]RetailCampScoreRule              `json:"camp_scores"`
	CampScorePolicy     string                                   `json:"camp_score_policy"`
	DailyRefresh        int64                                    `json:"daily_refresh"`
	InviteMaximum       int                                      `json:"invite_maximum"`
	InviteRewardMaximum int                                      `json:"invite_reward_maximum"`
}

type RetailCampNodeRule struct {
	Battles      []model.BattleID       `json:"battles"`
	Difficulties []int                  `json:"difficulties"`
	Camp         int                    `json:"camp"`
	Cost         map[model.ItemID]int64 `json:"cost"`
	Days         []int                  `json:"days"`
	Rewards      []model.RewardID       `json:"rewards"`
}

type RetailCampScoreRule struct {
	Item      model.ItemID `json:"item"`
	Base      int64        `json:"base"`
	PerSecond int64        `json:"per_second"`
	Limit     int64        `json:"limit"`
}
