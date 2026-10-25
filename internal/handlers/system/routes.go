package system

import (
	"errors"
	"math"
	"unicode/utf8"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	r.Handle("setHeroShow", rpc.Bind(DecodeLobbyHero, func(p *server.Player, r LobbyHero) error {
		err := s.system(p, func(a *account.Account, _ int64) error {
			return a.SetLobbyHero(s.Store.Rules, r.Hero, r.Skin, r.Dialogs)
		})
		if err != nil {
			return err
		}
		if err = s.Sync(p, "lobbyComp"); err != nil {
			return err
		}
		return p.Send("onSetHeroShow", true, r.Kind)
	}))
	r.Handle("triggerHeroDialog", rpc.Bind(protocol.DecodeNone, func(p *server.Player, _ struct{}) error {
		if p.State != server.Lobby {
			return server.Acknowledge(p)
		}
		changed := false
		err := s.system(p, func(a *account.Account, now int64) error {
			changed = a.TriggerHeroDialog(s.Store.Rules, now)
			return nil
		})
		return s.systemSync(p, err, changed, "taskComp")
	}))
	r.Handle("lobbyMirageInteract", rpc.Bind(DecodeMirage, func(p *server.Player, activity model.ActivityID) error {
		changed := false
		err := s.system(p, func(a *account.Account, now int64) error {
			var err error
			changed, err = a.MirageInteract(s.Store.Rules, activity, now)
			return err
		})
		return s.systemSync(p, err, changed, "activityComp")
	}))
	r.Handle("setChildrenProtectPwd", rpc.Bind(DecodePIN, func(p *server.Player, pin string) error {
		return s.protection(p, func(a *account.Account, now int64) error { return a.SetProtectionPIN(s.Store.Rules, pin, now) })
	}))
	r.Handle("unsetChildrenProtectPwd", rpc.Bind(DecodePIN, func(p *server.Player, pin string) error {
		return s.protection(p, func(a *account.Account, now int64) error { return a.UnsetProtectionPIN(pin, now) })
	}))
	r.Handle("forgetChildrenProtectPwd", rpc.Bind(protocol.DecodeNone, func(p *server.Player, _ struct{}) error {
		return s.protection(p, func(a *account.Account, now int64) error {
			a.ForgetProtectionPIN(s.Store.Rules, now)
			return nil
		})
	}))
	r.Handle("clearJingLing", rpc.Bind(protocol.DecodeNone, func(p *server.Player, _ struct{}) error {
		if p.Account.System.QuestionData == "" {
			return server.Acknowledge(p)
		}
		err := s.system(p, func(a *account.Account, _ int64) error {
			a.System.QuestionData = ""
			return nil
		})
		if err != nil {
			return err
		}
		return p.Update([]string{"jingLingComp"}, p.Account.Snapshot(s.Store.Catalog, s.Store.Rules).Questions)
	}))
	r.Handle("changeGameSetting", rpc.Bind(DecodeSetting, func(p *server.Player, setting Setting) error {
		if setting.Missing {
			return nil
		}
		err := s.system(p, func(a *account.Account, _ int64) error { return a.ChangeSystemSetting(setting.Key, setting.Value) })
		if errors.Is(err, account.ErrProgression) {
			p.Session.Logger.Debug("game setting ignored", "key", setting.Key, "error", err)
			return nil
		}
		return err
	}))
	r.Handle("bindPushToken", rpc.Bind(DecodePushToken, func(p *server.Player, token string) error {
		return s.system(p, func(a *account.Account, _ int64) error {
			if !utf8.ValidString(token) {
				return errInvalidPushToken
			}
			a.System.PushToken = token
			return nil
		})
	}))
	r.Handle("bindPushTags", rpc.Bind(DecodePushTags, func(p *server.Player, tags []string) error {
		err := s.system(p, func(a *account.Account, _ int64) error {
			a.System.PushTags = tags
			return nil
		})
		if err != nil {
			return err
		}
		return server.Acknowledge(p)
	}))
	r.Handle("getGiftPackByClient", rpc.Bind(DecodeGiftOffer, func(p *server.Player, reference int) error {
		err := s.system(p, func(a *account.Account, now int64) error { return a.OfferLocalGift(s.Store.Rules, reference, now) })
		if errors.Is(err, account.ErrProgression) {
			p.Session.Logger.Debug("client gift offer ignored", "gift", reference, "error", err)
			return nil
		}
		if err != nil {
			return err
		}
		return p.Update([]string{"popupPackComp"}, p.Account.Snapshot(s.Store.Catalog, s.Store.Rules).PopupPacks)
	}))
	r.Handle("sweepLastShipMark", rpc.Bind(DecodeFlag, func(p *server.Player, _ bool) error {
		if p.Account.System.LastReceipt == nil {
			return server.Acknowledge(p)
		}
		return s.receipts(p, func(a *account.Account) { a.System.LastReceipt = nil })
	}))
	r.Handle("markReturnPay", rpc.Bind(DecodeFlag, func(p *server.Player, enabled bool) error {
		if p.Account.System.ReturnedPayment == enabled {
			return server.Acknowledge(p)
		}
		return s.receipts(p, func(a *account.Account) { a.System.ReturnedPayment = enabled })
	}))
	r.Handle("recordNewPlayerLeaveReason", rpc.Bind(DecodeLeaveReason, func(p *server.Player, r LeaveReason) error {
		err := s.system(p, func(a *account.Account, now int64) error {
			a.System.LastLeave = &model.LocalLeaveReason{Stage: r.Stage, Tags: r.Tags, Content: r.Content, Created: now}
			return nil
		})
		if err != nil {
			return err
		}
		return p.Send("onRecordNewPlayerLeaveReason")
	}))
	r.Handle("bugReport", rpc.Bind(DecodeBugReport, func(p *server.Player, r BugReport) error {
		err := s.system(p, func(a *account.Account, now int64) error {
			diagnostics := a.System.Diagnostics
			record, exists := diagnostics[r.Battle]
			if !exists && len(diagnostics) >= 128 {
				oldest := model.BattleID(0)
				for battle, value := range diagnostics {
					if oldest == 0 || value.LastTime < diagnostics[oldest].LastTime {
						oldest = battle
					}
				}
				delete(diagnostics, oldest)
			}
			record.Count = min(math.MaxInt32, record.Count+1)
			record.Version, record.LastTime, record.RecordDigest = r.Version, now, r.Digest
			diagnostics[r.Battle] = record
			return nil
		})
		if err != nil {
			return err
		}
		return server.Acknowledge(p)
	}))
	r.Handle("passThroughCheck", rpc.Bind(DecodePurchase, s.purchase))
}
