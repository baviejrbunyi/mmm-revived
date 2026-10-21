package model

type JourneyState struct {
	DailySpent     map[ItemID]int64 `json:"daily_spent"`
	SpentDay       int64            `json:"spent_day"`
	WeeklySpent    map[ItemID]int64 `json:"weekly_spent"`
	SpentWeek      int64            `json:"spent_week"`
	GuildJoined    int64            `json:"guild_joined"`
	GuildLoginDays int              `json:"guild_login_days"`
	GuildLoginDay  int64            `json:"guild_login_day"`

	SpentItems       map[ItemID]int64 `json:"spent_items"`
	TaskBaselines    map[TaskID]int64 `json:"task_baselines"`
	TaskDay          int64            `json:"task_day"`
	DailyClaims      int64            `json:"daily_claims"`
	GuildMember      bool             `json:"guild_member"`
	AutomaticPending bool             `json:"-"`
	ProcessingTasks  bool             `json:"-"`

	Challenges    map[int]JourneyChallengeRecord `json:"challenges"`
	ChallengeRun  *JourneyChallengeRun           `json:"challenge_run,omitempty"`
	Spiral        map[int]int                    `json:"spiral"`
	SpiralRewards map[int][]int                  `json:"spiral_rewards"`
	InviteRewards map[UserID]map[int]bool        `json:"invite_rewards"`
	SharedInvite  bool                           `json:"shared_invite"`

	Initialized bool                           `json:"initialized"`
	Novice      JourneyNovice                  `json:"novice"`
	Pass        JourneyPass                    `json:"pass"`
	Monthly     JourneyMonthly                 `json:"monthly"`
	Activities  map[ActivityID]JourneyActivity `json:"activities"`
	LoginDays   int                            `json:"login_days"`
	LoginDay    int64                          `json:"login_day"`
}

type JourneyNovice struct {
	Task       TaskID            `json:"task"`
	Node       int               `json:"node"`
	Nodes      map[int]int       `json:"nodes"`
	Completed  map[TaskID]bool   `json:"completed"`
	Rewards    map[int]bool      `json:"rewards"`
	BattleWins map[BattleID]bool `json:"battle_wins"`
}

type JourneyPass struct {
	Tasks       []TaskID    `json:"tasks" wire:"currentTaskIds"`
	WeeklyTasks []TaskID    `json:"weekly_tasks" wire:"weekTaskIds"`
	ID          int         `json:"id" wire:"passId"`
	Level       int         `json:"level" wire:"passLevel"`
	Experience  int64       `json:"experience" wire:"passExp"`
	WeeklyXP    int64       `json:"weekly_xp" wire:"weekPassExp"`
	Week        int64       `json:"week" wire:"-"`
	Active      int         `json:"active" wire:"activeType"`
	Free        map[int]int `json:"free" wire:"freePassReward"`
	Paid        map[int]int `json:"paid" wire:"feePassReward"`
}

type JourneyMonthly struct {
	Expires         int64 `json:"expires"`
	LastDaily       int64 `json:"last_daily"`
	PendingPurchase int   `json:"pending_purchase"`
}

type JourneyMission struct {
	Nodes    map[int]MissionNode `json:"nodes" wire:"nodes"`
	Passed   map[int]int         `json:"passed" wire:"passedNodes"`
	Node     int                 `json:"node" wire:"currNodeId"`
	Finished int                 `json:"finished" wire:"finished"`
}

type JourneyActivity struct {
	Camp         *RetailCampState   `json:"camp,omitempty" wire:"-"`
	RetailPVP    *RetailPVPState    `json:"retail_pvp,omitempty" wire:"-"`
	DailyDay     int64              `json:"daily_day" wire:"-"`
	Interacted   int                `json:"interacted" wire:"interact"`
	Open         bool               `json:"open" wire:"-"`
	HyperPoint   int                `json:"hyper_point" wire:"hyperFlowPoint"`
	HyperMaximum int                `json:"hyper_maximum" wire:"hyperFlowPointMax"`
	BuffReset    int                `json:"buff_reset" wire:"buffResetTimes"`
	BuffNodes    []JourneyBuffNode  `json:"buff_nodes" wire:"buffNodes"`
	BattleNodes  []JourneyHyperNode `json:"battle_nodes" wire:"battleNodes"`

	Initialized    int64                  `json:"initialized" wire:"initTime"`
	Signed         int                    `json:"signed" wire:"signed"`
	LastSign       int64                  `json:"last_sign" wire:"-"`
	SignRewards    []int                  `json:"sign_rewards" wire:"rwded"`
	TrailerRewards map[int]int            `json:"trailer_rewards" wire:"rwdState"`
	TaskStart      int64                  `json:"task_start" wire:"taskStartTime"`
	TaskForever    int                    `json:"task_forever" wire:"taskForever"`
	TaskClaimed    bool                   `json:"task_claimed" wire:"-"`
	Star           int                    `json:"star" wire:"star"`
	BestScore      int                    `json:"best_score" wire:"bestScore"`
	Rewards        []int                  `json:"rewards" wire:"rewarded"`
	StarRewards    []int                  `json:"star_rewards" wire:"rwdStarList"`
	NodeStars      map[int]int            `json:"node_stars" wire:"nodeStar"`
	Pass           JourneyPass            `json:"pass" wire:"battlePass"`
	Missions       map[int]JourneyMission `json:"missions" wire:"storyLevelStat"`
	MiniGames      map[int]int64          `json:"mini_games" wire:"gridMovementLevelStat"`
}

func (JourneyActivity) isActivityInfo() {}

type JourneyBuffNode struct {
	ID     int  `json:"id" wire:"buffNodeId"`
	Active bool `json:"active" wire:"active"`
	Choice int  `json:"choice" wire:"chooseBuffNodeId"`
}

type JourneyHyperBattle struct {
	ID      BattleID `json:"id" wire:"battleId"`
	Stars   int      `json:"stars" wire:"star"`
	Heroes  []HeroID `json:"heroes" wire:"heros"`
	Died    []HeroID `json:"died" wire:"heroDie"`
	UseTime int64    `json:"use_time" wire:"useTime"`
}

type JourneyHyperNode struct {
	Rewarded         int                `json:"rewarded" wire:"-"`
	ID               int                `json:"id" wire:"nodeId"`
	Open             bool               `json:"open" wire:"opened"`
	Stars            int                `json:"stars" wire:"star"`
	First            JourneyHyperBattle `json:"first" wire:"battle1"`
	Second           JourneyHyperBattle `json:"second" wire:"battle2"`
	BannedArtifacts  []InstanceID       `json:"banned_artifacts" wire:"bannedArtifacts"`
	BannedHighlights []InstanceID       `json:"banned_highlights" wire:"bannedHighlights"`
}

type JourneyChallengeRecord struct {
	Stars       int   `json:"stars" wire:"star"`
	Bitmap      int   `json:"bitmap" wire:"-"`
	StarRewards int   `json:"star_rewards" wire:"-"`
	Time        int64 `json:"time" wire:"duration"`
	Kills       int   `json:"kills" wire:"killedCount"`
}

type JourneyChallengeRun struct {
	ID     int      `json:"id"`
	Battle BattleID `json:"battle"`
	Count  int      `json:"count"`
}

type JourneySpiralGroup struct {
	Rewards []int `wire:"rwdStarList"`
}

type InviteProgress struct {
	UID       UserID
	Level     int
	Story     bool
	Purchases map[int]bool
	Bound     int64
}
