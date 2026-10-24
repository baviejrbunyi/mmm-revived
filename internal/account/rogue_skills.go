package account

import (
	"maps"
	"slices"

	"mmm-revived/internal/model"
)

const (
	rogueHeroPrefix  = "h"
	rogueRelicPrefix = "r"
	rogueSkillBase   = 7000000
)

type rogueTrigger uint8

const (
	roguePrepare rogueTrigger = iota
	rogueFinish
	rogueRefresh
	rogueBuy
	rogueSale
	rogueMerged
)

func (a *Account) rogueTargets(relicType int, alive int) []model.InstanceID {
	run := &a.RogueProgress.Run
	result := make([]model.InstanceID, 0, len(run.Heroes))
	for index := range len(run.Heroes) {
		oid := run.Heroes[index]
		hero := run.Objects[oid]
		relic := run.Objects[hero.Relic]
		if relicType != 0 && relic.ID/1000 != relicType {
			continue
		}
		dead := slices.Contains(run.LastDeaths, model.HeroID(hero.ID))
		if alive == 1 && dead || alive == -1 && !dead {
			continue
		}
		result = append(result, oid)
	}
	return result
}

func (a *Account) rogueLevelTargets(kind, alive, count, amount int) error {
	targets := a.rogueTargets(kind, alive)
	if count > 0 && count < len(targets) {
		var err error
		targets, err = sample(targets, count)
		if err != nil {
			return err
		}
	}
	objects := a.RogueProgress.Run.Objects
	for _, oid := range targets {
		value := objects[oid]
		value.Level = min(10000, max(1, value.Level+amount))
		objects[oid] = value
	}
	return nil
}

func (a *Account) rogueRelicTargets(kind, count int, extra, discount int) error {
	run := &a.RogueProgress.Run
	var targets []model.InstanceID
	for _, oid := range run.Heroes {
		hero := run.Objects[oid]
		if value, exists := run.Objects[hero.Relic]; exists && (kind == 0 || value.ID/1000 == kind) {
			targets = append(targets, hero.Relic)
		}
	}
	for _, entry := range run.Shop {
		if entry.Status != 3 {
			if value, exists := run.Objects[entry.Object]; exists && (kind == 0 || value.ID/1000 == kind) {
				targets = append(targets, entry.Object)
			}
		}
	}
	slices.Sort(targets)
	if count > 0 && count < len(targets) {
		var err error
		targets, err = sample(targets, count)
		if err != nil {
			return err
		}
	}
	for _, oid := range targets {
		value := run.Objects[oid]
		value.ExtraLevel = min(10000, value.ExtraLevel+extra)
		value.Discount += int64(discount)
		run.Objects[oid] = value
	}
	return nil
}

func (a *Account) rogueEnsureShop(r *Rules, kind, count int, same bool) error {
	run := &a.RogueProgress.Run
	var pool []int
	if same {
		for _, oid := range run.Heroes {
			hero := run.Objects[oid]
			if value, exists := run.Objects[hero.Relic]; exists {
				pool = append(pool, value.ID)
			}
		}
	} else {
		for _, id := range r.Rogue.Runs[run.ID].RelicPool {
			if id/1000 == kind {
				pool = append(pool, id)
			}
		}
	}
	if len(pool) == 0 {
		return nil
	}
	existing := 0
	for _, entry := range run.Shop {
		if entry.Status != 3 && slices.Contains(pool, run.Objects[entry.Object].ID) {
			existing++
		}
	}
	for index := range len(run.Shop) {
		if existing >= count {
			break
		}
		entry := run.Shop[index]
		value := run.Objects[entry.Object]
		if entry.Status == 2 || entry.Status == 3 || slices.Contains(pool, value.ID) {
			continue
		}
		id, err := randomChoice(pool)
		if err != nil {
			return err
		}
		value.ID = id
		run.Objects[entry.Object] = value
		existing++
	}
	return nil
}

func (a *Account) rogueRestock(r *Rules, kind, level int, same bool) error {
	if err := a.rogueShop(r, true); err != nil {
		return err
	}
	return a.rogueEnsureShop(r, kind, level, same)
}

