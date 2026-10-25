package account

import (
	"maps"
	"slices"

	"mmm-revived/internal/model"
	"mmm-revived/internal/social"
)

type ConstantRules struct {
	Constants       ClientConstants                 `json:"constants"`
	Items           ClientItems                     `json:"item_ids"`
	HonorTags       []string                        `json:"honor_tags"`
	SystemUnlocks   map[string]SystemUnlockRule     `json:"system_unlocks"`
	HeroStarRewards map[model.HeroID]model.RewardID `json:"hero_star_rewards"`
	GuildLevels     map[int]GuildLevelRule          `json:"guild_levels"`
	GuildWars       map[int]GuildWarPeriodRule      `json:"guild_wars"`
}

type ClientConstants struct {
	PVPInitialPoint              int64                  `json:"PVP_init_point"`
	FriendLimit                  int                    `json:"friend_limit"`
	FriendRequestLimit           int                    `json:"friend_request_limit"`
	BlacklistLimit               int                    `json:"blacklist_limit"`
	FriendGiftReceive            int                    `json:"friend_gift_max_recv"`
	FriendGiftSend               int                    `json:"friend_gift_max_send"`
	FriendGift                   []int64                `json:"friend_gift_item_id"`
	FriendSupportReward          []int64                `json:"friend_support_reward"`
	FriendSupportLimit           []int64                `json:"friend_support_limit"`
	FriendHelpLimit              int64                  `json:"friend_help_limit"`
	FriendHelpUses               int64                  `json:"friend_help_uses_limit"`
	ChatWordCount                int                    `json:"chat_word_count"`
	TrialHeroMaxLevel            int                    `json:"trial_hero_max_lv"`
	TrialHeroMaxTalent           int                    `json:"trial_hero_max_talent"`
	HeroStarUnlockLevel          int                    `json:"hero_star_unlock_lv"`
	HeroStarAwakeItems           map[int]model.ItemID   `json:"hero_star_awake_item"`
	HighlightAwakeItems          map[int]model.ItemID   `json:"highlight_awake_item"`
	AwakeResetCost               map[model.ItemID]int64 `json:"awake_hero_reset_item"`
	ArtifactRefreshCost          map[model.ItemID]int64 `json:"artifact_refresh_cost"`
	ArtifactRefreshLock          model.ItemID           `json:"artifact_refresh_lock"`
	HeroProficiency              int64                  `json:"hero_prof_num"`
	HeroProficiencyModes         []int                  `json:"hero_prof_valid_battle_mode"`
	HeroProficiencyDispatch      int64                  `json:"hero_prof_num_dispatch"`
	DispatchOffers               int                    `json:"dispatch_refresh_count"`
	DispatchRefreshLimit         int                    `json:"dispatch_daily_max_refresh"`
	DispatchDuration             int64                  `json:"dispatch_duration"`
	DailyShareTask               model.TaskID           `json:"daily_share_task_id"`
	SweepChallengeItem           model.ItemID           `json:"sweep_challenge_item"`
	ChallengeStarMultipliers     []int                  `json:"challenge_star_can_multi"`
	HyperflowResetBase           int64                  `json:"hyperflow_reset_coin_base"`
	HyperflowResetIncrement      int64                  `json:"hyperflow_reset_coin_increment"`
	HyperflowResetMaximum        int64                  `json:"hyperflow_reset_coin_max"`
	InviteCloseBindLevel         int                    `json:"invite_module_close_bind_lv"`
	InviteParameters             []int                  `json:"invite_system_params"`
	RealtimeDeployRoundTime      int                    `json:"realtime_pvp_deploy_round_time"`
	RealtimeFirstDeployRoundTime int                    `json:"realtime_pvp_deploy_round_time1"`
	NameMinimum                  int                    `json:"nick_name_length_limit"`
	NameMaximum                  int                    `json:"nick_name_length_limit_max"`
	SignatureMaximum             int                    `json:"teaminfo_signature_max_lenth"`
	GuildCreation                []int64                `json:"alliance_create_item"`
	GuildPlayerApplications      int                    `json:"alliance_player_apply_max"`
	GuildApplications            int                    `json:"alliance_apply_max"`
	GuildAdmins                  int                    `json:"alliance_max_admin"`
	GuildInvites                 int                    `json:"alliance_max_invite"`
	GuildLeaveCooldown           int64                  `json:"alliance_leave_cd"`
	GuildInviteTime              int64                  `json:"alliance_invite_time"`
	GuildApplyTime               int64                  `json:"alliance_apply_time"`
	GuildExperienceRate          int64                  `json:"alliance_exp_trans"`
	GuildRenameCooldown          int64                  `json:"alliance_change_name_cd"`
	GuildNameLength              int                    `json:"alliance_name_length"`
	GuildNameMinimum             int                    `json:"team_name_length_limit"`
	GuildDescriptionLength       int                    `json:"alliance_des_length"`
	GuildNoticeLength            int                    `json:"alliance_notice_length"`
	MailHead                     model.ItemID           `json:"mail_Head"`
	MailHeadFrame                model.ItemID           `json:"mail_HeadFrame"`
	RogueRecordLength            int                    `json:"rogue_record_len"`
	WorldBossDailyTime           int                    `json:"world_boss_daily_time"`
	MailLimit                    int                    `json:"avt_mail_limit"`
	RenameItem                   model.ItemID           `json:"change_name_item"`
	RenameCurrency               []int64                `json:"change_name_item_currency"`
}

