package account

import "mmm-revived/internal/model"

type BondRule struct {
	Heroes []model.HeroID `json:"heroes"`
	Levels []BondLevel    `json:"levels"`
}

type BondLevel struct {
	Heroes     int              `json:"hero_num"`
	Attributes map[string]int64 `json:"extra_attr"`
}

func (a *Account) BondAttributes(r *Rules) map[string]int64 {
	attributes := map[string]int64{}
	if r == nil {
		return attributes
	}
	for _, bond := range r.Bonds {
		count := 0
		for _, id := range bond.Heroes {
			if hero, owned := a.Heroes[id]; owned && hero.Trial != 1 {
				count++
			}
		}
		for _, level := range bond.Levels {
			if count < level.Heroes {
				break
			}
			for name, value := range level.Attributes {
				attributes[name] += value
			}
		}
	}
	return attributes
}
