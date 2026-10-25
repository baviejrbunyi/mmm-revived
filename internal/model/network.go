package model

type NetworkState struct {
	RankCooldowns map[string]int64       `json:"rank_cooldowns"`
	Initialized   bool                   `json:"initialized"`
	Day           int64                  `json:"day"`
	Score         int                    `json:"score"`
	MaxScore      int                    `json:"max_score"`
	Attempts      int                    `json:"attempts"`
	Refreshed     int                    `json:"refreshed"`
	LastRequest   int64                  `json:"last_request"`
	LastRefresh   int64                  `json:"last_refresh"`
	Defense       map[HeroID]int         `json:"defense"`
	DefenseOrder  []HeroID               `json:"defense_order"`
	Interactions  map[int][]int          `json:"interactions"`
	Environment   int                    `json:"environment"`
	Camp          int                    `json:"camp"`
	Activity      ActivityID             `json:"activity"`
	Targets       []AsyncTarget          `json:"targets"`
	Friends       map[UserID]AsyncTarget `json:"friends"`
	Rewarded      []int                  `json:"rewarded"`
	Points        int64                  `json:"points"`
	RewardedTasks []TaskID               `json:"rewarded_tasks"`
	Support       SupportSelection       `json:"support"`
	Pending       *NetworkBattle         `json:"pending,omitempty"`
	RealtimeDay   int64                  `json:"realtime_day,omitempty"`
	SeasonMail    []SeasonMail           `json:"season_mail,omitempty"`
}

type SeasonMail struct {
	Template int64            `json:"template,omitempty"`
	Title    string           `json:"title"`
	Content  string           `json:"content"`
	Sender   string           `json:"sender,omitempty"`
	Items    map[ItemID]int64 `json:"items,omitempty"`
	Reward   RewardID         `json:"reward,omitempty"`
	Mode     RankMode         `json:"mode,omitempty"`
	Rank     int              `json:"rank,omitempty"`
	Score    int64            `json:"score,omitempty"`
	Limit    int64            `json:"limit,omitempty"`
}

type SupportSelection struct {
	Owner UserID `json:"owner"`
	Hero  HeroID `json:"hero"`
	Guild bool   `json:"guild"`
	Used  int    `json:"used"`
	Day   int64  `json:"day"`
}

type NetworkBattle struct {
	Activity      ActivityID   `json:"activity"`
	Target        *AsyncTarget `json:"target,omitempty"`
	Mode          int          `json:"mode"`
	Defender      AsyncDefense `json:"defender"`
	Battle        BattleID     `json:"battle"`
	Opponent      int64        `json:"opponent"`
	Friend        bool         `json:"friend"`
	Started       int64        `json:"started"`
	InitialRecord string       `json:"initial_record"`
	Guild         GuildID      `json:"guild"`
	War           int          `json:"war"`
	Area          int          `json:"area"`
	Level         int          `json:"level"`
	Token         string       `json:"token"`
}

type AsyncTarget struct {
	UID     int64  `json:"uid" wire:"uid"`
	Score   int    `json:"score" wire:"score"`
	Level   int    `json:"level" wire:"level"`
	Defense string `json:"defend" wire:"defend"`
	Shown   string `json:"showDefend" wire:"showDefend"`
	AI      int    `json:"targetAI" wire:"targetAI"`
	AILevel int    `json:"aiLevel" wire:"aiLevel"`
}

type AsyncHeroInfo struct {
	TeamLevel      int                         `wire:"teamLv"`
	SkinCollection int                         `wire:"skinCollectionLv"`
	Heroes         map[HeroID]Hero             `wire:"hero_attr"`
	EquippedGear   map[HeroID]map[int]Artifact `wire:"arti"`
	Equipped       map[HeroID]Highlight        `wire:"highlight"`
	Bonds          map[string]int64            `wire:"bond_attr"`
}

