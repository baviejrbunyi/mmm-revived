package rogue

import (
	"errors"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	none := func(name string, action rogueAction) {
		r.Handle(name, rpc.Bind(protocol.DecodeNone, func(p *server.Player, _ struct{}) error { return s.rogue(p, action) }))
	}
	index := func(name string, action func(a *account.Account, rules *account.Rules, now int64, replies *server.Outbox, index int) (map[model.ItemID]int64, error)) {
		r.Handle(name, rpc.Bind(DecodeIndex, func(p *server.Player, i int) error {
			return s.rogue(p, func(a *account.Account, rules *account.Rules, now int64, replies *server.Outbox) (map[model.ItemID]int64, error) {
				return action(a, rules, now, replies, i)
			})
		}))
	}
	pair := func(name string, action func(a *account.Account, rules *account.Rules, replies *server.Outbox, pair Pair) (map[model.ItemID]int64, error)) {
		r.Handle(name, rpc.Bind(DecodePair, func(p *server.Player, v Pair) error {
			return s.rogue(p, func(a *account.Account, rules *account.Rules, _ int64, replies *server.Outbox) (map[model.ItemID]int64, error) {
				return action(a, rules, replies, v)
			})
		}))
	}
	none("initRogue", func(a *account.Account, rules *account.Rules, now int64, _ *server.Outbox) (map[model.ItemID]int64, error) {
		return nil, a.InitRogue(rules, now)
	})
	none("startRogue", func(a *account.Account, rules *account.Rules, _ int64, _ *server.Outbox) (map[model.ItemID]int64, error) {
		return nil, a.StartRogue(rules)
	})
	index("selectHero", func(a *account.Account, rules *account.Rules, _ int64, _ *server.Outbox, i int) (map[model.ItemID]int64, error) {
		return nil, a.SelectRogueHero(rules, i)
	})
	pair("switchHero", func(a *account.Account, _ *account.Rules, replies *server.Outbox, v Pair) (map[model.ItemID]int64, error) {
		replies.Add("onSwitchHero", v.First, v.Second)
		return nil, a.SwitchRogueHeroes(v.First, v.Second)
	})
	pair("buyRogueRelic", func(a *account.Account, rules *account.Rules, replies *server.Outbox, v Pair) (map[model.ItemID]int64, error) {
		up, err := a.BuyRogueRelic(rules, v.First, v.Second)
		replies.Add("onBuyRelic", v.First, v.Second, up)
		return nil, err
	})
	index("saleRogueRelic", func(a *account.Account, rules *account.Rules, _ int64, replies *server.Outbox, i int) (map[model.ItemID]int64, error) {
		replies.Add("onSaleRelic", i)
		return nil, a.SaleRogueRelic(rules, i)
	})
	pair("mergeShopRelic", func(a *account.Account, rules *account.Rules, replies *server.Outbox, v Pair) (map[model.ItemID]int64, error) {
		replies.Add("onMergeShopRelic", v.First, v.Second)
		return nil, a.MergeRogueShop(rules, v.First, v.Second)
	})
	r.Handle("openRogueChestBox", rpc.Bind(DecodeChestOpen, func(p *server.Player, v Pair) error {
		return s.rogue(p, func(a *account.Account, rules *account.Rules, _ int64, _ *server.Outbox) (map[model.ItemID]int64, error) {
			return a.OpenRogueChest(rules, v.First, v.Second)
		})
	}))
	none("refreshRogueShop", func(a *account.Account, rules *account.Rules, _ int64, _ *server.Outbox) (map[model.ItemID]int64, error) {
		return nil, a.RefreshRogueShop(rules)
	})
	none("finishRogueShopping", func(a *account.Account, rules *account.Rules, _ int64, replies *server.Outbox) (map[model.ItemID]int64, error) {
		replies.Add("onFinishShopping")
		return nil, a.FinishRogueShopping(rules)
	})
	pair("switchRogueHeroRelic", func(a *account.Account, rules *account.Rules, replies *server.Outbox, v Pair) (map[model.ItemID]int64, error) {
		up, err := a.SwitchRogueRelics(rules, v.First, v.Second)
		replies.Add("onSwitchHeroRelic", v.First, v.Second, up)
		return nil, err
	})
	index("setRogueRelicLock", func(a *account.Account, _ *account.Rules, _ int64, replies *server.Outbox, i int) (map[model.ItemID]int64, error) {
		state, err := a.LockRogueRelic(i)
		replies.Add("onRelicShopStChange", i, state)
		return nil, err
	})
	none("getRogueReward", func(a *account.Account, rules *account.Rules, now int64, _ *server.Outbox) (map[model.ItemID]int64, error) {
		return a.ClaimRogue(rules, false, now)
	})
	none("getRogueWeekScoreRwd", func(a *account.Account, rules *account.Rules, now int64, _ *server.Outbox) (map[model.ItemID]int64, error) {
		return a.ClaimRogueWeekly(rules, false, now)
	})
	none("settleRogue", func(a *account.Account, rules *account.Rules, _ int64, _ *server.Outbox) (map[model.ItemID]int64, error) {
		return nil, a.SettleRogue(rules)
	})
	index("tryRoguePve", func(a *account.Account, rules *account.Rules, now int64, _ *server.Outbox, i int) (map[model.ItemID]int64, error) {
		return nil, a.TryRoguePVE(rules, i, now)
	})
	r.Handle("roguePveChooseHero", rpc.Bind(DecodeRogueTeam, func(p *server.Player, heroes []model.HeroID) error {
		return s.rogue(p, func(a *account.Account, rules *account.Rules, _ int64, replies *server.Outbox) (map[model.ItemID]int64, error) {
			return nil, advanceRogueEvent(a, rules, replies, a.ChooseRoguePVEHeroes(rules, heroes))
		})
	}))
	none("doRoguePveEvent", executeRogueEvent(0))
	r.Handle("doSelectOneToExecute", rpc.Bind(DecodeIndex, func(p *server.Player, i int) error {
		err := s.rogue(p, executeRogueEvent(i))
		if err != nil && errors.Is(err, account.ErrProgression) {
			if sent := p.Send("RogueEventGainToken", 0); sent != nil {
				return sent
			}
		}
		return err
	}))
	index("doRogueSelectBuff", func(a *account.Account, rules *account.Rules, _ int64, replies *server.Outbox, i int) (map[model.ItemID]int64, error) {
		return nil, advanceRogueEvent(a, rules, replies, a.SelectRogueBuff(rules, i))
	})
	r.Handle("doBuyRogueBuff", rpc.Bind(DecodeRogueBuffs, func(p *server.Player, buffs []int) error {
		return s.rogue(p, func(a *account.Account, rules *account.Rules, _ int64, _ *server.Outbox) (map[model.ItemID]int64, error) {
			return nil, a.BuyRogueBuffs(rules, buffs)
		})
	}))
	none("finishBuyBuff", func(a *account.Account, rules *account.Rules, _ int64, replies *server.Outbox) (map[model.ItemID]int64, error) {
		return nil, advanceRogueEvent(a, rules, replies, a.FinishRogueBuffShopping(rules))
	})
	none("settleRoguePve", func(a *account.Account, rules *account.Rules, _ int64, _ *server.Outbox) (map[model.ItemID]int64, error) {
		return nil, a.SettleRoguePVE(rules)
	})
	none("endRoguePve", func(a *account.Account, rules *account.Rules, now int64, _ *server.Outbox) (map[model.ItemID]int64, error) {
		return a.ClaimRogue(rules, true, now)
	})
	none("getRoguePveWeekScoreRwd", func(a *account.Account, rules *account.Rules, now int64, _ *server.Outbox) (map[model.ItemID]int64, error) {
		return a.ClaimRogueWeekly(rules, true, now)
	})
}
