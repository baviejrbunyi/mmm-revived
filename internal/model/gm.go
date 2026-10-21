package model

type GMCoverage struct {
	Enabled  bool             `json:"enabled"`
	User     string           `json:"user"`
	Started  int64            `json:"started"`
	Reported int64            `json:"reported"`
	Methods  map[string]int64 `json:"methods"`
	Issues   []string         `json:"issues"`
}

type GMState struct {
	History       string     `json:"history"`
	ClockOffset   *int64     `json:"clock_offset,omitempty"`
	LastCommand   string     `json:"last_command"`
	LastError     string     `json:"last_error"`
	LastErrorTime int64      `json:"last_error_time"`
	Commands      int64      `json:"commands"`
	Coverage      GMCoverage `json:"coverage"`
}

type GMCommandKind uint8

const (
	GMUnknown GMCommandKind = iota
	GMList
	GMGetAll
	GMAddItem
	GMSetItem
	GMTeamLevel
	GMHeroLevel
	GMHeroStar
	GMAddHero
	GMGuideFinish
	GMNoviceFinish
	GMNoviceSet
	GMDreamlandFinish
	GMNextDay
	GMGetTime
	GMSetTime
	GMMail
	GMSetEnergy
	GMBossReset
	GMScienceReset
	GMScienceLevel
	GMRogueInitialize
	GMRoguePVEInitialize
	GMSave
	GMCheckChapters
	GMResetBattle
	GMGuild
)

type GMCommand struct {
	Kind  GMCommandKind
	Name  string
	ID    int64
	Value int64
	Text  string
}

type GMCommandResult struct {
	Text        string
	Items       map[ItemID]int64
	Clock       *int64
	Mail        int
	ResetBattle bool
}

type GMComponent struct {
	History string `wire:"gmStrRec"`
}
