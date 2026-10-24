package battle

import (
	"maps"
	"slices"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

type TrialInfo struct {
	Heroes     model.Empty `wire:"heroAttrInfo"`
	Artifacts  model.Empty `wire:"artifactInfo"`
	Highlights model.Empty `wire:"highlightInfo"`
}

func GearInfo(a *account.Account) (map[model.HeroID]map[int]model.Artifact, map[model.HeroID]model.Highlight) {
	artifacts := make(map[model.HeroID]map[int]model.Artifact, len(a.Heroes))
	highlights := make(map[model.HeroID]model.Highlight, len(a.Heroes))
	for id, hero := range a.Heroes {
		equipped := make(map[int]model.Artifact, len(hero.Artifacts))
		for slot, instance := range hero.Artifacts {
			if artifact, ok := a.Artifacts[instance]; ok && instance != "" {
				equipped[slot] = artifact
			}
		}
		artifacts[id] = equipped
		if highlight, ok := a.Highlights[hero.Highlights[1]]; ok && hero.Highlights[1] != "" {
			highlights[id] = highlight
		}
	}
	return artifacts, highlights
}

type templateHero struct {
	ID       model.HeroID `wire:"id"`
	Level    int          `wire:"lv"`
	Break    int          `wire:"breakLv"`
	Star     int          `wire:"star"`
	AwakeIDs []int        `wire:"awakeIds"`
	Talents  map[int]int  `wire:"talents"`
}

func battleGear(a *account.Account, rule account.BattleRule) (any, map[model.HeroID]map[int]model.Artifact, map[model.HeroID]model.Highlight) {
	artifacts, highlights := GearInfo(a)
	var heroes map[model.HeroID]any
	for id, template := range rule.TemplateHeroes {
		if heroes == nil {
			heroes = make(map[model.HeroID]any, len(a.Heroes)+len(rule.TemplateHeroes))
			for owned, hero := range a.Heroes {
				heroes[owned] = hero
			}
		}
		state := template.At(a.Level)
		talents := maps.Clone(state.Talents)
		if talents == nil {
			talents = map[int]int{}
		}
		heroes[id] = templateHero{ID: id, Level: state.Level, Break: state.Break, Star: state.Star, AwakeIDs: append([]int{}, state.Awake...), Talents: talents}
		equipped := make(map[int]model.Artifact, len(state.Artifacts))
		for slot, artifact := range state.Artifacts {
			equipped[slot+1] = model.Artifact{ID: artifact.ID, Level: artifact.Level, Template: artifact.Template, Install: id, Sub: maps.Clone(artifact.Sub)}
		}
		artifacts[id] = equipped
		delete(highlights, id)
		if state.Highlight != nil {
			highlights[id] = model.Highlight{ID: state.Highlight.ID, Level: state.Highlight.Level, Awake: state.Highlight.Awake, Install: id}
		}
	}
	if heroes == nil {
		return a.Heroes, artifacts, highlights
	}
	return heroes, artifacts, highlights
}

func fixedHeroes(rule account.BattleRule) []model.HeroID {
	return slices.Sorted(maps.Keys(rule.TemplateHeroes))
}

func templateTrialHero(templates []model.HeroID, hero model.HeroID) bool {
	for _, template := range templates {
		if template/100 == hero {
			return true
		}
	}
	return false
}

type battleInfo struct {
	TeamLevel      int              `wire:"teamLv"`
	SeasonScience  []int            `wire:"seasonScienceNodeIds"`
	HyperFlowBuffs []int            `wire:"hyper_flow_buffs"`
	Bonds          map[string]int64 `wire:"bondAttr"`
	SkinCollection int              `wire:"skinCollectionLv"`
	Activity       model.ActivityID `wire:"activityId,omitempty"`
	Environments   []int            `wire:"environmentIds,omitempty"`
}

var mayhemEnvironments = [...]int{1001, 1002, 1003, 1004, 1005, 1006, 1007}

func mayhemEnvironment(parameter int) int {
	return mayhemEnvironments[(max(parameter, 1)-1)%len(mayhemEnvironments)]
}

func newBattleInfo(s *server.Service, a *account.Account) battleInfo {
	return battleInfo{TeamLevel: a.Level, SeasonScience: slices.Clone(a.Events.Season.Nodes), HyperFlowBuffs: []int{}, Bonds: a.BondAttributes(s.Store.Rules), SkinCollection: a.SkinCollectionLevel(s.Store.Rules)}
}
