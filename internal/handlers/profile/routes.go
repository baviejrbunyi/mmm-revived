package profile

import (
	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("changeTeamName", rpc.Bind(DecodeText, s.changeTeamName))
	r.Handle("changeSignature", rpc.Bind(DecodeText, s.changeSignature))
	r.Handle("recordRedDot", rpc.Bind(DecodeRedDot, func(p *server.Player, dot RedDot) error {
		return s.editProfile(p, func(a *account.Account) { a.Profile.RedDots[dot.Key] = dot.Value })
	}))
	r.Handle("setShowHero", rpc.Bind(DecodeShownHeroes, func(p *server.Player, heroes []model.HeroID) error {
		if err := requireHeroes(p.Account, heroes); err != nil {
			return err
		}
		if _, err := s.edit(p, func(a *account.Account) { a.Profile.ShownHeroes = heroes }); err != nil {
			return err
		}
		return p.Send("onSetShowHero")
	}))
	r.Handle("clientSetSupportHero", rpc.Bind(DecodeShownHeroes, func(p *server.Player, heroes []model.HeroID) error {
		if err := requireHeroes(p.Account, heroes); err != nil {
			return err
		}
		return s.editProfile(p, func(a *account.Account) { a.Profile.SupportHeroes = heroes })
	}))
	r.Handle("setShowTags", rpc.Bind(DecodeTags, func(p *server.Player, tags model.HonorTags) error {
		return s.editProfile(p, func(a *account.Account) { a.Profile.Tags = tags })
	}))
	r.Handle("changeHead", s.cosmetic(1022, func(c *model.ProfileCustomization) *model.ItemID { return &c.Head }))
	r.Handle("changeHeadFrame", s.cosmetic(1023, func(c *model.ProfileCustomization) *model.ItemID { return &c.Frame }))
	r.Handle("changeBackground", s.cosmetic(1045, func(c *model.ProfileCustomization) *model.ItemID { return &c.Background }))
	r.Handle("changeAchieveBadge", s.cosmetic(1035, func(c *model.ProfileCustomization) *model.ItemID { return &c.Badge }))
	r.Handle("makeLineup", rpc.Bind(DecodeLineup, s.makeLineup))
	r.Handle("changeLineupName", rpc.Bind(DecodeLineupName, s.changeLineupName))
	r.Handle("changeLineupType", rpc.Bind(DecodeLineupType, s.changeLineupType))
	r.Handle("deleteLineup", rpc.Bind(DecodeLineupSlot, s.deleteLineup))
	r.Handle("clientReqNameCard", rpc.Bind(DecodeNameCards, s.nameCards))
}