type ClientItems struct {
	Gold                model.ItemID `json:"GOLD"`
	Diamond             model.ItemID `json:"DIAMOND"`
	Yuanbao             model.ItemID `json:"YUANBAO"`
	Energy              model.ItemID `json:"ENERGY"`
	HeroExperience      model.ItemID `json:"HERO_EXP"`
	ArtifactExperience  model.ItemID `json:"ARTIFACT_EXP"`
	HighlightExperience model.ItemID `json:"HIGHLIGHT_EXP"`
	PVPScore            model.ItemID `json:"PVP_SCORE"`
	AsyncPVPScore       model.ItemID `json:"ASYNC_PVP_SCORE"`
	AwakeResetCurrency  model.ItemID `json:"AWAKE_RESET_CURRENCY"`
	RogueCurrency       model.ItemID `json:"ROGUE_CURRENCY"`
	RoguePVECurrency    model.ItemID `json:"ROGUE_PVE_CURRENCY"`
	GuildWarTicket      model.ItemID `json:"ALLIANCE_WAR_TICKET"`
	GuildWarScore       model.ItemID `json:"ALLIANCE_WAR_SCORE"`
	HalloweenCurrency   model.ItemID `json:"HALLOWEN_CURRENCY"`
	HalloweenPoint      model.ItemID `json:"HALLOWEN_POINT"`
	DrawKey1            model.ItemID `json:"DRAW_KEY1"`
	DrawKey2            model.ItemID `json:"DRAW_KEY2"`
}

type SystemUnlockRule struct {
	Level      int             `json:"level"`
	LoginDay   int             `json:"login_day"`
	LoginLevel int             `json:"login_level"`
	Mission    model.MissionID `json:"mission"`
	Task       model.TaskID    `json:"task"`
}

type GuildLevelRule struct {
	Experience  int64          `json:"experience"`
	Members     int            `json:"members"`
	Reward      model.RewardID `json:"reward"`
	Minimum     int64          `json:"minimum"`
	PerReward   int64          `json:"per_reward"`
	RewardLimit int64          `json:"reward_limit"`
}

type GuildWarPeriodRule struct {
	Prepare int64   `json:"prepare"`
	Battle  int64   `json:"battle"`
	End     int64   `json:"end"`
	Phases  []int64 `json:"phases"`
}

