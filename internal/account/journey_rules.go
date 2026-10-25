package account

import (
	"mmm-revived/internal/model"
)

type JourneyRules struct {
	ChallengeEarlyLevelMargin int                            `json:"challenge_early_level_margin"`
	InviteTasks               map[int]model.TaskID           `json:"invite_tasks"`
	LimitedEventPasses        []int                          `json:"limited_event_passes"`
	InviteShareTask           model.TaskID                   `json:"invite_share_task"`
	MiniGameUnlockMission     int                            `json:"minigame_unlock_mission"`
	Challenges                map[int]JourneyChallengeRule   `json:"challenges"`
	ChallengeTypes            map[int][]int                  `json:"challenge_types"`
	SpiralGroups              map[int]JourneySpiralGroupRule `json:"spiral_groups"`
	SpiralMissions            map[int]JourneySpiralRule      `json:"spiral_missions"`
	InviteRewards             map[int]model.RewardID         `json:"invite_rewards"`
	InviteCountTasks          map[int]model.TaskID           `json:"invite_count_tasks"`
	InviteBindReward          model.RewardID                 `json:"invite_bind_reward"`

	Hyperflows   map[int]JourneyHyperRule       `json:"hyperflows"`
	HyperBuffs   map[int]JourneyHyperBuffRule   `json:"hyper_buffs"`
	HyperBattles map[int]JourneyHyperBattleRule `json:"hyper_battles"`

	CalendarOffset         int64                                    `json:"calendar_offset"`
	MonthlyPurchaseReward  model.RewardID                           `json:"monthly_purchase_reward"`
	MonthlyDailyReward     model.RewardID                           `json:"monthly_daily_reward"`
	MonthlyDuration        int                                      `json:"monthly_duration"`
	MonthlyMaximumDuration int                                      `json:"monthly_maximum_duration"`
	DreamguardTask         model.TaskID                             `json:"dreamguard_task"`
	DreamguardDuration     int64                                    `json:"dreamguard_duration"`
	DreamguardPrice        map[model.ItemID]int64                   `json:"dreamguard_price"`
	DreamguardReward       model.RewardID                           `json:"dreamguard_reward"`
	NoviceTasks            map[model.TaskID]JourneyNoviceRule       `json:"novice_tasks"`
	NoviceNodes            map[int]JourneyNoviceNode                `json:"novice_nodes"`
	Passes                 map[int]JourneyPassRule                  `json:"passes"`
	LimitedPasses          map[int]JourneyPassRule                  `json:"limited_passes"`
	LimitedBosses          map[int]JourneyLimitedBossRule           `json:"limited_bosses"`
	LimitedBossRewards     map[int]JourneyLimitedBossReward         `json:"limited_boss_rewards"`
	Activities             map[model.ActivityID]JourneyActivityRule `json:"activities"`
	Trailers               map[int]JourneyTrailerRule               `json:"trailers"`
	SoloGroups             map[int]JourneySoloRule                  `json:"solo_groups"`
	SoloNodes              map[int]model.BattleID                   `json:"solo_nodes"`
	MiniGames              map[int]JourneyMiniGameRule              `json:"minigames"`
	Missions               map[int]JourneyMissionRule               `json:"missions"`
	MissionNodes           map[int]JourneyMissionNode               `json:"mission_nodes"`
	SevenDay               JourneySevenDayRule                      `json:"seven_day"`
}

type JourneySevenDayRule struct {
	Mark  model.ItemID   `json:"mark"`
	Tasks []model.TaskID `json:"tasks"`
}

type JourneyNoviceRule struct {
	Nodes []int          `json:"nodes"`
	Next  []model.TaskID `json:"next"`
}

type JourneyNoviceNode struct {
	Type       int     `json:"type"`
	Parameters []int64 `json:"parameters"`
}

type JourneyPassReward struct {
	Free model.RewardID   `json:"free"`
	Paid []model.RewardID `json:"paid"`
}

type JourneyPassRule struct {
	Start              int64                     `json:"start"`
	End                int64                     `json:"end"`
	MaximumLevel       int                       `json:"maximum_level"`
	ExperiencePerLevel int64                     `json:"experience_per_level"`
	WeeklyCap          int64                     `json:"weekly_cap"`
	Currency           model.ItemID              `json:"currency"`
	LevelPrice         int64                     `json:"level_price"`
	Tasks              []model.TaskID            `json:"tasks"`
	WeeklyTasks        []model.TaskID            `json:"weekly_tasks"`
	Rewards            map[int]JourneyPassReward `json:"rewards"`
}

