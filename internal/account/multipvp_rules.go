package account

import "mmm-revived/internal/model"

type RetailPVPRules struct {
	DailyDefenses    int                                      `json:"daily_defenses"`
	Activities       map[model.ActivityID]JourneyActivityRule `json:"activities"`
	Main             map[int]RetailPVPMainRule                `json:"main"`
	Points           map[int64]RetailPVPPointRule             `json:"points"`
	Mines            map[int]RetailPVPMineRule                `json:"mines"`
	AI               map[int]RetailPVPAIRule                  `json:"ai"`
	DailyHeroAttacks int                                      `json:"daily_hero_attacks"`
	MaximumRevenge   int                                      `json:"maximum_revenge"`
}

type RetailPVPMainRule struct {
	DailyRefresh int   `json:"daily_refresh"`
	Mines        []int `json:"mine_map"`
	WinMine      int64 `json:"win_base_mine"`
	WinPoint     int64 `json:"win_base_point"`
}

type RetailPVPPointRule struct {
	AILevel    int   `json:"ai_level"`
	AIType     int   `json:"ai_type"`
	Challenges int   `json:"challenge"`
	Mines      int   `json:"mine"`
	MineLimit  int64 `json:"mine_limit"`
	PointLimit int64 `json:"point_limit"`
}

type RetailPVPMineRule struct {
	DefeatPenalty     int              `json:"minus_eff"`
	DefenseBonus      int              `json:"bonus_eff"`
	Battles           []model.BattleID `json:"battle_mode"`
	BaseMine          int64            `json:"base_mine"`
	BasePoint         int64            `json:"base_point"`
	InitialEfficiency int              `json:"init_eff"`
	MinimumEfficiency int              `json:"lower_limit_eff"`
	MaximumEfficiency int              `json:"upper_limit_eff"`
	AutoGrow          int              `json:"auto_grow"`
}

type RetailPVPAIRule struct {
	Mine    int                  `json:"ai_mine"`
	Type    int                  `json:"ai_type"`
	Version int                  `json:"version"`
	Lineup  map[model.HeroID]int `json:"lineup"`
}

func (r RetailPVPRules) PointRule(score int64) (RetailPVPPointRule, bool) {
	var chosen RetailPVPPointRule
	threshold := int64(-1)
	for point, row := range r.Points {
		if point <= score && point > threshold {
			threshold, chosen = point, row
		}
	}
	return chosen, threshold >= 0
}
