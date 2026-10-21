package inventory

import (
	"time"

	"mmm-revived/internal/handlers/hero"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

type service struct{ *server.Service }

func Register(srv *server.Service, r *server.Router) {
	s := service{srv}
	batch := rpc.NewRouter[*inventory](nil)
	operations := rpc.NewRouter[*inventory](nil)
	for _, register := range []*inventoryRouter{batch, operations} {
		register.Handle("doCraft", rpc.Bind(DecodeCraft, s.craft))
		register.Handle("useItem", rpc.Bind(DecodeUseItem, s.useItem))
	}
	batch.Handle("talentHero", rpc.Bind(hero.DecodeHeroValue, func(i *inventory, r hero.HeroValue) error {
		return i.account.TalentHero(i.rules, r.Hero, r.Value)
	}))
	batch.Handle("upHeroLv", rpc.Bind(hero.DecodeHeroLevel, func(i *inventory, r hero.HeroLevel) error {
		if err := i.account.UpgradeHero(i.rules, r.Hero, r.Levels); err != nil {
			return err
		}
		i.Add("onUpHeroLv", 0, r.Hero)
		return nil
	}))
	batch.Handle("upHighlight", rpc.Bind(hero.DecodeInstanceLevel, func(i *inventory, r hero.InstanceLevel) error {
		return i.account.UpgradeHighlight(i.rules, r.Instance, r.Levels)
	}))
	operations.Handle("getHeroResetItems", rpc.Bind(hero.DecodeHero, func(i *inventory, hero model.HeroID) error {
		quantities, err := i.account.HeroResetItems(i.rules, hero)
		return i.preview(quantities, err, "onGetHeroResetItems")
	}))
	operations.Handle("getHighlightResetItems", rpc.Bind(DecodeInstance, func(i *inventory, id model.InstanceID) error {
		quantities, err := i.account.HighlightResetItems(i.rules, id)
		return i.preview(quantities, err, "onGetHighlightResetItems")
	}))
	operations.Handle("resetHeroAwake", rpc.Bind(hero.DecodeHero, func(i *inventory, hero model.HeroID) error {
		return s.reward(i, func() (map[model.ItemID]int64, error) {
			return i.account.ResetHeroAwake(i.rules, hero, time.Now().Unix())
		}, showReset, "onHeroAwakeReset", hero)
	}))
	operations.Handle("freeResetHero", rpc.Bind(DecodeFreeHeroReset, func(i *inventory, r FreeReset[model.HeroID]) error {
		return s.freeReset(i, r.Activity, func() (map[model.ItemID]int64, error) { return i.account.FreeResetHero(i.rules, r.Target) })
	}))
	operations.Handle("freeResetHighlight", rpc.Bind(DecodeFreeHighlightReset, func(i *inventory, r FreeReset[model.InstanceID]) error {
		return s.freeReset(i, r.Activity, func() (map[model.ItemID]int64, error) { return i.account.FreeResetHighlight(i.rules, r.Target) })
	}))
	operations.Handle("decArtifacts", rpc.Bind(DecodeInstances, func(i *inventory, ids []model.InstanceID) error {
		return s.reward(i, func() (map[model.ItemID]int64, error) { return i.account.SalvageGear(i.rules, ids, false) }, showNormal, "")
	}))
	operations.Handle("decHighlights", rpc.Bind(DecodeInstances, func(i *inventory, ids []model.InstanceID) error {
		return s.reward(i, func() (map[model.ItemID]int64, error) { return i.account.SalvageGear(i.rules, ids, true) }, showNormal, "onDestroyHighlights", ids)
	}))
	operations.Handle("awakeHighlight", rpc.Bind(DecodeHighlightAwakening, func(i *inventory, r HighlightAwakening) error {
		if err := i.account.AwakeHighlight(i.rules, r.Instance, r.Consumed, r.Materials); err != nil {
			return err
		}
		i.Add("onAwakeHighlight", r.Instance, i.account.Highlights[r.Instance].Awake)
		return nil
	}))
	operations.Handle("refineArtifact", rpc.Bind(DecodeRefinement, func(i *inventory, r ArtifactRefinement) error {
		return i.then(i.account.RefineArtifact(i.rules, r.Instance, r.Locked), "onArtifactChange", r.Instance)
	}))
	operations.Handle("confirmArtifactRefine", rpc.Bind(DecodeInstance, func(i *inventory, id model.InstanceID) error {
		return i.then(i.account.ConfirmArtifactRefine(id), "onArtifactChange", id)
	}))
	operations.Handle("gainHeroProfRwd", rpc.Bind(DecodeProficiency, func(i *inventory, r ProficiencyRewards) error {
		return s.reward(i, func() (map[model.ItemID]int64, error) { return i.account.ClaimProficiency(i.rules, r.Heroes, r.Level) }, showProficiency, "onGainProfRwd", r.Heroes[0], r.Level)
	}))
	operations.Handle("gainHeroProfRwdList", rpc.Bind(DecodeProficiencyList, func(i *inventory, r ProficiencyRewards) error {
		return s.reward(i, func() (map[model.ItemID]int64, error) { return i.account.ClaimProficiency(i.rules, r.Heroes, r.Level) }, showNormal, "onGainProfRwdList", r.Heroes)
	}))
	operations.Handle("buyEnergy", rpc.Bind(DecodeEnergyCount, func(i *inventory, count int) error {
		return i.then(i.account.BuyEnergy(i.rules, count, time.Now().Unix()), "onBuyEnergy", true)
	}))
	operations.Handle("transEnergyBank", rpc.Bind(DecodeAmount, func(i *inventory, amount int64) error {
		return i.then(i.account.TransferEnergy(amount), "onTransEnergyBank", true)
	}))
	operations.Handle("convertYuanbao", rpc.Bind(DecodeAmount, func(i *inventory, amount int64) error {
		return i.then(i.account.ConvertYuanbao(i.rules, amount), "convertYuanbaoFinish", amount)
	}))
	operations.Handle("buyCommodity", rpc.Bind(DecodeCommodity, func(i *inventory, r CommodityPurchase) error {
		before := *i.account
		quantities, err := i.account.BuyCommodity(i.rules, r.ID, r.Count, time.Now().Unix(), i.guildLevel)
		if err != nil {
			return err
		}
		i.Add("onBuyCommodity", r.ID, r.Count, true)
		i.Add("showReward", []model.RewardID{}, s.ItemRewards(before, i.account, quantities), showNormal)
		return nil
	}))
	for name := range operations.Names() {
		components, guild := inventoryComponents, false
		switch name {
		case "buyEnergy", "convertYuanbao":
			components = currencyComponents
		case "transEnergyBank":
			components = energyComponents
		case "buyCommodity":
			guild = true
		}
		route, _ := operations.Route(name)
		r.Handle(name, s.inventoryEndpoint(route.Endpoint, components, guild))
	}
	r.Handle("batchRpc", func(c protocol.Call) (rpc.Handler[*server.Player], error) {
		handlers, err := protocol.DecodeBatch(c, func(call protocol.Call) (rpc.Handler[*inventory], error) {
			route, ok := batch.Route(call.Name)
			if !ok {
				return nil, errBatchMethod
			}
			return route.Endpoint(call)
		})
		if err != nil {
			return nil, err
		}
		return func(p *server.Player) error { return s.runInventory(p, inventoryComponents, false, handlers...) }, nil
	})
}