type JourneyActivityRule struct {
	PassID       int                    `json:"pass_id"`
	Type         int                    `json:"type"`
	Level        int                    `json:"level"`
	Start        int64                  `json:"start"`
	End          int64                  `json:"end"`
	Duration     int                    `json:"duration"`
	Parameter    int                    `json:"parameter"`
	Rewards      map[int]model.RewardID `json:"rewards"`
	Thresholds   map[int]int            `json:"thresholds"`
	Battles      []model.BattleID       `json:"battles"`
	Environments []int                  `json:"environments"`
	LoginDay     int                    `json:"login_day"`
	LoginLevel   int                    `json:"login_level"`
	Tasks        []model.TaskID         `json:"tasks"`
	Conditions   []JourneyConditionRule `json:"conditions"`
	Rotation     []int                  `json:"rotation"`
	Permanent    []model.TaskID         `json:"permanent"`
	Coin         model.ItemID           `json:"coin"`
	WeeklyCap    int                    `json:"weekly_cap"`
}

type JourneyConditionRule struct {
	Tasks []model.TaskID `json:"tasks"`
	Daily bool           `json:"daily"`
	Once  bool           `json:"once"`
}

type JourneyTrailerRule struct {
	Level  int            `json:"level"`
	Days   int            `json:"days"`
	Reward model.RewardID `json:"reward"`
}

type JourneySoloRule struct {
	Nodes   []int                  `json:"nodes"`
	Rewards map[int]model.RewardID `json:"rewards"`
}

type JourneyMiniGameRule struct {
	Predecessor int            `json:"predecessor"`
	Start       int64          `json:"start"`
	Level       int            `json:"level"`
	Reward      model.RewardID `json:"reward"`
}

type JourneyLimitedBossRule struct {
	Battles     []model.BattleID `json:"battles"`
	Cost        []int64          `json:"cost"`
	Open        int64            `json:"open"`
	End         int64            `json:"end"`
	Rewards     []model.RewardID `json:"rewards"`
	LevelReward int              `json:"level_reward"`
}

type JourneyLimitedBossReward struct {
	Tag    int            `json:"tag"`
	Levels []int          `json:"levels"`
	Reward model.RewardID `json:"reward"`
}

type JourneyMissionRule struct {
	Level        int            `json:"level"`
	Predecessors []int          `json:"predecessors"`
	FirstNode    int            `json:"first_node"`
	Reward       model.RewardID `json:"reward"`
}

type JourneyMissionNode struct {
	Next      int   `json:"next"`
	Type      int   `json:"type"`
	Parameter int64 `json:"parameter"`
}

type JourneyHyperRule struct {
	BuffNodes   []int          `json:"buff_nodes"`
	BattleNodes []int          `json:"battle_nodes"`
	Tasks       []model.TaskID `json:"tasks"`
}

type JourneyHyperBuffRule struct {
	Type      int   `json:"type"`
	Choices   []int `json:"choices"`
	Threshold int   `json:"threshold"`
}

type JourneyHyperBattleRule struct {
	Rewards [][]model.RewardID `json:"rewards"`
	Battles []model.BattleID   `json:"battles"`
	Next    int                `json:"next"`
}

type JourneyChallengeRule struct {
	Difficulty  int              `json:"difficulty"`
	Group       int              `json:"group"`
	Battle      model.BattleID   `json:"battle"`
	Level       int              `json:"level"`
	Predecessor int              `json:"predecessor"`
	Energy      int64            `json:"energy"`
	Active      int              `json:"active"`
	Rewards     []model.RewardID `json:"rewards"`
	StarRewards []model.RewardID `json:"star_rewards"`
}

type JourneySpiralGroupRule struct {
	Missions []int                  `json:"missions"`
	Rewards  map[int]model.RewardID `json:"rewards"`
}

type JourneySpiralRule struct {
	Battle      model.BattleID   `json:"battle"`
	Predecessor int              `json:"predecessor"`
	Rewards     []model.RewardID `json:"rewards"`
}
