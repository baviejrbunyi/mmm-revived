package account

import (
	"maps"
	"slices"

	"mmm-revived/internal/model"
)

func (a *Account) MissionActive(r *Rules, id model.MissionID) bool {
	if id == 0 {
		return false
	}
	if r.Missions[id].Memento {
		return a.CurrentMemento == id
	}
	return a.CurrentStory == id
}

func initializeMemento(mission *model.Mission, rule MissionRule, tiles map[model.MementoChipID]MementoTileRule) {
	mission.Oxygen = rule.Switches[3]
	for id, tile := range tiles {
		if tile.Initial {
			mission.Visited[id] = 0
		}
	}
}

func lokiExpired(rule *MissionRule, mission *model.Mission, chip model.MementoChipID) bool {
	return rule.Switches[2] != 0 && mission.LokiRecord[chip] > 0 && mission.Step-mission.LokiRecord[chip] >= 2
}

func (a *Account) mementoTile(r *Rules, id model.MissionID, chip model.MementoChipID) (model.Mission, MementoTileRule, error) {
	mission, ok := a.Missions[id]
	node := r.Nodes[mission.Node.ID]
	if !ok || !a.MissionActive(r, id) || mission.Fail != 0 || node.Type != 5 {
		return mission, MementoTileRule{}, rejected("Memento is not active")
	}
	rule := r.Missions[id]
	if rule.MaximumSteps > 0 && mission.Step >= rule.MaximumSteps || rule.Switches[3] > 0 && mission.Oxygen <= 0 {
		return mission, MementoTileRule{}, rejected("Memento run has expired")
	}
	tiles := r.MementoMaps[node.Map]
	tile, ok := tiles[chip]
	if !ok {
		return mission, tile, rejected("unknown Memento tile")
	}
	expired := lokiExpired(&rule, &mission, chip)
	if _, visited := mission.Visited[chip]; visited && !expired {
		return mission, tile, rejected("Memento tile is already visited")
	}
	adjacent := slices.ContainsFunc(tile.Next, func(neighbour model.MementoChipID) bool {
		_, visited := mission.Visited[neighbour]
		entry := tiles[neighbour]
		return visited && !lokiExpired(&rule, &mission, neighbour) || entry.Type == model.MementoShield && int64(mission.ShieldDecrease) < entry.Parameter
	})
	if !adjacent {
		return mission, tile, rejected("Memento tile is not reachable")
	}
	if expired {
		tile.Type = model.MementoEmpty
		return mission, tile, nil
	}
	switch tile.Type {
	case model.MementoFire:
		if int64(mission.Step) >= tile.Parameter {
			return mission, tile, rejected("Memento tile is burning")
		}
	case model.MementoShield:
		if int64(mission.ShieldDecrease) < tile.Parameter {
			return mission, tile, rejected("Memento tile is shielded")
		}
	case model.MementoDoor:
		unlocked := false
		for key, entry := range tiles {
			if _, visited := mission.Visited[key]; visited && entry.Type == model.MementoKey && entry.Parameter == tile.Parameter {
				unlocked = true
				break
			}
		}
		if !unlocked {
			return mission, tile, rejected("Memento key is missing")
		}
	}
	if rule.Switches[1] != 0 && (tile.Type == model.MementoBattle || tile.Type == model.MementoBoss || tile.Type == model.MementoDeployment) {
		step := mission.Step
		for key := range mission.Visited {
			if tiles[key].Type == model.MementoMaintainMoon {
				step--
			}
		}
		if (tile.Moon+step)%2 != 0 {
			return mission, tile, rejected("Memento battle is unavailable in this moon phase")
		}
	}
	return mission, tile, nil
}

func (a *Account) MementoBattle(r *Rules, id model.MissionID, chip model.MementoChipID, battle model.BattleID) error {
	_, tile, err := a.mementoTile(r, id, chip)
	if err != nil {
		return err
	}
	if (tile.Type != model.MementoBattle && tile.Type != model.MementoBoss) || tile.Parameter != int64(battle) {
		return rejected("battle does not match Memento tile")
	}
	return nil
}

