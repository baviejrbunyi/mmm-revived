package account

import "mmm-revived/internal/model"

type RogueBattleData struct {
	Limited map[model.HeroID]model.RogueLimitedHero
	Extra   model.RogueBattleExtra
}

func (a *Account) RogueBattleData(r *Rules, id model.BattleID) (RogueBattleData, error) {
	result := RogueBattleData{Limited: map[model.HeroID]model.RogueLimitedHero{}, Extra: model.RogueBattleExtra{TeamLevel: a.Level, Mana: -1, EnemyTeam: [][]int64{}, RogueHeroes: map[model.HeroID]model.RogueEnhancement{}, RogueBuffs: []int{}, InheritedHeroes: map[model.HeroID]model.RogueHeroStatus{}, EnemyBuffs: []int{}}}
	if err := a.ValidateRogueBattle(r, id); err != nil {
		return result, err
	}
	switch {
	case r.RogueRunBattle(id):
		run := &a.RogueProgress.Run
		rule := r.Rogue.Runs[run.ID]
		for _, oid := range run.Heroes {
			hero := run.Objects[oid]
			entry := model.RogueLimitedHero{ID: model.HeroID(hero.ID), Level: hero.Level, Index: hero.Index, Star: hero.Star, Talents: hero.Talents, Awake: hero.Awake, Highlights: hero.Highlights}
			enhancement := 0
			if relic, exists := run.Objects[hero.Relic]; exists {
				entry.Relic, entry.RelicLevel = relic.ID, relic.Level
				if bonuses := r.Rogue.Relics[relic.ID].HeroLevels; relic.Level > 0 && relic.Level <= len(bonuses) {
					enhancement = bonuses[relic.Level-1] + relic.ExtraLevel
					entry.Level += enhancement
				}
			}
			entry.Level = min(r.Rogue.MaxHeroLevel, entry.Level)
			result.Limited[entry.ID] = entry
			result.Extra.RogueHeroes[entry.ID] = model.RogueEnhancement{Level: enhancement}
		}
		result.Extra.EnemyTeam = make([][]int64, 0, len(run.Heroes))
		used := make(map[model.HeroID]bool, len(run.Heroes))
		for index := range len(run.Heroes) {
			pool := rule.HeroPool[(run.Index+index)%len(rule.HeroPool)]
			chosen := pool[0]
			for _, hero := range pool {
				if !used[hero] {
					chosen = hero
					break
				}
			}
			used[chosen] = true
			level := max(1, rule.Levels[min(run.Index, len(rule.Levels)-1)])
			result.Extra.EnemyTeam = append(result.Extra.EnemyTeam, []int64{int64(chosen), int64(level), 0, 1, 0})
		}
	case r.RoguePVEBattle(id):
		v := &a.RogueProgress.PVE
		rule := r.Rogue.PVE[v.ID]
		result.Extra.Mana = v.Mana
		if v.Round < len(rule.Levels) {
			result.Extra.EnemyLevel = rule.Levels[v.Round]
		}
		result.Extra.RogueBuffs = v.Battle.Buffs
		if v.Round < len(rule.EnemyBuffs) {
			result.Extra.EnemyBuffs = rule.EnemyBuffs[v.Round]
		}
		for _, id := range v.Battle.Heroes {
			hero := a.Heroes[id]
			result.Limited[id] = model.RogueLimitedHero{ID: id, Level: hero.Level, Star: hero.Star, BreakLevel: hero.Break, Talents: hero.Talents, Awake: hero.AwakeIDs, Artifact: hero.Artifacts[0], Highlights: hero.Highlights}
			if v.Round > 0 {
				result.Extra.InheritedHeroes[id] = v.Battle.HeroStatus[id]
			}
		}
	}
	return result, nil
}

func (a *Account) BeginRogueBattle(r *Rules, id model.BattleID) error {
	if err := a.ValidateRogueBattle(r, id); err != nil {
		return err
	}
	if r.RoguePVEBattle(id) && a.RogueProgress.PVE.Status == model.RoguePVEPrepare {
		a.RogueProgress.PVE.Status = model.RoguePVEBattle
	}
	return nil
}
