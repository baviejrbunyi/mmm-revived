package model

type ShopID int32
type CraftID int32
type DrawPoolID int32

type ShopPurchase struct {
	Total      int   `json:"total"`
	Count      int   `json:"count"`
	Period     int64 `json:"period"`
	PeriodType int   `json:"period_type"`
}

type PlayerEnergy struct {
	Initialized bool  `json:"initialized"`
	Bank        int64 `json:"bank"`
	Bought      int   `json:"bought"`
	Day         int64 `json:"day"`
	Refreshed   int64 `json:"refreshed"`
}
