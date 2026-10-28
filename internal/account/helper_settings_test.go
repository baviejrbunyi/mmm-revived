package account

import (
	"encoding/json"
	"reflect"
	"testing"

	"mmm-revived/internal/model"
)

func TestHelperDrawOverridesApplyOnlyToSender(t *testing.T) {
	rules := &Rules{
		Draw: DrawRules{
			Pools: map[model.DrawPoolID]DrawPoolRule{
				1: {Ticket: 99, Level: 1, Sets: []model.DrawSetID{10}, Strategies: []model.DrawStrategyID{1}, WeightStrategy: 1},
			},
			Sets: map[model.DrawSetID]DrawSetRule{
				10: {Items: []model.ItemID{2, 3}, Up: []model.ItemID{3}, Weight: 1},
			},
			Strategies: map[model.DrawStrategyID]DrawStrategyRule{
				1: {Counter: 3, Number: 2, Sets: []model.DrawSetID{10}, TriggerSets: []model.DrawSetID{10}, AdvanceNumber: 2},
			},
		},
		ItemRules: map[model.ItemID]ItemRule{
			2: {},
			3: {},
		},
	}
	sender := &Account{Level: 1, Items: map[model.ItemID]int64{99: 1}}
	other := &Account{Level: 1, Items: map[model.ItemID]int64{99: 1}}
	sender.SetHelperDrawFiftyFifty(false)
	item, err := sender.DrawOne(rules, 1, 1)
	if err != nil {
		t.Fatalf("DrawOne with 50/50 disabled: %v", err)
	}
	if item != 3 {
		t.Errorf("sender drew item %d, want featured item 3", item)
	}
	if rules.Draw.Strategies[1].AdvanceNumber != 2 {
		t.Error("sender's override mutated the shared draw rules")
	}

	otherItem, err := other.DrawOne(rules, 1, 1)
	if err != nil {
		t.Fatalf("DrawOne for other account: %v", err)
	}
	if otherItem != 2 && otherItem != 3 {
		t.Errorf("other account drew unexpected item %d", otherItem)
	}
}

func TestHelperDrawSelectionPriceIsAccountScoped(t *testing.T) {
	rules := &Rules{
		Shops: map[model.ShopID]ShopRule{
			3005: {Currency: 402, Prices: []int64{80}, Reward: 1, Level: 1, Limit: 1, Period: 4},
		},
		Rewards: map[model.RewardID][][2]int64{
			1: {{2, 1}},
		},
		ItemRules: map[model.ItemID]ItemRule{2: {}},
	}
	sender := &Account{Level: 1, Items: map[model.ItemID]int64{402: 50}, ShopPurchases: map[model.ShopID]model.ShopPurchase{}}
	other := &Account{Level: 1, Items: map[model.ItemID]int64{402: 50}, ShopPurchases: map[model.ShopID]model.ShopPurchase{}}
	if err := sender.SetHelperDrawSelectCount(50); err != nil {
		t.Fatal(err)
	}
	if _, err := sender.BuyCommodity(rules, 3005, 1, 1, 0); err != nil {
		t.Fatalf("sender should purchase at their configured cost: %v", err)
	}
	if _, err := other.BuyCommodity(rules, 3005, 1, 1, 0); err == nil {
		t.Fatal("other account purchased at the sender's overridden cost")
	}
	if rules.Shops[3005].Prices[0] != 80 {
		t.Errorf("shared shop rule price = %d, want 80", rules.Shops[3005].Prices[0])
	}
}

