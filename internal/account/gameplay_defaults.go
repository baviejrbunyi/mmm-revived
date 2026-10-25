package account

import "mmm-revived/internal/model"

type GameplayDefaults struct {
	Retail RetailRules `json:"retail"`
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
	result := make([]int, 0, len(rule.Battles))
	for _, entry := range rule.Battles {
		if battle, mapped := r.PVPBattleMaps[entry]; mapped && r.BattleModes[battle] == RealtimeBattleMode {
			result = append(result, int(battle))
		}
	}
	if len(result) == 0 {
		return nil, rejected("matchmaking battle data is unavailable")
	}
	return result, nil
}