func (a *Account) visitMemento(r *Rules, id model.MissionID, chip model.MementoChipID, heroes []model.HeroID, battleStars int) error {
	mission, tile, err := a.mementoTile(r, id, chip)
	if err != nil {
		return err
	}
	isBattle := tile.Type == model.MementoBattle || tile.Type == model.MementoBoss
	if isBattle != (battleStars != 0) {
		return rejected("Memento battle result is required")
	}
	if tile.Type == model.MementoDeployment {
		if len(heroes) != int(tile.Parameter) {
			return rejected("wrong Memento deployment size")
		}
		level := r.Missions[mission.ID].DeploymentLevel
		seen := make(map[model.HeroID]bool, len(heroes))
		for _, hero := range heroes {
			entry, owned := a.Heroes[hero]
			if !owned || seen[hero] || entry.Level < level || mission.Heroes[hero] != 0 || tile.Camp != 0 && r.HeroTraits[hero].Camp != tile.Camp {
				return rejected("invalid Memento deployment hero")
			}
			seen[hero] = true
		}
	} else if len(heroes) != 0 {
		return rejected("unexpected Memento deployment heroes")
	}
	rule := r.Missions[id]
	mission.Visited, mission.LokiRecord, mission.Heroes = maps.Clone(mission.Visited), maps.Clone(mission.LokiRecord), maps.Clone(mission.Heroes)
	mission.Step++
	mission.Visited[chip] = mission.Step
	if rule.Switches[2] != 0 {
		mission.LokiRecord[chip] = mission.Step
	}
	if rule.Switches[3] > 0 {
		mission.Oxygen--
	}
	switch tile.Type {
	case model.MementoBox:
		progress := a.Passed[id]
		if progress.BoxRewarded == 0 {
			if err = a.Grant(r, model.RewardID(tile.Parameter)); err != nil {
				return err
			}
			progress.BoxRewarded = 1
			a.Passed[id] = progress
		}
		mission.BoxReward = int(tile.Parameter)
	case model.MementoDeployment:
		for _, hero := range heroes {
			mission.Heroes[hero] = 1
		}
		mission.ShieldDecrease++
	case model.MementoBattle:
		mission.ShieldDecrease++
	case model.MementoPortal:
		for _, target := range tile.Targets {
			mission.Visited[target] = mission.Step
		}
	case model.MementoRecover:
		clear(mission.Heroes)
	case model.MementoFreeStep:
		mission.Step--
	case model.MementoOxygen:
		mission.Oxygen = min(mission.Oxygen+int(tile.Parameter), rule.Switches[3])
	case model.MementoBoss:
		mission.BattleResult = battleStars
		next := r.Nodes[mission.Node.ID].Next
		if next == 0 {
			return rejected("Memento reward node is missing")
		}
		mission.Node = model.MissionNode{ID: next}
		stars := 1
		if len(rule.StarConditions) > 0 {
			condition := rule.StarConditions[0]
			pass := false
			switch condition[0] {
			case 1:
				pass = mission.Step <= condition[1]
			case 2:
				tiles := r.MementoMaps[r.Nodes[rule.FirstNode].Map]
				count := 0
				for key := range mission.Visited {
					if tiles[key].Type == model.MementoDeployment {
						count++
					}
				}
				pass = count >= condition[1]
			}
			if pass {
				stars |= 2
			}
		}
		if battleStars == 7 {
			stars |= 4
		}
		progress := a.Passed[id]
		progress.Stars |= stars
		a.Passed[id] = progress
	}
	a.Missions[id] = mission
	return nil
}

func (a *Account) VisitMemento(r *Rules, id model.MissionID, chip model.MementoChipID, heroes []model.HeroID) error {
	return a.visitMemento(r, id, chip, heroes, 0)
}

func (a *Account) FinishMementoBattle(r *Rules, id model.MissionID, chip model.MementoChipID, stars int, selected []model.HeroID) error {
	if stars < 1 || stars > 7 {
		return rejected("invalid Memento battle stars")
	}
	for _, hero := range selected {
		if _, owned := a.Heroes[hero]; !owned || a.Missions[id].Heroes[hero] != 0 {
			return rejected("Memento hero is unavailable")
		}
	}
	if err := a.visitMemento(r, id, chip, nil, stars); err != nil {
		return err
	}
	mission := a.Missions[id]
	for _, hero := range selected {
		mission.Heroes[hero] = 1
	}
	a.Missions[id] = mission
	return nil
}
