package account

import (
	"maps"

	"mmm-revived/internal/model"
)

const (
	DefaultResetActivity model.ActivityID = 3
	DefaultResetDuration int64            = 31 * 86400
	DefaultPVPActivity   model.ActivityID = 1203
	DefaultTeamLevel                      = 50
	DefaultHeroLevel                      = 1
	DefaultItemQuantity  int64            = 9999999
	DefaultEnergy        int64            = 9999
	DefaultHead          model.ItemID     = 5001
	DefaultHeadFrame     model.ItemID     = 6001
	DefaultBackground    model.ItemID     = 9001
)

func NewAccount(uid model.UserID, name string, createdAt int64, catalog Catalog) Account {
	a := Account{
		UID:       uid,
		Name:      name,
		TeamName:  name,
		CreatedAt: createdAt,
		Level:     DefaultTeamLevel,
		Items:     map[model.ItemID]int64{},
		Heroes:    map[model.HeroID]model.Hero{},
		Guides:    map[string]string{},
	}
	for _, id := range catalog.Items {
		a.Items[id] = DefaultItemQuantity
	}
	for id, template := range catalog.Heroes {
		talents := make(map[int]int, len(template.Talents))
		maps.Copy(talents, template.Talents)
		a.Heroes[id] = model.Hero{
			ID:         id,
			Level:      DefaultHeroLevel,
			Star:       template.Star,
			Talents:    talents,
			AwakeIDs:   []int{},
			Artifacts:  map[int]model.InstanceID{},
			Highlights: map[int]model.InstanceID{},
		}
	}
	return a
}
