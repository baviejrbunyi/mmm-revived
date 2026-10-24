package account

import (
	"iter"
	"maps"
	"slices"

	"mmm-revived/internal/model"
)

type SkinCollectionRule struct {
	Scores map[int]int `json:"scores"`
	Levels []int       `json:"levels"`
}

func (r SkinCollectionRule) Level(rarities map[model.SkinID]int, owned map[model.SkinID]int) int {
	score := 0
	for id := range owned {
		score += r.Scores[rarities[id]]
	}
	level := 0
	for index, threshold := range r.Levels {
		if score >= threshold {
			level = index + 1
		}
	}
	return level
}

func (a *Account) ownedSkins(all iter.Seq[model.SkinID], defaults []model.SkinID, skinItem func(model.ItemID) model.SkinID) map[model.SkinID]int {
	owned := map[model.SkinID]int{}
	if a.Retail {
		for id, count := range a.Items {
			if skin := skinItem(id); skin != 0 && count > 0 {
				owned[skin] = 1
			}
		}
	} else {
		for skin := range all {
			owned[skin] = 1
		}
	}
	for _, skin := range defaults {
		delete(owned, skin)
	}
	return owned
}

func (a *Account) OwnedSkins(c Catalog) map[model.SkinID]int {
	return a.ownedSkins(slices.Values(c.Skins), c.DefaultSkins, func(id model.ItemID) model.SkinID { return c.SkinItems[id] })
}

func (r *Rules) skinItem(id model.ItemID) model.SkinID {
	if rule := r.ItemRules[id]; rule.Type == 1032 {
		return model.SkinID(rule.Parameter)
	}
	return 0
}

func (r *Rules) defaultSkin(skin model.SkinID) bool {
	return slices.Contains(r.Defaults.Retail.DefaultSkins, skin)
}

func (a *Account) rulesSkins(r *Rules) map[model.SkinID]int {
	return a.ownedSkins(maps.Keys(r.Skins), r.Defaults.Retail.DefaultSkins, r.skinItem)
}

func (a *Account) ownsSkin(r *Rules, skin model.SkinID) bool {
	_, owned := a.rulesSkins(r)[skin]
	return owned
}

func (a *Account) SkinCollectionLevel(r *Rules) int {
	if r == nil {
		return 0
	}
	return r.SkinCollection.Level(r.SkinRarities, a.rulesSkins(r))
}

func (a *Account) normalizeSkins(r *Rules) {
	for id, hero := range a.Heroes {
		if hero.Skin != 0 && r.defaultSkin(hero.Skin) {
			hero.Skin = 0
			a.Heroes[id] = hero
		}
	}
}
