package model

type DispatchID string
type DispatchRuleID int32
type DispatchMapID int32

type DispatchMission struct {
	Type    DispatchRuleID `json:"type" wire:"typeId"`
	Map     DispatchMapID  `json:"map" wire:"mapId"`
	Started int64          `json:"started" wire:"startTime"`
	Ends    int64          `json:"ends" wire:"endTime"`
	Heroes  []HeroID       `json:"heroes" wire:"usedHeroIdList"`
}

type DispatchState struct {
	Initialized bool                           `json:"initialized"`
	Day         int64                          `json:"day"`
	Refreshes   int                            `json:"refreshes"`
	Started     int                            `json:"started"`
	NextID      int64                          `json:"next_id"`
	Missions    map[DispatchID]DispatchMission `json:"missions"`
}
