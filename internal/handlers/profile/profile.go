package profile

import (
	"errors"
	"fmt"
	"maps"
	"reflect"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

const (
	renameItem     model.ItemID = 100004
	renameCurrency model.ItemID = 2
	renamePrice    int64        = 200
)

func (s service) edit(p *server.Player, apply func(*account.Account)) (bool, error) {
	before, name := p.Account.Profile, p.Account.TeamName
	before.RedDots = maps.Clone(before.RedDots)
	apply(p.Account)
	if name == p.Account.TeamName && reflect.DeepEqual(before, p.Account.Profile) {
		return false, nil
	}
	return true, s.SaveAndSync(p, "basicComp", "makeupComp", "heroComp", "lobbyComp")
}

func (s service) editProfile(p *server.Player, apply func(*account.Account)) error {
	changed, err := s.edit(p, apply)
	if err != nil || changed {
		return err
	}
	return server.Acknowledge(p)
}

func requireHeroes(a *account.Account, heroes []model.HeroID) error {
	for _, id := range heroes {
		if _, ok := a.Heroes[id]; id != 0 && !ok {
			return fmt.Errorf("%w: hero is not owned", account.ErrProgression)
		}
	}
	return nil
}

func (s service) cosmetic(kind int, field func(*model.ProfileCustomization) *model.ItemID) rpc.Endpoint[*server.Player] {
	return rpc.Bind(DecodeCosmetic, func(p *server.Player, item model.ItemID) error {
		if !(kind == 1035 && item == 0) && !p.Account.OwnsCosmetic(s.Store.Catalog, kind, item) {
			return fmt.Errorf("%w: cosmetic is not owned", account.ErrProgression)
		}
		return s.editProfile(p, func(a *account.Account) { *field(&a.Profile) = item })
	})
}

func (s service) changeTeamName(p *server.Player, name string) error {
	code := teamNameCode(name)
	a := p.Account
	if code == 0 && name != a.TeamName && a.TeamName != "" && a.TeamName != a.Name {
		cost := map[model.ItemID]int64{renameItem: 1}
		if a.Items[renameItem] < 1 {
			cost = map[model.ItemID]int64{renameCurrency: renamePrice}
		}
		if err := a.Spend(cost); errors.Is(err, account.ErrProgression) {
			code = codeItemsMissing
		} else if err != nil {
			return err
		}
	}
	if code != 0 {
		return p.Send("changeTeamNameResult", code, name)
	}
	if _, err := s.edit(p, func(a *account.Account) { a.TeamName = name }); err != nil {
		return err
	}
	if err := s.Sync(p, "itemComp"); err != nil {
		return err
	}
	return p.Send("changeTeamNameResult", 0, name)
}

func (s service) changeSignature(p *server.Player, signature string) error {
	if code := signatureCode(signature); code != 0 {
		return p.Send("onSignatureChange", code)
	}
	if _, err := s.edit(p, func(a *account.Account) { a.Profile.Signature = signature }); err != nil {
		return err
	}
	return p.Send("onSignatureChange", 0)
}

func validSlot(slot int) bool { return slot >= 1 && slot <= model.LineupSlots }

func validLineupType(kind int) bool { return kind >= 1 && kind <= model.LineupTypes }

func (s service) editLineup(p *server.Player, slot int, apply func(*model.Lineup)) error {
	lineup, ok := p.Account.Lineups[slot]
	if !ok || lineup.Type == 0 {
		lineup.Type = model.DefaultLineupType
	}
	apply(&lineup)
	p.Account.Lineups[slot] = lineup
	return s.SaveAndSync(p, "lineupComp")
}

func (s service) makeLineup(p *server.Player, r Lineup) error {
	code := 0
	if !validSlot(r.Slot) || !validLineupType(r.Kind) || len(r.Heroes) > lineupHeroes || requireHeroes(p.Account, r.Heroes) != nil {
		code = codeCommon
	} else if err := s.editLineup(p, r.Slot, func(lineup *model.Lineup) { lineup.Heroes, lineup.Type = r.Heroes, r.Kind }); err != nil {
		return err
	}
	return p.Send("sendMakeLineupResult", code, r.Slot, r.Kind, r.Heroes)
}

func (s service) changeLineupName(p *server.Player, r Lineup) error {
	code := lineupNameCode(r.Name)
	if !validSlot(r.Slot) {
		code = codeCommon
	}
	if code == 0 {
		if err := s.editLineup(p, r.Slot, func(lineup *model.Lineup) { lineup.Name = r.Name }); err != nil {
			return err
		}
	}
	return p.Send("changeLineupNameResult", code, r.Slot, r.Name)
}

func (s service) changeLineupType(p *server.Player, r Lineup) error {
	code := 0
	if !validSlot(r.Slot) || !validLineupType(r.Kind) {
		code = codeCommon
	} else if err := s.editLineup(p, r.Slot, func(lineup *model.Lineup) { lineup.Type = r.Kind }); err != nil {
		return err
	}
	return p.Send("changeLineupTypeResult", code, r.Slot, r.Kind)
}

func (s service) deleteLineup(p *server.Player, slot int) error {
	code := 0
	if !validSlot(slot) {
		code = codeCommon
	} else if err := s.editLineup(p, slot, func(lineup *model.Lineup) { *lineup = model.Lineup{Type: model.DefaultLineupType} }); err != nil {
		return err
	}
	return p.Send("deleteLineupResult", code, slot)
}
