package model

type BasicComponent struct {
	TeamName          string `wire:"teamName"`
	Level             int    `wire:"lv"`
	Break             int    `wire:"breakLv"`
	Experience        int64  `wire:"exp"`
	Gold              int64  `wire:"gold"`
	Diamond           int64  `wire:"diamond"`
	Yuanbao           int64  `wire:"yuanbao"`
	RewardedLevels    []int  `wire:"rwdedLv"`
	Signature         string `wire:"signature"`
	SignatureExpiry   int64  `wire:"signatureLimitExpire"`
	ProtectionCode    string `wire:"childrenProtectPwd"`
	ProtectionTime    int64  `wire:"childrenProtectRestTime"`
	ProtectionUpdated int64  `wire:"childrenProtectRestTimeLastCalc"`
	Mobile            string `wire:"bindMobile"`
	MobileReward      bool   `wire:"bindMobileReward"`
}

type LobbyHero struct {
	Hero HeroID `wire:"heroId"`
	Skin SkinID `wire:"skin"`
}

type LobbyComponent struct {
	Heroes     map[int]LobbyHero `wire:"heroShow"`
	Guides     map[string]string `wire:"guides"`
	GuideBooks []int             `wire:"guideBookOpen"`
	RedDots    map[string]int64  `wire:"redDot"`
	HiddenIDs  []string          `wire:"notShowId"`
}

type NoviceComponent struct {
	Task   TaskID         `wire:"noviceId"`
	Status int            `wire:"noviceStatus"`
	Node   int            `wire:"curNode"`
	Nodes  map[int]int    `wire:"sceneNodes"`
	Place  map[string]int `wire:"loc"`
}

const (
	LineupSlots       = 9
	LineupTypes       = 9
	DefaultLineupType = 1
)

type Lineup struct {
	Name   string   `wire:"name"`
	Type   int      `wire:"lineupType"`
	Heroes []HeroID `wire:"heroIds"`
}

type LimitedCosmetic struct {
	Valid int64 `wire:"valid"`
}

type LineupComponent struct {
	Lineups map[int]Lineup `wire:"lineups"`
}

type AppearanceComponent struct {
	Head          ItemID                  `wire:"head"`
	Frame         ItemID                  `wire:"headFrame"`
	Background    ItemID                  `wire:"background"`
	Heads         []ItemID                `wire:"ownHeads"`
	Frames        []ItemID                `wire:"ownHeadFrames"`
	Backgrounds   []ItemID                `wire:"ownBackground"`
	Support       SupportHeroes           `wire:"supportHeroes"`
	Tags          HonorTags               `wire:"showTags"`
	Badge         ItemID                  `wire:"achieveBadge"`
	BadgeInfo     map[int]int             `wire:"ownAchieveBadgeInfo"`
	LimitedHeads  map[int]LimitedCosmetic `wire:"limitHeads"`
	LimitedFrames map[int]LimitedCosmetic `wire:"limitHeadFrames"`
	ActMaxScore   map[int]int64           `wire:"actMaxScore"`
}