func TestHelperSettingsAreScopedToAccount(t *testing.T) {
	rules := &Rules{
		Constants: ConstantRules{
			Items:         ClientItems{Energy: 1},
			FormationSize: 3,
		},
		TeamSize: 3,
		TeamLevels: map[int]TeamLevelRule{
			1: {EnergyLimit: 100},
		},
		Economy: Economy{
			FreeDraws:      true,
			DreamstoneShop: true,
			AllBanners:     true,
			Pools:          map[model.DrawPoolID]bool{5: true, 12: true},
		},
		Draw: DrawRules{Pools: map[model.DrawPoolID]DrawPoolRule{
			5:  {End: 100, Ticket: 20},
			12: {End: 100, Ticket: 21},
		}},
		BattleSettings: map[model.BattleID]BattleRule{
			1: {Heroes: 3},
		},
	}

	sender := &Account{
		Level: 1,
		Items: map[model.ItemID]int64{},
	}
	other := &Account{
		Level: 1,
		Items: map[model.ItemID]int64{},
	}
	for _, setting := range []string{"infinite_energy", "free_draws", "dreamstone_shop", "all_banners"} {
		value := setting == "infinite_energy"
		if err := sender.SetHelperBoolean(setting, value); err != nil {
			t.Fatalf("SetHelperBoolean(%q): %v", setting, err)
		}
	}
	featured := []model.DrawPoolID{5}
	sender.SetHelperFeaturedBanners(featured)
	featured[0] = 12
	if err := sender.SetHelperTeamSize(4); err != nil {
		t.Fatalf("SetHelperTeamSize: %v", err)
	}

	sender.ApplyHelperEconomy(rules)
	other.ApplyHelperEconomy(rules)

	if got := sender.Items[rules.Constants.Items.Energy]; got != 100 {
		t.Errorf("sender energy = %d, want 100", got)
	}
	if got := other.Items[rules.Constants.Items.Energy]; got != 0 {
		t.Errorf("other account energy = %d, want 0", got)
	}
	if got := sender.Items[20]; got != 0 {
		t.Errorf("sender free-draw ticket count = %d, want 0", got)
	}
	if got := other.Items[20]; got != freeDrawKeys {
		t.Errorf("other account free-draw ticket count = %d, want %d", got, freeDrawKeys)
	}

	senderDraw := model.DrawComponent{}
	sender.HelperDrawRulesSnapshot(&senderDraw)
	sender.EconomySnapshot(rules, &senderDraw)
	otherDraw := model.DrawComponent{}
	other.HelperDrawRulesSnapshot(&otherDraw)
	other.EconomySnapshot(rules, &otherDraw)
	if senderDraw.DreamstoneShop != 0 || otherDraw.DreamstoneShop != 1 {
		t.Errorf("Dreamstone shop visibility: sender=%d, other=%d; want 0, 1", senderDraw.DreamstoneShop, otherDraw.DreamstoneShop)
	}
	if senderDraw.Open[5] != 1 || senderDraw.Open[12] != 0 {
		t.Errorf("sender banner visibility = %v, want pool 5 open and 12 closed", senderDraw.Open)
	}
	if otherDraw.Open[5] != 1 || otherDraw.Open[12] != 1 {
		t.Errorf("other account banner visibility = %v, want both pools open", otherDraw.Open)
	}

	if limit, custom := sender.BattleHeroLimit(rules, 1); limit != 4 || !custom {
		t.Errorf("sender battle limit = %d, custom=%t; want 4, true", limit, custom)
	}
	if limit, custom := other.BattleHeroLimit(rules, 1); limit != 3 || custom {
		t.Errorf("other account battle limit = %d, custom=%t; want 3, false", limit, custom)
	}

	if err := sender.SetHelperDrawPity(40); err != nil {
		t.Fatal(err)
	}
	sender.SetHelperDrawFiftyFifty(false)
	if err := sender.SetHelperDrawSelectCount(50); err != nil {
		t.Fatal(err)
	}
	if got := sender.HelperDrawStrategy(DrawStrategyRule{Counter: 3, Number: 80}); got.Number != 40 {
		t.Errorf("sender's pity strategy number = %d, want 40", got.Number)
	}
	if got := other.HelperDrawStrategy(DrawStrategyRule{Counter: 3, Number: 80}); got.Number != 80 {
		t.Errorf("other account's pity strategy number = %d, want 80", got.Number)
	}
	if got := sender.HelperDrawStrategy(DrawStrategyRule{Counter: 3, Number: 80, AdvanceNumber: 2}); got.AdvanceNumber != 1 {
		t.Errorf("sender's 50/50 guarantee interval = %d, want 1", got.AdvanceNumber)
	}
	if got := other.HelperDrawStrategy(DrawStrategyRule{Counter: 3, Number: 80, AdvanceNumber: 2}); got.AdvanceNumber != 2 {
		t.Errorf("other account's 50/50 guarantee interval = %d, want 2", got.AdvanceNumber)
	}
	if got := sender.HelperShopRule(3005, ShopRule{Prices: []int64{80}}); got.Prices[0] != 50 {
		t.Errorf("sender's selection cost = %d, want 50", got.Prices[0])
	}
	if got := other.HelperShopRule(3005, ShopRule{Prices: []int64{80}}); got.Prices[0] != 80 {
		t.Errorf("other account's selection cost = %d, want 80", got.Prices[0])
	}

	sender.SetHelperFeaturedBanners(nil)
	resetDraw := model.DrawComponent{}
	sender.HelperDrawRulesSnapshot(&resetDraw)
	sender.EconomySnapshot(rules, &resetDraw)
	if resetDraw.HelperDrawPity != 40 || resetDraw.HelperFiftyFifty != 2 || resetDraw.HelperSelectCount != 50 {
		t.Errorf("draw setting snapshot = pity:%d 50/50:%d select:%d", resetDraw.HelperDrawPity, resetDraw.HelperFiftyFifty, resetDraw.HelperSelectCount)
	}
	if resetDraw.Open[5] != 1 || resetDraw.Open[12] != 1 {
		t.Errorf("clearing banner overrides did not restore server defaults: %v", resetDraw.Open)
	}

	allBannersOff := &Account{}
	if err := allBannersOff.SetHelperBoolean("all_banners", false); err != nil {
		t.Fatal(err)
	}
	normalRotation := model.DrawComponent{}
	allBannersOff.EconomySnapshot(rules, &normalRotation)
	if len(normalRotation.Open) != 0 {
		t.Errorf("all_banners=false forced a static banner list instead of calendar rotation: %v", normalRotation.Open)
	}
	if !allBannersOff.drawPoolOpen(rules, 5, rules.Draw.Pools[5], 50) {
		t.Error("all_banners=false did not preserve a banner open by the calendar")
	}
}

