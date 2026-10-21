package inventory

import (
	"errors"
	"math"

	"mmm-revived/internal/account"
	"mmm-revived/internal/handlers/hero"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
)

type CommodityPurchase struct {
	ID    model.ShopID
	Count int
}
type CraftItem struct {
	ID        model.CraftID
	Count     int
	Materials map[model.ItemID]int64
	Up        int
	Position  int
	Show      bool
	Notify    bool
}
type UseItem struct {
	Item      model.ItemID
	Count     int64
	Selection int
	Show      bool
}

func DecodeEnergyCount(c protocol.Call) (int, error) {
	count, err := c.Number(0, 1, 100)
	return int(count), err
}

func DecodeAmount(c protocol.Call) (int64, error) { return c.Number(0, 1, 1000000000) }

func DecodeCommodity(c protocol.Call) (CommodityPurchase, error) {
	id, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return CommodityPurchase{}, err
	}
	count, err := c.Number(1, 1, account.MaxBulkCount)
	return CommodityPurchase{ID: model.ShopID(id), Count: int(count)}, err
}

func DecodeCraft(c protocol.Call) (CraftItem, error) {
	if len(c.Args) != 7 {
		return CraftItem{}, errors.New("doCraft requires seven arguments")
	}
	id, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return CraftItem{}, err
	}
	count, err := c.Number(1, 1, account.MaxBulkCount)
	if err != nil {
		return CraftItem{}, err
	}
	materials, err := c.ItemQuantities(2)
	if err != nil {
		return CraftItem{}, err
	}
	up, err := c.Number(3, 0, math.MaxInt32)
	if err != nil {
		return CraftItem{}, err
	}
	position, err := c.Number(4, 0, 6)
	if err != nil {
		return CraftItem{}, err
	}
	show, err := c.Number(5, 0, 1)
	if err != nil {
		return CraftItem{}, err
	}
	notify, err := c.Number(6, 0, 1)
	return CraftItem{ID: model.CraftID(id), Count: int(count), Materials: materials, Up: int(up), Position: int(position), Show: show == 1, Notify: notify == 1}, err
}

func DecodeUseItem(c protocol.Call) (UseItem, error) {
	id, err := c.Number(0, 1, math.MaxInt32)
	if err != nil {
		return UseItem{}, err
	}
	count, err := c.Number(1, 1, account.MaxBulkCount)
	if err != nil {
		return UseItem{}, err
	}
	var info struct {
		Selection int  `json:"selectIdx"`
		Show      *int `json:"showRwd"`
	}
	if err = c.Record(2, &info); err != nil {
		return UseItem{}, err
	}
	if info.Selection < 0 || info.Selection > 1000 || info.Show != nil && (*info.Show < 0 || *info.Show > 1) {
		return UseItem{}, errors.New("invalid item options")
	}
	return UseItem{Item: model.ItemID(id), Count: count, Selection: info.Selection, Show: info.Show == nil || *info.Show == 1}, nil
}

type HighlightAwakening struct {
	Instance  model.InstanceID
	Consumed  []model.InstanceID
	Materials map[model.ItemID]int64
}

type ArtifactRefinement struct {
	Instance model.InstanceID
	Locked   string
}

type ProficiencyRewards struct {
	Heroes []model.HeroID
	Level  int
}

type FreeReset[T any] struct {
	Activity model.ActivityID
	Target   T
}

func DecodeInstance(c protocol.Call) (model.InstanceID, error) { return c.InstanceID(0) }

func DecodeInstances(c protocol.Call) ([]model.InstanceID, error) {
	if len(c.Args) > 0 {
		if _, single := c.Args[0].(string); single {
			id, err := c.InstanceID(0)
			return []model.InstanceID{id}, err
		}
	}
	return c.InstanceIDs(0, 5000, false)
}

func DecodeFreeHeroReset(c protocol.Call) (FreeReset[model.HeroID], error) {
	activity, err := protocol.DecodeActivity(c)
	if err != nil {
		return FreeReset[model.HeroID]{}, err
	}
	hero, err := c.Number(1, 1, math.MaxInt32)
	return FreeReset[model.HeroID]{Activity: activity, Target: model.HeroID(hero)}, err
}

func DecodeFreeHighlightReset(c protocol.Call) (FreeReset[model.InstanceID], error) {
	activity, err := protocol.DecodeActivity(c)
	if err != nil {
		return FreeReset[model.InstanceID]{}, err
	}
	id, err := c.InstanceID(1)
	return FreeReset[model.InstanceID]{Activity: activity, Target: id}, err
}

func DecodeHighlightAwakening(c protocol.Call) (HighlightAwakening, error) {
	id, err := c.InstanceID(0)
	if err != nil {
		return HighlightAwakening{}, err
	}
	consumed, err := c.InstanceIDs(1, 30, true)
	if err != nil {
		return HighlightAwakening{}, err
	}
	items, err := c.ItemQuantities(2)
	return HighlightAwakening{Instance: id, Consumed: consumed, Materials: items}, err
}

func DecodeRefinement(c protocol.Call) (ArtifactRefinement, error) {
	id, err := c.InstanceID(0)
	if err != nil {
		return ArtifactRefinement{}, err
	}
	locked, err := c.Text(1, 64)
	return ArtifactRefinement{Instance: id, Locked: locked}, err
}

func DecodeProficiency(c protocol.Call) (ProficiencyRewards, error) {
	hero, err := hero.DecodeHero(c)
	if err != nil {
		return ProficiencyRewards{}, err
	}
	level, err := c.Number(1, 1, 5)
	return ProficiencyRewards{Heroes: []model.HeroID{hero}, Level: int(level)}, err
}

func DecodeProficiencyList(c protocol.Call) (ProficiencyRewards, error) {
	ids, err := c.Integers(0, 100, 1, math.MaxInt32)
	heroes := make([]model.HeroID, 0, len(ids))
	for _, id := range ids {
		heroes = append(heroes, model.HeroID(id))
	}
	return ProficiencyRewards{Heroes: heroes}, err
}
