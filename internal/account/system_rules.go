package account

import "mmm-revived/internal/model"

type SystemRules struct {
	Products         map[int]SystemProductRule       `json:"products"`
	Shops            map[model.ShopID]SystemShopRule `json:"shops"`
	Passes           map[int]map[int]SystemPassRule  `json:"passes"`
	Gifts            map[int]SystemGiftRule          `json:"gifts"`
	HeroPacks        map[int]SystemHeroPackRule      `json:"hero_packs"`
	MirageActivities []model.ActivityID              `json:"mirage_activities"`
	Recharges        map[string]int                  `json:"recharges"`
}

type SystemProductRule struct {
	Yuanbao int64 `json:"yuanbao"`
	Bonus   int64 `json:"bonus"`
	Monthly bool  `json:"monthly"`
}

type SystemShopRule struct {
	PeriodHours     int64          `json:"period_time"`
	Product         int            `json:"recharge_id"`
	DiscountProduct int            `json:"discount_recharge_id"`
	DiscountStart   int64          `json:"discount_start"`
	DiscountEnd     int64          `json:"discount_end"`
	Level           int            `json:"unlock_lv"`
	Period          int            `json:"limit_type"`
	Limit           int            `json:"limit_param"`
	Start           int64          `json:"start_time"`
	End             int64          `json:"end_time"`
	Reward          model.RewardID `json:"reward"`
	Predecessor     model.ShopID   `json:"before_shop"`
	Kind            int            `json:"shop_type"`
}

type SystemPassRule struct {
	Kind   int            `json:"kind"`
	Reward model.RewardID `json:"reward"`
}

type SystemGiftRule struct {
	Product  int            `json:"product"`
	Reward   model.RewardID `json:"reward"`
	Duration int64          `json:"duration"`
}

type SystemHeroPackRule struct {
	Product  int          `json:"product"`
	Discount int          `json:"discount"`
	Item     model.ItemID `json:"item"`
}
