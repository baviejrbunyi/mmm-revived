package model

type Selection struct {
	Hero HeroID `json:"heroId" wire:"heroId"`
	Born int    `json:"bornId" wire:"bornId"`
}

type MatchedPlayer struct {
	UID    UserID          `wire:"uid"`
	Heroes map[HeroID]Hero `wire:"limitedHeros"`
	Seed   int64           `wire:"aiRandSeed"`
}

type BattlePlayer struct {
	UID      UserID          `wire:"uid"`
	Heroes   map[HeroID]Hero `wire:"limitedHeroInfo"`
	Selected []Selection     `wire:"selectedHeroInfo"`
	Callback Empty           `wire:"callbackData"`
}

type MatchReport struct {
	Player   UserID
	GID      string
	Winner   UserID
	Opponent UserID
	Frame    int
}
