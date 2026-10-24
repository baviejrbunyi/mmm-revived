package model

type ItemComponent struct {
	Normal  map[ItemID]int64       `wire:"normals"`
	Limited map[ItemID]LimitedItem `wire:"limitNormals"`
	Crafts  []CraftID              `wire:"crafts"`
}

type LimitedItem struct {
	HeroGain []HeroID `wire:"heroGain"`
}

type EnergyComponent struct {
	Energy      int64 `wire:"energy"`
	Bank        int64 `wire:"energyBank"`
	RefreshedAt int64 `wire:"refreshTime"`
	Purchases   int   `wire:"buyCount"`
}

type ShopComponent struct {
	Bought map[ShopID]int `wire:"boughtNum"`
	Counts map[ShopID]int `wire:"buyCount"`
}

type DrawComponent struct {
	Pools          map[DrawPoolID]DrawPoolState `wire:"drawInfo"`
	Records        []DrawRecord                 `wire:"drawRec"`
	SharedPity     map[int]DrawCounter          `wire:"shareInsure"`
	Open           map[DrawPoolID]int           `wire:"poolOpen,omitempty"`
	DreamstoneShop int                          `wire:"dreamstoneShop,omitempty"`
}

type MonthlyPassComponent struct {
	ExpiresAt   int64 `wire:"expireTime"`
	LastReward  int64 `wire:"lastDailyRewardTime"`
	RewardReady bool  `wire:"canDailyRwd"`
}

type BattlePassComponent struct {
	PassID       int         `wire:"passId"`
	Level        int         `wire:"passLevel"`
	Experience   int64       `wire:"passExp"`
	WeeklyXP     int64       `wire:"weekPassExp"`
	ActiveType   int         `wire:"activeType"`
	CurrentTasks []TaskID    `wire:"currentTaskIds"`
	WeeklyTasks  []TaskID    `wire:"weekTaskIds"`
	FreeRewards  map[int]int `wire:"freePassReward"`
	PaidRewards  map[int]int `wire:"feePassReward"`
}

type Currency string

type PurchaseComponent struct {
	TotalPrice      map[Currency]int64  `wire:"totalPrice"`
	TotalPriceCeil  map[Currency]int64  `wire:"totalPriceCeil"`
	FreePrice       map[Currency]int64  `wire:"totalFreePrice"`
	FreePriceCeil   map[Currency]int64  `wire:"totalFreePriceCeil"`
	FirstPayment    map[int]map[int]int `wire:"firstPayMark"`
	FirstPurchase   int64               `wire:"firstShipTime"`
	LastPurchase    []int64             `wire:"lastShipMark"`
	ReturnedPayment bool                `wire:"testReturnPayMark"`
	VIPBound        int                 `wire:"bindVIPSupport"`
}

type PopupPackComponent struct {
	Current  map[int]int `wire:"currPack"`
	Finished map[int]int `wire:"finishedPack"`
	ClientID int         `wire:"clientPackId"`
}

type HeroPurchaseComponent struct {
	Packs map[int]SystemHeroPack `wire:"packInfo"`
}

type BenefitsComponent struct {
	Limited []string `wire:"ltsHistory"`
	Weekly  []string `wire:"weeklyHistory"`
}
