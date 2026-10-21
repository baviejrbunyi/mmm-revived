package model

type RetailPVPState struct {
	Pending        *RetailPVPBattle            `json:"pending,omitempty" wire:"-"`
	DailyDefenses  int                         `json:"daily_defenses" wire:"-"`
	Initialized    int64                       `json:"initialized" wire:"initTime"`
	Day            int64                       `json:"day" wire:"-"`
	Point          int64                       `json:"point" wire:"point"`
	BoxMineCoin    int64                       `json:"box_mine_coin" wire:"boxMineCoin"`
	BoxPoint       int64                       `json:"box_point" wire:"boxPoint"`
	MineRemainder  int64                       `json:"mine_remainder" wire:"-"`
	PointRemainder int64                       `json:"point_remainder" wire:"-"`
	LastBoxRequest int64                       `json:"last_box_request" wire:"lastBoxReq"`
	Refreshed      int                         `json:"refreshed" wire:"refreshed"`
	SettleStatus   int                         `json:"settle_status" wire:"settleSt"`
	Mines          map[string]RetailPVPMine    `json:"mines" wire:"mineInfo"`
	Enemies        map[int]RetailPVPTarget     `json:"enemies" wire:"enemies"`
	Revenges       map[UserID]RetailPVPRevenge `json:"revenges" wire:"revenges"`
	Attacked       []int64                     `json:"attacked" wire:"attacked"`
	DailyUsed      map[HeroID]int              `json:"daily_used" wire:"dailyUsed"`
	Records        []RetailPVPRecord           `json:"records" wire:"records"`
}

const RankRetailPVP RankMode = "multi_pvp"

type RetailPVPBattle struct {
	Defender      AsyncDefense    `json:"defender"`
	Card          RetailPVPCard   `json:"card"`
	RevengeRecord string          `json:"revenge_record"`
	Battle        BattleID        `json:"battle"`
	Token         string          `json:"token"`
	Started       int64           `json:"started"`
	Target        RetailPVPTarget `json:"target"`
	RevengeIndex  int             `json:"revenge_index"`
	EnemyIndex    int             `json:"enemy_index"`
	Locked        bool            `json:"locked"`
	Heroes        []HeroID        `json:"heroes"`
}

type RetailPVPCard struct {
	UID    int64  `wire:"uid"`
	Name   string `wire:"name"`
	Level  int    `wire:"lv"`
	Head   ItemID `wire:"h"`
	Frame  ItemID `wire:"hf"`
	Host   int    `wire:"logic_hostnum"`
	Server int    `wire:"serverset"`
}

func (RetailPVPState) isActivityInfo() {}

type RetailPVPPosition struct {
	Born int `json:"bornId" wire:"bornId"`
}

type RetailPVPMine struct {
	Mine       int                          `json:"mine_id" wire:"mineId"`
	Efficiency int                          `json:"efficiency" wire:"eff"`
	Defense    map[HeroID]RetailPVPPosition `json:"defense" wire:"pvpInfo"`
}

type RetailPVPTarget struct {
	UID        int64  `json:"uid" wire:"euid"`
	Mine       int    `json:"mine_id" wire:"mineId"`
	MineIndex  string `json:"mine_index" wire:"mineIdx"`
	Efficiency int    `json:"efficiency" wire:"eff"`
	Level      int    `json:"level" wire:"lv"`
	Defense    string `json:"defense" wire:"defendInfo"`
	AIDefense  string `json:"ai_defense" wire:"AIDefendInfo"`
}

type RetailPVPRevenge struct {
	DailyAttacks int              `json:"daily_attacks" wire:"dailyAtt"`
	Refreshed    int64            `json:"refreshed" wire:"refreshT"`
	First        *RetailPVPTarget `json:"first,omitempty" wire:"0,omitempty"`
	Second       *RetailPVPTarget `json:"second,omitempty" wire:"1,omitempty"`
	Third        *RetailPVPTarget `json:"third,omitempty" wire:"2,omitempty"`
}

type RetailPVPRecord struct {
	UID             int64  `json:"uid" wire:"euid"`
	Win             int    `json:"win" wire:"win"`
	Mine            int    `json:"mine_id" wire:"mineId"`
	MineGain        int64  `json:"mine_gain" wire:"gainMine"`
	PointGain       int64  `json:"point_gain" wire:"gainPoint"`
	EfficiencyDelta int    `json:"efficiency_delta" wire:"effDiff"`
	Revenged        int    `json:"revenged" wire:"revenged"`
	AILevel         int    `json:"ai_level" wire:"aiLv"`
	BattleInfo      string `json:"battle_info" wire:"bInfo"`
	Attack          bool   `json:"attack" wire:"-"`
}
