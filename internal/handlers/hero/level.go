package hero

import "mmm-revived/internal/server"

func (s service) upgradeHero(p *server.Player, r HeroLevel) error {
	if s.Store.Rules == nil {
		return server.ErrProgressionRules
	}
	if err := p.Account.UpgradeHero(s.Store.Rules, r.Hero, r.Levels); err != nil {
		return err
	}
	if err := s.SaveAndSync(p, "heroComp", "itemComp", "taskComp"); err != nil {
		return err
	}
	return p.Send("onUpHeroLv", 0, r.Hero)
}

func (s service) upgradeHighlight(p *server.Player, r InstanceLevel) error {
	if s.Store.Rules == nil {
		return server.ErrProgressionRules
	}
	if err := p.Account.UpgradeHighlight(s.Store.Rules, r.Instance, r.Levels); err != nil {
		return err
	}
	if err := s.SaveAndSync(p, "heroComp", "itemComp"); err != nil {
		return err
	}
	return s.notify(p, "onHighlightChange", r.Instance)
}

func (s service) upgradeArtifact(p *server.Player, r InstanceLevel) error {
	if s.Store.Rules == nil {
		return server.ErrProgressionRules
	}
	previous := p.Account.Artifacts[r.Instance]
	if err := p.Account.UpgradeArtifact(s.Store.Rules, r.Instance, r.Levels); err != nil {
		return err
	}
	if err := s.SaveAndSync(p, "heroComp", "itemComp"); err != nil {
		return err
	}
	if err := p.Send("onUpArtifact", r.Instance, previous.Level, p.Account.Artifacts[r.Instance].Level, previous.Sub); err != nil {
		return err
	}
	return s.notify(p, "onArtifactChange", r.Instance)
}
