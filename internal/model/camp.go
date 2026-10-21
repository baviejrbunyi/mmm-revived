package model

type RetailCampRun struct {
	Node    int      `json:"node"`
	Battle  BattleID `json:"battle"`
	Started int64    `json:"started"`
}

type RetailCampPending struct {
	Node     int      `json:"node"`
	Battle   BattleID `json:"battle"`
	Duration int64    `json:"duration"`
	Score    int64    `json:"score"`
}

type RetailCampState struct {
	BestScores map[BattleID]int64 `json:"best_scores"`
	Run        *RetailCampRun     `json:"run,omitempty"`
	Pending    *RetailCampPending `json:"pending,omitempty"`
	Settled    *RetailCampPending `json:"settled,omitempty"`
}

type RetailCampInfo struct {
	BestScores map[BattleID]int64 `wire:"bestScoreDict"`
	Pending    []int64            `wire:"waitSettleBattleNode"`
}

func (RetailCampInfo) isActivityInfo() {}

func (s RetailCampState) Snapshot() RetailCampInfo {
	info := RetailCampInfo{BestScores: s.BestScores, Pending: []int64{}}
	if s.Pending != nil {
		info.Pending = []int64{int64(s.Pending.Node), int64(s.Pending.Battle), s.Pending.Duration, s.Pending.Score}
	}
	return info
}
