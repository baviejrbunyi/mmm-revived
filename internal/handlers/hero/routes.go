package hero

import (
	"fmt"
	"maps"
	"slices"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("starHero", rpc.Bind(DecodeHero, s.starHero))
	r.Handle("talentHero", rpc.Bind(DecodeHeroValue, func(p *server.Player, r HeroValue) error {
		return s.develop(p, func(rules *account.Rules) error { return p.Account.TalentHero(rules, r.Hero, r.Value) })
	}))
	r.Handle("awakeHero", rpc.Bind(DecodeHeroValue, func(p *server.Player, r HeroValue) error {
		return s.develop(p, func(rules *account.Rules) error { return p.Account.AwakeHero(rules, r.Hero, r.Value) })
	}))
	r.Handle("setHeroSkin", rpc.Bind(DecodeHeroValue, func(p *server.Player, r HeroValue) error {
		return s.develop(p, func(rules *account.Rules) error { return p.Account.SetSkin(rules, r.Hero, model.SkinID(r.Value)) })
	}))
	r.Handle("markHeroLike", rpc.Bind(DecodeHeroValue, func(p *server.Player, r HeroValue) error {
		hero, ok := p.Account.Heroes[r.Hero]
		if !ok || r.Value < 0 || r.Value > 1 {
			return fmt.Errorf("%w: invalid favorite", account.ErrProgression)
		}
		hero.Liked = r.Value
		p.Account.Heroes[r.Hero] = hero
		return s.equip(p, nil, "")
	}))
	r.Handle("installArtifact", rpc.Bind(DecodeEquipSlot, func(p *server.Player, r Equip) error {
		if s.Store.Rules == nil {
			return server.ErrProgressionRules
		}
		previous := p.Account.Heroes[r.Hero].Artifacts[r.Slot]
		return s.equip(p, p.Account.InstallArtifact(s.Store.Rules, r.Hero, r.Instance, r.Slot), "onArtifactChange", r.Instance, previous)
	}))
	r.Handle("uninstallArtifact", rpc.Bind(DecodeEquip, func(p *server.Player, r Equip) error {
		return s.equip(p, p.Account.UninstallArtifact(r.Hero, r.Instance), "onArtifactChange", r.Instance)
	}))
	r.Handle("installHighlight", rpc.Bind(DecodeEquip, func(p *server.Player, r Equip) error {
		previous := p.Account.Heroes[r.Hero].Highlight
		return s.equip(p, p.Account.InstallHighlight(r.Hero, r.Instance), "onHighlightChange", r.Instance, previous)
	}))
	r.Handle("uninstallHighlight", rpc.Bind(DecodeUnequipHighlight, func(p *server.Player, r Equip) error {
		installed := p.Account.Heroes[r.Hero].Highlights
		removed := []model.InstanceID{installed[r.Slot]}
		if r.Slot == 0 {
			removed = slices.Sorted(maps.Values(installed))
		}
		return s.equip(p, p.Account.UninstallHighlight(r.Hero, r.Slot), "onHighlightChange", removed...)
	}))
	r.Handle("artifactSetLock", rpc.Bind(DecodeLock, func(p *server.Player, r Lock) error {
		entry, ok := p.Account.Artifacts[r.Instance]
		if !ok {
			return fmt.Errorf("%w: artifact is not owned", account.ErrProgression)
		}
		entry.Locked = r.State
		p.Account.Artifacts[r.Instance] = entry
		return s.equip(p, nil, "onArtifactChange", r.Instance)
	}))
	r.Handle("highlightSetLock", rpc.Bind(DecodeLock, func(p *server.Player, r Lock) error {
		entry, ok := p.Account.Highlights[r.Instance]
		if !ok {
			return fmt.Errorf("%w: highlight is not owned", account.ErrProgression)
		}
		entry.Locked = r.State
		p.Account.Highlights[r.Instance] = entry
		if err := s.equip(p, nil, ""); err != nil {
			return err
		}
		if err := p.Send("onHighlightSetLock", r.Instance, int(r.State)); err != nil {
			return err
		}
		return s.notify(p, "onHighlightChange", r.Instance)
	}))
	r.Handle("upHeroLv", rpc.Bind(DecodeHeroLevel, s.upgradeHero))
	r.Handle("upHighlight", rpc.Bind(DecodeInstanceLevel, s.upgradeHighlight))
	r.Handle("upArtifact", rpc.Bind(DecodeInstanceLevel, s.upgradeArtifact))
}
