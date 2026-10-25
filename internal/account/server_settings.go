package account

import "mmm-revived/internal/model"

type ServerSettings struct {
	Mail               MailTemplates                       `json:"mail_templates"`
	Items              SpecialItems                        `json:"special_items"`
	PassLevelBonusItem model.ItemID                        `json:"pass_level_bonus_item"`
	BattleAchievements map[string]BattleAchievementSetting `json:"battle_achievements"`
	DoubleMaterial     int64                               `json:"double_material_multiplier"`
	Boss               BossSettings                        `json:"boss"`
	SupportLevelGap    int                                 `json:"support_level_gap"`
	Realtime           RealtimeSettings                    `json:"realtime"`
	ProtectionForget   int64                               `json:"protection_forget_delay"`
	AllEvents          AllEventSettings                    `json:"all_events"`
}

type MailTemplates struct {
	BossRank              int `json:"boss_rank"`
	BossScore             int `json:"boss_score"`
	GuildSettlement       int `json:"guild_settlement"`
	BifrostReward         int `json:"bifrost_reward"`
	BifrostRanking        int `json:"bifrost_ranking"`
	RetailRanking         int `json:"retail_ranking"`
	LimitedPassSettlement int `json:"limited_pass_settlement"`
	PassSettlement        int `json:"pass_settlement"`
}

type SpecialItems struct {
	TeamExperience        model.ItemID `json:"team_experience"`
	SeasonExperience      model.ItemID `json:"season_experience"`
	PassActivation        model.ItemID `json:"pass_activation"`
	PremiumPassActivation model.ItemID `json:"premium_pass_activation"`
	PassExperience        model.ItemID `json:"pass_experience"`
	MonthlyPass           model.ItemID `json:"monthly_pass"`
	LimitedPassExperience model.ItemID `json:"limited_pass_experience"`
	HyperflowPoints       model.ItemID `json:"hyperflow_points"`
}

type BattleAchievementSetting struct {
	TaskType int            `json:"task_type"`
	Battle   model.BattleID `json:"battle"`
}

type BossSettings struct {
	SettleDelay      int64 `json:"settle_delay"`
	RankDisplay      int64 `json:"rank_display"`
	RankBoardMinimum int64 `json:"rank_board_minimum"`
}

type RealtimeSettings struct {
	Rounds         int `json:"rounds"`
	HeroesPerRound int `json:"heroes_per_round"`
}