type AsyncDefense struct {
	UID          int64                    `wire:"uid"`
	PVPLevel     int                      `wire:"pvp_lv"`
	WinFix       int                      `wire:"win_fix"`
	LoseFix      int                      `wire:"lose_fix"`
	Positions    map[HeroID]AsyncPosition `wire:"pvp_info"`
	Order        []HeroID                 `wire:"pvp_info_list"`
	Interactions map[int]AsyncInteraction `wire:"interact_object_info"`
	Environment  int                      `wire:"environment_id"`
	Camp         int                      `wire:"camp_env_id"`
	CampDisplay  int                      `wire:"campEnvId"`
	HeroInfo     AsyncHeroInfo            `wire:"hero_info"`
}

type AsyncBattleInfo struct {
	TeamLevel      int              `wire:"teamLv"`
	SkinCollection int              `wire:"skinCollectionLv"`
	Attacker       AsyncDefense     `wire:"attacker"`
	Defender       AsyncDefense     `wire:"defender"`
	Bonds          map[string]int64 `wire:"bondAttr"`
}

type GuildWarState struct {
	Areas     []int                     `json:"areas"`
	Blocked   int                       `json:"blocked"`
	Endurance map[int]int               `json:"endurance"`
	Killed    int64                     `json:"killed"`
	Members   map[UserID]GuildWarMember `json:"members"`
	DayScheme int                       `json:"day_scheme"`
}

type GuildWarMember struct {
	Day      int64         `json:"day" wire:"day"`
	Tickets  int           `json:"ticket" wire:"ticket"`
	Score    int64         `json:"score" wire:"score"`
	Assists  int           `json:"assist" wire:"assist"`
	Finished map[int][]int `json:"finishAreaGroup" wire:"finishAreaGroup"`
	Unlocked map[int]int   `json:"areaUnlockSet" wire:"areaUnlockSet"`
}

type GuildWarGeneral struct {
	Areas     []int          `wire:"areaIndex"`
	Blocked   int            `wire:"blockArea"`
	Endurance map[int]int    `wire:"areaEndurance"`
	Killed    int64          `wire:"finalBossKillTime"`
	Member    GuildWarMember `wire:"memberData"`
	Score     int64          `wire:"score"`
}

type GuildWarHistory struct {
	UID        UserID            `json:"uid" wire:"uid"`
	TeamLevel  int               `json:"teamLevel" wire:"teamLevel"`
	Duration   int64             `json:"useTime" wire:"useTime"`
	Timestamp  int64             `json:"timestamp" wire:"timestamp"`
	Heroes     map[HeroID]string `json:"heroData" wire:"heroData"`
	Best       bool              `json:"best" wire:"best"`
	Assist     HeroID            `json:"assistHero" wire:"assistHero"`
	AssistUID  UserID            `json:"assistUid" wire:"assistUid"`
	AssistData string            `json:"assistHeroData" wire:"assistHeroData"`
}

type GuildWarAssist struct {
	Member UserID `wire:"memberId"`
	Hero   HeroID `wire:"heroId"`
	Data   string `wire:"heroData"`
	Score  int64  `wire:"score"`
}

type AsyncActivity struct {
	Score   int64          `wire:"score"`
	Tasks   map[TaskID]int `wire:"pointTask"`
	Claimed []TaskID       `wire:"rwdedTasks"`
}

func (AsyncActivity) isActivityInfo() {}

type LocalWorldChannel struct {
	ID      string `json:"channelId"`
	SubID   string `json:"subChannelId"`
	Name    string `json:"name"`
	Members int64  `json:"members"`
}

type RankMode string

const (
	RankBoss      RankMode = "boss"
	RankChallenge RankMode = "challenge"
	RankAchieve   RankMode = "achieve"
	RankInfinite  RankMode = "infinite_mayhem"
	RankAsync     RankMode = "async_pvp"
	RankRogue     RankMode = "rogue"
	RankRoguePVE  RankMode = "rogue_pve"
	RankGuildWar  RankMode = "guild_war"
)

type RankEntry struct {
	Score    int64 `json:"score" wire:"score"`
	Achieved int64 `json:"t" wire:"t"`
}

type AsyncPosition struct {
	Born int `wire:"bornId"`
}
type AsyncInteraction struct {
	Slots []int `wire:"slot"`
}
