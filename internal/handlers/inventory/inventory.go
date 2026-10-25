package inventory

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/protocol"
	"mmm-revived/internal/rpc"
	"mmm-revived/internal/server"
)

var (
	errInventoryRules     = errors.New("inventory rules are unavailable")
	errBatchMethod        = errors.New("method is not allowed in an inventory batch")
	inventoryComponents   = []string{"itemComp", "heroComp", "taskComp", "shopComp"}
	currencyComponents    = []string{"itemComp", "taskComp"}
	energyComponents      = []string{"itemComp"}
	errResetActivityClose = fmt.Errorf("%w: reset activity is not open", account.ErrProgression)
)

const (
	showNormal      = 1
	showProficiency = 9
	showReset       = 13
)

type inventory struct {
	account    *account.Account
	rules      *account.Rules
	guildLevel int
	server.Outbox
}

type inventoryRouter = rpc.Router[*inventory]

func (s service) inventoryEndpoint(endpoint rpc.Endpoint[*inventory], components []string, guild bool) rpc.Endpoint[*server.Player] {
	return func(c protocol.Call) (rpc.Handler[*server.Player], error) {
		handler, err := endpoint(c)
		if err != nil {
			return nil, err
		}
		return func(p *server.Player) error { return s.runInventory(p, components, guild, handler) }, nil
	}
}

func (s service) runInventory(p *server.Player, components []string, guild bool, operations ...rpc.Handler[*inventory]) error {
	if s.Store.Rules == nil {
		return errInventoryRules
	}
	i := &inventory{account: p.Account, rules: s.Store.Rules}
	if guild {
		state, err := s.Store.Social.GuildSnapshot(p.Account.UID)
		if err != nil {
			return err
		}
		i.guildLevel = state.Level
	}
	err := s.Store.Update(p.Account, func(*sql.Tx) error {
		for _, operation := range operations {
			if err := operation(i); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err = s.Sync(p, components...); err != nil {
		return err
	}
	return p.Flush(i.Outbox)
}

func (i *inventory) then(err error, method string, args ...any) error {
	if err != nil {
		return err
	}
	i.Add(method, args...)
	return nil
}

func (i *inventory) preview(quantities map[model.ItemID]int64, err error, method string) error {
	return i.then(err, method, quantities)
}

func (s service) reward(i *inventory, claim func() (map[model.ItemID]int64, error), show int, method string, args ...any) error {
	before := *i.account
	quantities, err := claim()
	if err != nil {
		return err
	}
	if method != "" {
		i.Add(method, args...)
	}
	i.Add("showReward", []model.RewardID{}, s.ItemRewards(before, i.account, quantities), show)
	return nil
}

func (s service) freeReset(i *inventory, activity model.ActivityID, reset func() (map[model.ItemID]int64, error)) error {
	before := *i.account
	if id, _, _ := i.rules.FreeResetActivity(); !i.account.ResetActivityOpen(i.rules, time.Now().Unix()) || activity != id {
		return errResetActivityClose
	}
	quantities, err := reset()
	if err != nil {
		return err
	}
	i.Add("onFreeResetHeroResetHero", quantities)
	i.Add("showReward", []model.RewardID{}, s.ItemRewards(before, i.account, quantities), showReset)
	return nil
}

func (s service) craft(i *inventory, r CraftItem) error {
	before := *i.account
	quantities, err := i.account.Craft(i.rules, r.ID, r.Count, r.Materials, r.Up, r.Position)
	if err != nil || !r.Notify {
		return err
	}
	show := 0
	if r.Show {
		show = 1
	}
	i.Add("onDoCraft", r.ID, show, 0, s.ItemRewards(before, i.account, quantities))
	return nil
}

func (s service) useItem(i *inventory, r UseItem) error {
	before := *i.account
	quantities, err := i.account.UseItem(i.rules, r.Item, r.Count, r.Selection)
	if err != nil {
		return err
	}
	i.Add("onUseItem", true)
	if r.Show && len(quantities) > 0 {
		i.Add("showReward", []model.RewardID{}, s.ItemRewards(before, i.account, quantities), showNormal)
	}
	return nil
}