func (r *Rules) SocialConfig() social.Config {
	c := r.Constants.Constants
	config := social.Config{
		Limits:          social.Limits{Friends: c.FriendLimit, Applications: c.FriendRequestLimit, SentGifts: c.FriendGiftSend, ReceivedGifts: c.FriendGiftReceive, Blocked: c.BlacklistLimit, Message: c.ChatWordCount},
		GuildLimits:     social.GuildLimits{Admins: c.GuildAdmins, Applications: c.GuildApplications, PlayerApplications: c.GuildPlayerApplications, Invites: c.GuildInvites, LeaveCooldown: c.GuildLeaveCooldown, RequestLifetime: max(c.GuildApplyTime, c.GuildInviteTime), RenameCooldown: c.GuildRenameCooldown, MaximumLevel: r.MaximumTeamLevel(), ExperienceRate: c.GuildExperienceRate, NameMinimum: c.GuildNameMinimum, NameMaximum: c.GuildNameLength, Signature: c.GuildDescriptionLength, Announcement: c.GuildNoticeLength},
		SettlementTexts: r.MailTexts(r.Settings.Mail.GuildSettlement),
		SettlementMail:  int64(r.Settings.Mail.GuildSettlement),
	}
	if len(c.FriendGift) > 1 {
		config.Limits.GiftItem, config.Limits.GiftEnergy = model.ItemID(c.FriendGift[0]), c.FriendGift[1]
	}
	if len(c.GuildCreation) > 1 {
		config.GuildLimits.CreationItem, config.GuildLimits.CreationCost = model.ItemID(c.GuildCreation[0]), c.GuildCreation[1]
	}
	for _, level := range slices.Sorted(maps.Keys(r.Constants.GuildLevels)) {
		rule := r.Constants.GuildLevels[level]
		rewards := map[model.ItemID]int64{}
		for _, row := range r.Rewards[rule.Reward] {
			rewards[model.ItemID(row[0])] += row[1]
		}
		config.GuildLevels = append(config.GuildLevels, social.GuildLevel{Experience: rule.Experience, Members: rule.Members, Rewards: rewards, Minimum: rule.Minimum, PerReward: rule.PerReward, RewardLimit: rule.RewardLimit})
	}
	if len(config.GuildLevels) > 0 {
		config.GuildLimits.Members = config.GuildLevels[0].Members
	}
	for _, id := range slices.Sorted(maps.Keys(r.Constants.GuildWars)) {
		war := r.Constants.GuildWars[id]
		config.Wars = append(config.Wars, social.WarPeriod{ID: id, Prepare: war.Prepare, Battle: war.Battle, End: war.End, Changes: war.Phases})
	}
	return config
}

func (r *Rules) ProficiencyBattle(battle model.BattleID) bool {
	if r == nil {
		return false
	}
	mode, known := r.BattleModes[battle]
	return known && slices.Contains(r.Constants.Constants.HeroProficiencyModes, mode)
}

func (r *Rules) SystemLevel(system string) int {
	if r == nil {
		return 0
	}
	return r.Constants.SystemUnlocks[system].Level
}

func (r *Rules) StoryMissions() []model.MissionID {
	var missions []model.MissionID
	if r == nil {
		return missions
	}
	for _, id := range slices.Sorted(maps.Keys(r.Chapters)) {
		if r.GM.Chapters[id] == "story" {
			missions = append(missions, r.Chapters[id].Missions...)
		}
	}
	return missions
}

func (r *Rules) MaximumTeamLevel() int {
	if r == nil || len(r.TeamLevels) == 0 {
		return 0
	}
	return slices.Max(slices.Collect(maps.Keys(r.TeamLevels)))
}

func (r *Rules) ActivityOfType(kind int) (model.ActivityID, JourneyActivityRule, bool) {
	if r == nil {
		return 0, JourneyActivityRule{}, false
	}
	for _, id := range slices.Sorted(maps.Keys(r.Journey.Activities)) {
		if rule := r.Journey.Activities[id]; rule.Type == kind {
			return id, rule, true
		}
	}
	return 0, JourneyActivityRule{}, false
}
