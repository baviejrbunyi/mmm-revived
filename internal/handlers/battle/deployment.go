package battle

import (
	"fmt"
	"slices"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

func (s service) lockHeroes(p *server.Player, r Deployment) error {
	if p.State != server.Deployment && p.State != server.Combat {
		return fmt.Errorf("%w: no battle awaiting deployment", account.ErrProgression)
	}
	var rule account.BattleRule
	if s.Store.Rules != nil {
		rule = s.Store.Rules.BattleSettings[p.BattleID]
	}
	support := p.Account.Network.Support
	selected := make([]model.HeroID, 0, len(r.Heroes))
	for _, entry := range r.Heroes {
		_, owned := p.Account.Heroes[entry.Hero]
		owned = owned || support.Owner != 0 && support.Hero == entry.Hero
		_, fixed := rule.TemplateHeroes[entry.Hero]
		if !owned && p.Limited[entry.Hero].ID == 0 && !fixed && (entry.Trial != 1 || !templateTrialHero(rule.TrialHeroes, entry.Hero)) {
			return fmt.Errorf("%w: deployment hero is not owned", account.ErrProgression)
		}
		if p.Chip != nil && p.Account.Missions[p.Mission].Heroes[entry.Hero] != 0 {
			return fmt.Errorf("%w: Memento hero is unavailable", account.ErrProgression)
		}
		selected = append(selected, entry.Hero)
	}
	if s.Store.Rules != nil {
		if p.ActivityBattle > 0 {
			if err := p.Account.ValidateJourneyActivityHeroes(s.Store.Rules, p.ActivityBattle, p.ActivityNode, p.BattleID, selected); err != nil {
				return err
			}
		}
		if p.CampActivity > 0 {
			if err := p.Account.ValidateRetailCampHeroes(s.Store.Rules, p.CampActivity, p.CampNode, selected); err != nil {
				return err
			}
		}
		limit := rule.Heroes
		if p.HeroLimit > 0 {
			limit = p.HeroLimit
		}
		if len(selected) == 0 && len(rule.FixedHeroes) == 0 || limit > 0 && len(selected) > limit {
			return fmt.Errorf("%w: invalid deployment size", account.ErrProgression)
		}
		if len(r.Cards) > rule.CardCount {
			return fmt.Errorf("%w: too many common cards", account.ErrProgression)
		}
		for _, card := range r.Cards {
			if !slices.Contains(rule.Cards, card) {
				return fmt.Errorf("%w: common card is not available", account.ErrProgression)
			}
		}
	}
	for _, lock := range s.modes.Lock {
		if err := lock(s.Service, p, selected); err != nil {
			return err
		}
	}
	p.Deployment = slices.Clone(r.Heroes)
	p.SelectedHeroes, p.State = selected, server.Combat
	heroes, artifacts, highlights := battleGear(p.Account, rule)
	return p.Send("updateHero", heroes, artifacts, highlights, r.Cards)
}
