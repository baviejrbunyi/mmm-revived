package rogue

import (
	"database/sql"
	"fmt"
	"maps"
	"slices"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

type rogueAction func(a *account.Account, rules *account.Rules, now int64, replies *server.Outbox) (map[model.ItemID]int64, error)

func announceRogueEvent(a *account.Account, rules *account.Rules, replies *server.Outbox) {
	if id, active := a.ActiveRoguePVEEvent(rules); active {
		replies.Add("RoguePveEvent", id)
	}
}

func advanceRogueEvent(a *account.Account, rules *account.Rules, replies *server.Outbox, err error) error {
	if err == nil {
		announceRogueEvent(a, rules, replies)
	}
	return err
}

func executeRogueEvent(selected int) rogueAction {
	return func(a *account.Account, rules *account.Rules, _ int64, replies *server.Outbox) (map[model.ItemID]int64, error) {
		results, err := a.ExecuteRogueEvent(rules, selected)
		if err != nil {
			return nil, err
		}
		for _, result := range results {
			switch result.Kind {
			case "token":
				replies.Add("RogueEventGainToken", result.Token)
			case "buff":
				replies.Add("RogueEventGainBuff", result.Buffs)
			case "lose_buff":
				replies.Add("RogueEventLoseBuff", result.Buffs)
			case "heal":
				replies.Add("RogueEventHeal", result.Health)
			case "resurrect":
				replies.Add("RogueEventResurrection", result.Heroes)
			}
		}
		announceRogueEvent(a, rules, replies)
		return nil, nil
	}
}

func Sync(p *server.Player) error {
	if err := p.Update([]string{"rogueComp"}, p.Account.RogueComponent(false)); err != nil {
		return err
	}
	return p.Update([]string{"roguePveComp"}, p.Account.RogueComponent(true))
}

func (s service) rogue(p *server.Player, action rogueAction) error {
	if p.State != server.Lobby || s.Store.Rules == nil {
		return fmt.Errorf("%w: rogue is unavailable during a battle", account.ErrProgression)
	}
	rules := s.Store.Rules
	var replies server.Outbox
	var values map[model.ItemID]int64
	before := *p.Account
	var shop shopState
	err := s.Store.Update(p.Account, func(*sql.Tx) error {
		now := time.Now().Unix()
		p.Account.InitializeRogue(rules, now)
		shop = captureShop(p.Account.RogueProgress.Run)
		var err error
		values, err = action(p.Account, rules, now, &replies)
		return err
	})
	if err != nil {
		return err
	}
	if err = s.Sync(p, "itemComp", "heroComp", "taskComp"); err != nil {
		return err
	}
	if err = Sync(p); err != nil {
		return err
	}
	if err = shopUpdates(p, shop, p.Account.RogueProgress.Run); err != nil {
		return err
	}
	if err = p.Flush(replies); err != nil {
		return err
	}
	if len(values) > 0 {
		return p.ShowRewards(s.ItemRewards(before, p.Account, values))
	}
	return nil
}

type shopState struct {
	status    model.RogueStatus
	refreshes int
	shop      map[int]model.RogueShopEntry
	relics    map[model.InstanceID]int
}

func captureShop(run model.RogueRun) shopState {
	state := shopState{status: run.Status, refreshes: run.RefreshCount, shop: maps.Clone(run.Shop), relics: make(map[model.InstanceID]int, len(run.Shop))}
	for _, entry := range run.Shop {
		if object, exists := run.Objects[entry.Object]; exists {
			state.relics[entry.Object] = object.ID
		}
	}
	return state
}

func shopUpdates(p *server.Player, before shopState, after model.RogueRun) error {
	if after.Status != model.RogueShopping {
		return nil
	}
	refresh := before.status != model.RogueShopping || before.refreshes != after.RefreshCount || len(before.shop) != len(after.Shop)
	var sold []int
	for _, index := range slices.Sorted(maps.Keys(after.Shop)) {
		entry, previous := after.Shop[index], before.shop[index]
		switch {
		case entry.Status == 3 && previous.Status != 3:
			sold = append(sold, index)
		case entry.Status != 3 && (entry.Object != previous.Object || after.Objects[entry.Object].ID != before.relics[previous.Object]):
			refresh = true
		}
	}
	if refresh {
		return p.Send("onRefreshRogueShop")
	}
	for _, index := range sold {
		if err := p.Send("onRelicShopStChange", index, 3); err != nil {
			return err
		}
	}
	return nil
}
