package model

type PVPStatus int

const (
	PVPInactive PVPStatus = iota
	PVPMatching
	PVPBattling
	PVPChooseHero
	PVPUpgradeHero
	PVPRedraft
	PVPSettle
	PVPUpgradeMaximum
)

type PVPHero struct {
	ID       HeroID      `json:"id" wire:"heroId"`
	Level    int         `json:"level" wire:"lv"`
	Star     int         `json:"star" wire:"star"`
	Talents  map[int]int `json:"-" wire:"talents"`
	AwakeIDs []int       `json:"-" wire:"awakeIds"`
}

type PVPActivity struct {
	InitializedAt int64     `json:"initialized_at" wire:"initTime"`
	Round         int       `json:"round" wire:"round"`
	Upgraded      int       `json:"upgraded" wire:"uped"`
	UpgradeType   string    `json:"upgrade_type" wire:"upType"`
	Rerolled      bool      `json:"rerolled" wire:"rerolled"`
	Status        PVPStatus `json:"status" wire:"status"`
	Lineup        []PVPHero `json:"lineup" wire:"lineup"`
	Wins          int       `json:"wins" wire:"win"`
	MaxWins       int       `json:"max_wins" wire:"maxWin"`
	Points        int64     `json:"points" wire:"point"`
	History       []int     `json:"history" wire:"win_lose"`
	Chance        int       `json:"chance" wire:"chance"`
	NewHeroes     []HeroID  `json:"new_heroes" wire:"newHeroPool"`
	Upgrades      []int     `json:"upgrades" wire:"upHeroPool"`
}

type NameCard struct {
	Signature   string `wire:"sign"`
	Story       string `wire:"story"`
	Offline     string `wire:"offline"`
	Online      string `wire:"online"`
	SeasonLevel string `wire:"slv"`
	PVP         string `wire:"pvp"`
	Heroes      string `wire:"heroes"`
	Support     string `wire:"supportHeroes"`
	Show        string `wire:"show"`
	Badge       string `wire:"achieveBadge"`
	Background  string `wire:"bg"`
	Host        string `wire:"logic_hostnum"`
	Badges      string `wire:"ownAchieveBadgeInfo"`
	Tags        string `wire:"showTags"`
	Points      string `wire:"achievePoint"`
	Created     string `wire:"createTime"`
	Skins       string `wire:"skins"`
	Season      string `wire:"season"`
	Server      string `wire:"serverset"`
	GuildID     string `wire:"guildId"`
	GuildTitle  string `wire:"guildTitle"`
	Name        string `wire:"name"`
	Head        string `wire:"h"`
	Frame       string `wire:"hf"`
	Level       string `wire:"lv"`
	Deleted     string `wire:"deleted"`
}