func TestHelperSettingsPersistAndValidate(t *testing.T) {
	account := Account{}
	if err := account.SetHelperBoolean("unknown", true); err == nil {
		t.Fatal("SetHelperBoolean accepted an unknown setting")
	}
	if err := account.SetHelperTeamSize(0); err == nil {
		t.Fatal("SetHelperTeamSize accepted a team size below 1")
	}
	if err := account.SetHelperTeamSize(6); err == nil {
		t.Fatal("SetHelperTeamSize accepted a team size above 5")
	}

	if err := account.SetHelperBoolean("infinite_energy", false); err != nil {
		t.Fatal(err)
	}
	if err := account.SetHelperBoolean("free_draws", true); err != nil {
		t.Fatal(err)
	}
	if err := account.SetHelperBoolean("dreamstone_shop", false); err != nil {
		t.Fatal(err)
	}
	if err := account.SetHelperBoolean("all_banners", false); err != nil {
		t.Fatal(err)
	}
	account.SetHelperFeaturedBanners([]model.DrawPoolID{5, 12})
	if err := account.SetHelperTeamSize(4); err != nil {
		t.Fatal(err)
	}
	if err := account.SetHelperDrawPity(80); err != nil {
		t.Fatal(err)
	}
	account.SetHelperDrawFiftyFifty(true)
	if err := account.SetHelperDrawSelectCount(80); err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(account)
	if err != nil {
		t.Fatalf("marshal account: %v", err)
	}
	var restored Account
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal account: %v", err)
	}
	if !reflect.DeepEqual(restored.HelperSettings, account.HelperSettings) {
		t.Errorf("restored Helper settings = %+v, want %+v", restored.HelperSettings, account.HelperSettings)
	}

	for _, invalid := range []int{0, 81} {
		if err := account.SetHelperDrawPity(invalid); err == nil {
			t.Errorf("SetHelperDrawPity(%d) succeeded", invalid)
		}
	}
	for _, invalid := range []int{0, 1001} {
		if err := account.SetHelperDrawSelectCount(invalid); err == nil {
			t.Errorf("SetHelperDrawSelectCount(%d) succeeded", invalid)
		}
	}
}