func (a *Account) rogueSkills(r *Rules, event rogueTrigger, source model.RogueObject, sourceIndex int, onHero bool) error {
	run := &a.RogueProgress.Run
	if run.RoundTriggers == nil {
		run.RoundTriggers = map[model.InstanceID]int{}
	}
	type carrier struct {
		oid   model.InstanceID
		relic model.RogueObject
		index int
	}
	carriers := make([]carrier, 0, len(run.Heroes)+1)
	for index := range len(run.Heroes) {
		hero := run.Objects[run.Heroes[index]]
		if relic, exists := run.Objects[hero.Relic]; exists {
			carriers = append(carriers, carrier{hero.Relic, relic, index})
		}
	}
	if event == rogueSale {
		carriers = append(carriers, carrier{relic: source, index: sourceIndex})
	}
	for _, carrier := range carriers {
		value, level := carrier.relic, max(1, carrier.relic.Level)
		id := value.ID
		self := source.ID == id && carrier.index == sourceIndex
		before := rogueSnapshot(run)
		var err error
		switch event {
		case roguePrepare:
			switch id {
			case 2012:
				if level == 3 {
					value.ExtraLevel += 16
					value.Level = 2
					run.Objects[carrier.oid] = value
				}
			case 2014:
				err = a.rogueEnsureShop(r, 0, level, true)
			case 3008:
				err = a.rogueLevelTargets(0, -1, 1, 3*level)
			case 3012:
				value.ExtraLevel += run.DeathCount * level
				run.Objects[carrier.oid] = value
			case 3014:
				run.FreeRefresh += 3 * level
			case 4001:
				err = a.rogueRelicTargets(0, 1, level, 0)
			case 4005:
				err = a.rogueEnsureShop(r, 4, level, false)
			case 4007:
				err = a.rogueLevelTargets(4, 1, 0, level)
			case 4008:
				value.ExtraLevel += 2 * level
				run.Objects[carrier.oid] = value
			}
		case rogueFinish:
			switch id {
			case 2010:
				total := 0
				for _, oid := range run.Heroes {
					hero := run.Objects[oid]
					total += run.Objects[hero.Relic].Level
				}
				if level == 1 {
					total /= 2
				}
				err = a.rogueLevelTargets(0, 0, 3, total)
			case 3010:
				err = a.rogueLevelTargets(0, 0, 0, level*(1+run.RefreshCount/8))
			}
		case rogueRefresh:
			switch id {
			case 3004:
				err = a.rogueLevelTargets(3, 0, level, 1)
			case 3005:
				if run.RoundTriggers[carrier.oid] < level {
					run.Coin++
					run.RoundTriggers[carrier.oid]++
					err = a.rogueEnsureShop(r, 3, 1, false)
				}
			case 3014:
				err = a.rogueLevelTargets(3, 0, 2, level)
			case 5014:
				if run.RoundTriggers[carrier.oid] < 2*level {
					for _, entry := range run.Shop {
						if entry.Status != 3 {
							relic := run.Objects[entry.Object]
							if relic.ID/1000 == 5 {
								relic.Discount++
								run.Objects[entry.Object] = relic
							}
						}
					}
					run.RoundTriggers[carrier.oid]++
				}
			}
		case rogueBuy:
			switch id {
			case 2008:
				if self {
					err = a.rogueRestock(r, 0, level, true)
				}
			case 4012:
				if self {
					total := 0
					for _, oid := range run.Heroes {
						hero := run.Objects[oid]
						relic := run.Objects[hero.Relic]
						if relic.ID/1000 == 4 && relic.ID != id {
							total += relic.ExtraLevel
						}
					}
					value.ExtraLevel += total * level
					run.Objects[carrier.oid] = value
				}
			case 5001:
				if self {
					err = a.rogueLevelTargets(0, 0, 2, level)
				}
			case 5005:
				if self {
					err = a.rogueRestock(r, 5, level, false)
				}
			case 5007:
				err = a.rogueRelicTargets(5, 1, 2*level, 0)
			case 5008:
				if self {
					err = a.rogueLevelTargets(0, 0, 0, 2*level)
				}
			case 5010:
				err = a.rogueLevelTargets(0, 0, 0, level)
			case 5012:
				if self {
					err = a.rogueLevelTargets(0, 0, 0, 2*level)
					if err == nil {
						heroOID := run.Heroes[carrier.index]
						hero := run.Objects[heroOID]
						hero.Level += 6 * level
						run.Objects[heroOID] = hero
					}
				}
			case 5014:
				err = a.rogueLevelTargets(5, 0, 2, 2*level)
			}
		case rogueSale:
			switch id {
			case 2002:
				if self {
					run.Coin += int64(level)
				}
			case 2008:
				if self {
					err = a.rogueRestock(r, 0, level, true)
				}
			case 3002:
				if self {
					run.FreeRefresh += 2 * level
				}
			case 3008:
				if self {
					err = a.rogueLevelTargets(0, -1, 0, 3*level)
				}
			case 3012, 4002, 4008:
				if self {
					err = a.rogueLevelTargets(0, 0, level, value.ExtraLevel)
				}
			case 4005:
				if self {
					for _, entry := range run.Shop {
						relic := run.Objects[entry.Object]
						if entry.Status != 3 && relic.ID/1000 == 4 {
							relic.ExtraLevel += value.ExtraLevel
							run.Objects[entry.Object] = relic
						}
					}
				}
			case 4007:
				if self {
					err = a.rogueLevelTargets(4, 1, 0, 2*level)
				}
			case 4010:
				err = a.rogueLevelTargets(0, 0, level, source.ExtraLevel)
			case 4014:
				err = a.rogueRelicTargets(0, 0, 2*level, 0)
			case 5002:
				if self {
					err = a.rogueRelicTargets(0, 2, 0, level)
				}
			case 5005:
				if self {
					err = a.rogueRestock(r, 5, level, false)
				}
			case 5008:
				if self {
					err = a.rogueLevelTargets(0, 0, 0, 2*level)
				}
			}
		case rogueMerged:
			switch id {
			case 2005:
				if self {
					err = a.rogueEnsureShop(r, 2, len(run.Shop), false)
					if err == nil && onHero {
						err = a.rogueLevelTargets(2, 0, 0, 3*level)
					}
				}
			case 2007:
				if onHero {
					err = a.rogueLevelTargets(2, 0, 3, 3*level)
				}
			case 2012:
				if self && onHero {
					err = a.rogueLevelTargets(0, 0, 0, 8*level)
				}
			case 2014:
				err = a.rogueLevelTargets(2, 0, 3, 2*level)
			}
		}
		if err != nil {
			return err
		}
		a.logRogueTrigger(before, run, carrier.oid, rogueSkillBase+id*100+level)
	}
	if event == rogueMerged && !onHero && source.ID == 2005 {
		return a.rogueEnsureShop(r, 2, len(run.Shop), false)
	}
	return nil
}

