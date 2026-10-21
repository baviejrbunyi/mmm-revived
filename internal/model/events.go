package model

type EventProgress struct {
	Season SeasonProgress `json:"season"`
	Boss   BossProgress   `json:"boss"`
}

type SeasonProgress struct {
	Initialized bool  `json:"initialized"`
	ID          int   `json:"id"`
	Level       int   `json:"level"`
	Experience  int64 `json:"experience"`
	Nodes       []int `json:"nodes"`
	Resets      int   `json:"resets"`
	Week        int64 `json:"week"`
}

type BossProgress struct {
	Open        int           `json:"open"`
	Day         int64         `json:"day"`
	Count       int           `json:"count"`
	TotalScore  int64         `json:"total_score"`
	Best        map[int]int64 `json:"best"`
	Daily       map[int]int   `json:"daily"`
	Rewards     []int         `json:"rewards"`
	RankClaimed []int         `json:"rank_claimed"`
	Pending     *BossResult   `json:"pending"`
}

type BossResult struct {
	Mission int   `json:"mission"`
	Score   int64 `json:"score"`
	First   bool  `json:"first"`
}
