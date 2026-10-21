package model

type SeasonComponent struct {
	Season       int         `wire:"season"`
	Level        int         `wire:"lv"`
	Experience   int64       `wire:"exp"`
	Missions     map[int]int `wire:"mission"`
	Science      []int       `wire:"scienceNodes"`
	ScienceReset int         `wire:"scienceReset"`
}

type TaskComponent struct {
	Tasks map[TaskID]Task `wire:"tasks"`
}

type Task struct {
	Status int `json:"status" wire:"st"`
	Value  int `json:"value" wire:"val"`
}

type MissionNode struct {
	ID       NodeID `json:"id" wire:"nodeId"`
	Rewarded int    `json:"rewarded" wire:"rwded"`
}

type Mission struct {
	BoxMessageUID  UserID                `json:"box_message_uid" wire:"boxMsgUid"`
	BoxMessage     string                `json:"box_message" wire:"boxMsg"`
	Visited        map[MementoChipID]int `json:"visited" wire:"visited"`
	LokiRecord     map[MementoChipID]int `json:"loki_record" wire:"lokiRec"`
	Step           int                   `json:"step" wire:"step"`
	Oxygen         int                   `json:"oxygen" wire:"oxygen"`
	ShieldDecrease int                   `json:"shield_decrease" wire:"shieldDec"`
	BattleResult   int                   `json:"battle_result" wire:"bResult"`
	BoxReward      int                   `json:"box_reward" wire:"boxRwd"`
	ID             MissionID             `json:"id" wire:"missionId"`
	Node           MissionNode           `json:"node" wire:"node"`
	Fail           int                   `json:"fail" wire:"fail"`
	Heroes         map[HeroID]int        `json:"heroes" wire:"herovisited"`
}

type MissionProgress struct {
	Stars       int `json:"stars" wire:"star"`
	Rewarded    int `json:"rewarded" wire:"passRwded"`
	BoxRewarded int `json:"box_rewarded" wire:"boxRwded"`
}

type ActivityInfo interface{ isActivityInfo() }

func (PVPActivity) isActivityInfo() {}

type ActivityComponent struct {
	Info   map[ActivityID]ActivityInfo `wire:"activityInfo"`
	Status map[ActivityID]int          `wire:"activitySt"`
}

type DreamlandComponent struct {
	Chapters map[ChapterID]ChapterProgress `wire:"chapterInfo"`
	Missions map[MissionID]Mission         `wire:"missionInfo"`
	Passed   map[MissionID]MissionProgress `wire:"passedMission"`
	Memento  MissionID                     `wire:"curMemento"`
	Story    MissionID                     `wire:"curStory"`
}

type ChapterProgress struct {
	Rewards []int `json:"rewards" wire:"sumStarRwded"`
}

type ChallengeComponent struct {
	Challenges map[int]JourneyChallengeRecord `wire:"challengeData"`
	Double     map[int]int                    `wire:"doubleRwdChanceDict"`
	DoubleType int                            `wire:"doubleRwdAllType"`
	Chances    int                            `wire:"doubleRwdChance"`
}

type BossComponent struct {
	Passed     map[int]int `wire:"bossPassed"`
	Daily      map[int]int `wire:"dailyPassed"`
	Rewards    []int       `wire:"scoreRwded"`
	TotalScore int64       `wire:"totalScore"`
	DailyCount int         `wire:"dailyNum"`
	OpenID     int         `wire:"openId"`
	Settled    int         `wire:"rankSettled"`
	Records    []Empty     `wire:"bossRecord"`
}

type ManualComponent struct {
	Monsters   map[int]int `wire:"monsterManual"`
	Artifacts  map[int]int `wire:"artifactSuitManual"`
	Highlights map[int]int `wire:"highlightManual"`
	Items      map[int]int `wire:"itemManual"`
	EasterEggs map[int]int `wire:"easterEgg2num"`
}

type PuzzleComponent struct {
	Rewards map[int]int `wire:"rwdDict"`
}

type AchievementComponent struct {
	Achievements map[int]int `wire:"achieveInfo"`
	Points       int64       `wire:"achievePoint"`
}

type SpiralComponent struct {
	Groups   map[int]JourneySpiralGroup `wire:"groupInfoDict"`
	Missions map[int]int                `wire:"missionFinishDict"`
}

type DispatchComponent struct {
	Acceptable map[DispatchID]DispatchMission `wire:"canAccept"`
	Active     map[DispatchID]DispatchMission `wire:"inProgress"`
	Count      int                            `wire:"todayDispatchCount"`
	Refreshes  int                            `wire:"todayRefreshCount"`
}

type RogueComponent struct {
	Status          int                        `wire:"status"`
	Battles         []BattleID                 `wire:"battles"`
	WeeklyCoin      int64                      `wire:"weekCoin"`
	WeeklyScore     int64                      `wire:"weekScore"`
	WeeklyRewards   map[int]int                `wire:"weekScoreRwded"`
	PVEBattle       RoguePVEBattleInfo         `wire:"roguePveBattleInfo"`
	AllFirstRewards int                        `wire:"firstRwdAllGet"`
	Heroes          map[int]InstanceID         `wire:"heroes"`
	Shop            map[int]RogueShopEntry     `wire:"shop"`
	Objects         map[InstanceID]RogueObject `wire:"rogueObjMgr"`
	AddHeroPool     []HeroID                   `wire:"addHeroPool"`
	Banned          []HeroID                   `wire:"baned"`
	RogueID         int                        `wire:"rogueId"`
	PVEID           int                        `wire:"roguePveId"`
	Version         int                        `wire:"rogueVer"`
	PVEVersion      int                        `wire:"roguePveVer"`
	Index           int                        `wire:"idx"`
	Life            int                        `wire:"life"`
	ChestLevel      int                        `wire:"chestlv"`
	TotalWins       int                        `wire:"totalWin"`
	FinalWins       int                        `wire:"finalWinNum"`
	RankScore       int64                      `wire:"rankScore"`
	TutorialPassed  int                        `wire:"passNewbie"`
	Difficulty      map[int]RogueDifficulty    `wire:"difficultInfo"`
	Round           int                        `wire:"round"`
	TotalTime       int64                      `wire:"totalTime"`
	MaxDifficulty   int                        `wire:"maxHard"`
	Coin            int64                      `wire:"coin"`
	FreeRefresh     int                        `wire:"freeRefresh"`
	RefreshCount    int                        `wire:"refreshCount"`
	SaleCount       int                        `wire:"saleCount"`
	MergeCount      int                        `wire:"mergeCount"`
	LastDeaths      []HeroID                   `wire:"lastBattleDieActors"`
	EventGroup      int                        `wire:"evtGroupId"`
	EventIndex      int                        `wire:"evtIdx"`
	Records         []string                   `wire:"rogueRecord"`
	PVERecords      []string                   `wire:"roguePveRec"`
}