func (a *Account) ApplyRogueBattleProgress(levels map[model.HeroID]int, deaths []model.HeroID, totalDeaths int) error {
	run := &a.RogueProgress.Run
	if run.Status != model.RogueBattle {
		return rejected("rogue battle is inactive")
	}
	heroes := map[model.HeroID]model.InstanceID{}
	for _, oid := range run.Heroes {
		heroes[model.HeroID(run.Objects[oid].ID)] = oid
	}
	for id, count := range levels {
		if _, known := heroes[id]; !known || count < 0 || count > 10000 {
			return rejected("invalid rogue battle level reward")
		}
	}
	run.LastDeaths = []model.HeroID{}
	for _, id := range deaths {
		if _, known := heroes[id]; known && !slices.Contains(run.LastDeaths, id) {
			run.LastDeaths = append(run.LastDeaths, id)
		}
	}
	run.DeathCount = max(0, min(totalDeaths, 10000))
	for id, count := range levels {
		oid := heroes[id]
		hero := run.Objects[oid]
		hero.Level = min(10000, hero.Level+count)
		run.Objects[oid] = hero
	}
	return nil
}

type rogueState struct {
	objects map[model.InstanceID]model.RogueObject
	shop    map[int]model.RogueShopEntry
	coin    int64
	free    int
}

func rogueSnapshot(run *model.RogueRun) rogueState {
	return rogueState{objects: maps.Clone(run.Objects), shop: maps.Clone(run.Shop), coin: run.Coin, free: run.FreeRefresh}
}

func (a *Account) logRogueTrigger(before rogueState, run *model.RogueRun, relic model.InstanceID, skill int) {
	targets := []model.InstanceID{}
	for _, oid := range slices.Sorted(maps.Keys(run.Objects)) {
		value := run.Objects[oid]
		if previous, existed := before.objects[oid]; existed && (previous.ID != value.ID || previous.Level != value.Level || previous.ExtraLevel != value.ExtraLevel || previous.Discount != value.Discount) {
			targets = append(targets, oid)
		}
	}
	if len(targets) == 0 && before.coin == run.Coin && before.free == run.FreeRefresh && maps.Equal(before.shop, run.Shop) {
		return
	}
	a.RogueTriggers = append(a.RogueTriggers, model.RogueTrigger{Relic: relic, Skill: skill, Targets: targets})
}
