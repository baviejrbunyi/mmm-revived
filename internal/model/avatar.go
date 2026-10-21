package model

type BattleComponent struct {
	PassMessages map[UserID]string `wire:"passMsg"`
	SupportUses  int               `wire:"useSupportHeroTime"`
}

type MatchComponent struct {
	Type      int   `wire:"currMatchType"`
	StartedAt int64 `wire:"startMatchTime"`
	Estimate  int   `wire:"estimatedTime"`
}

type Avatar struct {
	GM          GMComponent           `wire:"gmComp"`
	UID         UserID                `wire:"uid"`
	AccountID   string                `wire:"strAid"`
	Username    string                `wire:"user_name"`
	Host        int                   `wire:"logic_hostnum"`
	CreatedAt   int64                 `wire:"create_time"`
	LastLogin   int64                 `wire:"last_login_time"`
	Notice      map[int]int           `wire:"notice"`
	Basic       BasicComponent        `wire:"basicComp"`
	Heroes      HeroComponent         `wire:"heroComp"`
	Items       ItemComponent         `wire:"itemComp"`
	Lobby       LobbyComponent        `wire:"lobbyComp"`
	Novice      NoviceComponent       `wire:"noviceComp"`
	Energy      EnergyComponent       `wire:"energyComp"`
	Season      SeasonComponent       `wire:"seasonComp"`
	Tasks       TaskComponent         `wire:"taskComp"`
	Activities  ActivityComponent     `wire:"activityComp"`
	Lineups     LineupComponent       `wire:"lineupComp"`
	Battle      BattleComponent       `wire:"battleNewComp"`
	Match       MatchComponent        `wire:"matchComp"`
	Dreamland   DreamlandComponent    `wire:"dreamlandComp"`
	Shop        ShopComponent         `wire:"shopComp"`
	Challenges  ChallengeComponent    `wire:"challengeComp"`
	Boss        BossComponent         `wire:"bossComp"`
	Appearance  AppearanceComponent   `wire:"makeupComp"`
	Manual      ManualComponent       `wire:"manualComp"`
	Puzzle      PuzzleComponent       `wire:"puzzleComp"`
	Achievement AchievementComponent  `wire:"achieveComp"`
	Spiral      SpiralComponent       `wire:"spiralComp"`
	Dispatch    DispatchComponent     `wire:"dispatchComp"`
	Guild       GuildComponent        `wire:"guildComp"`
	MonthlyPass MonthlyPassComponent  `wire:"monthlyPassComp"`
	BattlePass  BattlePassComponent   `wire:"battlepassComp"`
	Draw        DrawComponent         `wire:"drawComp"`
	Friends     FriendComponent       `wire:"friendComp"`
	Mail        MailComponent         `wire:"mailComp"`
	History     HistoryComponent      `wire:"historyComp"`
	Purchases   PurchaseComponent     `wire:"shipComp"`
	Invites     InviteComponent       `wire:"inviteComp"`
	PopupPacks  PopupPackComponent    `wire:"popupPackComp"`
	HeroPacks   HeroPurchaseComponent `wire:"heroPurchaseComp"`
	Benefits    BenefitsComponent     `wire:"benefitsSummaryComp"`
	Questions   QuestionComponent     `wire:"jingLingComp"`
	Bindings    Empty                 `wire:"bindAccComp"`
	PVP         PVPComponent          `wire:"pvpComp"`
	Rogue       RogueComponent        `wire:"rogueComp"`
	RoguePVE    RogueComponent        `wire:"roguePveComp"`
}

type BattleExtra struct {
	TeamLevel      int   `wire:"teamLv"`
	SeasonScience  []int `wire:"seasonScienceNodeIds"`
	HyperFlowBuffs []int `wire:"hyper_flow_buffs"`
}
