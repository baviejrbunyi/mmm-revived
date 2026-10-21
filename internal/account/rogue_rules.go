package account

import "mmm-revived/internal/model"

type RogueRules struct {
	Runs            map[int]RogueRunRule   `json:"runs"`
	Shops           map[int]RogueShopRule  `json:"shops"`
	Relics          map[int]RogueRelicRule `json:"relics"`
	Chests          map[int]RogueChestRule `json:"chests"`
	PVE             map[int]RoguePVERule   `json:"pve"`
	Events          map[int]RogueEventRule `json:"events"`
	EventGroups     map[int][]int          `json:"event_groups"`
	Buffs           map[int]RogueBuffRule  `json:"buffs"`
	FirstRewards    map[int]model.RewardID `json:"first_rewards"`
	WeekRewards     map[int]model.RewardID `json:"week_rewards"`
	PVEFirstRewards map[int]model.RewardID `json:"pve_first_rewards"`
	PVEWeekRewards  map[int]model.RewardID `json:"pve_week_rewards"`
	WinReward       model.RewardID         `json:"win_reward"`
	PVEWinRewards   []model.RewardID       `json:"pve_win_rewards"`
	RelicPrices     []int64                `json:"relic_prices"`
	RefreshPrice    int64                  `json:"refresh_price"`
	SalePrice       int64                  `json:"sale_price"`
	Life            int                    `json:"life"`
	HeroLevel       int                    `json:"hero_level"`
	RechooseLimit   int                    `json:"rechoose_limit"`
	CoinLimit       int64                  `json:"coin_limit"`
	MaximumWins     int                    `json:"max_wins"`
}

type RogueRunRule struct {
	BattleSequence []string         `json:"battle_seq"`
	BattlePool     []model.BattleID `json:"battle_pool"`
	HeroPool       [][]model.HeroID `json:"hero_pool"`
	RelicPool      []int            `json:"relic_pool"`
	Shops          []int            `json:"shop_lv_seq"`
	Chests         []int            `json:"chest"`
	Levels         []int            `json:"battle_lv_seq"`
}

type RogueShopRule struct {
	Relics  int   `json:"relic_num"`
	Coin    int64 `json:"relic_coin"`
	AddHero []int `json:"add_hero"`
	Weights []int `json:"weights"`
}

type RogueRelicRule struct {
	Tier       int     `json:"tier"`
	Type       int     `json:"type"`
	HeroLevels []int   `json:"heroLvUp"`
	Skills     [][]int `json:"skill"`
	ShopSkills [][]int `json:"shopSkill"`
}

type RogueChestRule struct {
	Energy  int64                  `json:"energy_cost"`
	Cost    map[model.ItemID]int64 `json:"coin_cost"`
	Level   int                    `json:"limitLv"`
	Battle  model.BattleID         `json:"unlock_battle"`
	Rewards []model.RewardID       `json:"reward"`
}

type RoguePVERule struct {
	Battles        [][]model.BattleID `json:"battle_seq"`
	Levels         []int              `json:"battle_lv_seq"`
	EventGroups    [][]int            `json:"evt_group_seq"`
	EventPool      []int              `json:"evt_group_pool"`
	Buffs          []int              `json:"buff_pool"`
	EnemyBuffs     [][]int            `json:"enemy_buff_seq"`
	Level          int                `json:"unlock_lv"`
	Difficulty     int                `json:"difficulty"`
	Positions      [][]string         `json:"posList"`
	PositionSwitch []int              `json:"posSwitch"`
}

type RogueEventRule struct {
	Action1 int   `json:"action1"`
	Action2 int   `json:"action2"`
	Action3 int   `json:"action3"`
	Param1  []int `json:"param1"`
	Param2  []int `json:"param2"`
	Param3  []int `json:"param3"`
}

type RogueBuffRule struct {
	Tier  int   `json:"tier"`
	Cost  int64 `json:"cost"`
	Buffs []int `json:"buffid"`
}
