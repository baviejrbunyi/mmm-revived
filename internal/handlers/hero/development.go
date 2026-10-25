package hero

import (
	"slices"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

func (s service) developHero(p *server.Player, err error) error {
	if err != nil {
		return err
	}
	return s.SaveAndSync(p, "heroComp", "itemComp", "taskComp")
}

func (s service) develop(p *server.Player, step func(*account.Rules) error) error {
	if s.Store.Rules == nil {
		return server.ErrProgressionRules
	}
	return s.developHero(p, step(s.Store.Rules))
}

func (s service) starHero(p *server.Player, hero model.HeroID) error {
	rules := s.Store.Rules
	if rules == nil {
		return server.ErrProgressionRules
	}
	before := *p.Account
	star := p.Account.Heroes[hero].Star
	if err := s.developHero(p, p.Account.StarHero(rules, hero)); err != nil {
		return err
	}
	reward := rules.HeroStarReward(hero)
	if star != 2 || p.Account.Heroes[hero].Star != 3 || reward == 0 {
		return nil
	}
	items := s.ItemRewards(before, p.Account, account.RewardQuantities(rules, reward))
	if len(items) == 0 {
		return nil
	}
	return p.Send("showReward", []model.RewardID{reward}, items, 12)
}

func (s service) equip(p *server.Player, err error, reply string, ids ...model.InstanceID) error {
	if err != nil {
		return err
	}
	if err = s.SaveAndSync(p, "heroComp"); err != nil {
		return err
	}
	return s.notify(p, reply, ids...)
}

func (s service) notify(p *server.Player, reply string, ids ...model.InstanceID) error {
	for i, id := range ids {
		if id == "" || slices.Contains(ids[:i], id) {
			continue
		}
		if err := p.Send(reply, id); err != nil {
			return err
		}
	}
	return nil
}
