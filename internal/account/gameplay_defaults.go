package account

import "mmm-revived/internal/model"

type GameplayDefaults struct {
	RealtimeBattles []int       `json:"realtime_battles"`
	Retail          RetailRules `json:"retail"`
}

type RetailRules struct {
	Level         int                    `json:"level"`
	Heroes        []model.HeroID         `json:"heroes"`
	Energy        int64                  `json:"energy"`
	Items         map[model.ItemID]int64 `json:"items"`
	Head          model.ItemID           `json:"head"`
	Frame         model.ItemID           `json:"frame"`
	Background    model.ItemID           `json:"background"`
	NoviceTask    model.TaskID           `json:"novice_task"`
	EnergySeconds int64                  `json:"energy_seconds"`
	BankLimit     int64                  `json:"bank_limit"`
	BankSeconds   int64                  `json:"bank_seconds"`
	TeamBreaks    map[int]TeamBreakRule  `json:"team_breaks"`
	DefaultSkins  []model.SkinID         `json:"default_skins"`
	Daily         RetailDailyRules       `json:"daily"`
}

type TeamBreakRule struct {
	Battle model.BattleID `json:"battle"`
	Reward model.RewardID `json:"reward"`
	Task   model.TaskID   `json:"task"`
}

type RetailDailyRules struct {
	NewbieLevel int               `json:"newbie_level"`
	NewbieTasks []model.TaskID    `json:"newbie_tasks"`
	Counts      map[int]int       `json:"counts"`
	Tasks       []RetailDailyTask `json:"tasks"`
	PointItem   model.ItemID      `json:"point_item"`
	PointTasks  []model.TaskID    `json:"point_tasks"`
}

type RetailDailyTask struct {
	Task   model.TaskID `json:"task"`
	Type   int          `json:"type"`
	Level  int          `json:"level"`
	Weight int64        `json:"weight"`
}

const RealtimeBattleMode = 7

func (r *Rules) RealtimeBattlePool(activity model.ActivityID) ([]int, error) {
	rule, known := r.PVP[activity]
	if !known {
		return nil, rejected("unknown matchmaking activity")
	}
	valid := func(candidates []int) []int {
		result := make([]int, 0, len(candidates))
		for _, battle := range candidates {
			if r.BattleModes[model.BattleID(battle)] == RealtimeBattleMode {
				result = append(result, battle)
			}
		}
		return result
	}
	result := valid(rule.Battles)
	if len(result) == 0 {
		result = valid(r.Defaults.RealtimeBattles)
	}
	if len(result) == 0 {
		return nil, rejected("matchmaking battle data is unavailable")
	}
	return result, nil
}
