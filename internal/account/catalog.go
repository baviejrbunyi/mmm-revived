package account

import "mmm-revived/internal/model"

type Catalog struct {
	Activities   []model.ActivityID            `json:"activities,omitempty"`
	Cosmetics    map[int][]model.ItemID        `json:"cosmetics,omitempty"`
	Items        []model.ItemID                `json:"items"`
	Heroes       map[model.HeroID]HeroTemplate `json:"heroes"`
	Skins        []model.SkinID                `json:"skins"`
	Artifacts    []model.ArtifactID            `json:"artifacts"`
	Highlights   []model.HighlightID           `json:"highlights"`
	DefaultSkins []model.SkinID                `json:"-"`
	SkinItems    map[model.ItemID]model.SkinID `json:"-"`
}

type HeroTemplate struct {
	Star    int         `json:"star"`
	Talents map[int]int `json:"talents"`
}
