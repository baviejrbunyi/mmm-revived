package system

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/server"
)

var (
	errSystemRules      = fmt.Errorf("%w: system rules unavailable", account.ErrProgression)
	errInvalidPushToken = errors.New("invalid push token")
)

func Ship(srv *server.Service, p *server.Player, r Purchase) error {
	return service{srv}.purchase(p, r)
}

func (s service) system(p *server.Player, action func(*account.Account, int64) error) error {
	if s.Store.Rules == nil {
		return errSystemRules
	}
	return s.Store.Update(p.Account, func(*sql.Tx) error {
		now := time.Now().Unix()
		p.Account.InitializeSystem()
		return action(p.Account, now)
	})
}

func (s service) systemSync(p *server.Player, err error, changed bool, component string) error {
	if err != nil {
		return err
	}
	if !changed {
		return server.Acknowledge(p)
	}
	return s.Sync(p, component)
}

func (s service) protection(p *server.Player, action func(*account.Account, int64) error) error {
	before := p.Account.System
	err := s.system(p, action)
	after := p.Account.System
	changed := before.ProtectionHash != after.ProtectionHash || before.ProtectionReset != after.ProtectionReset || before.ProtectionPIN != after.ProtectionPIN
	return s.systemSync(p, err, changed, "basicComp")
}

func (s service) receipts(p *server.Player, apply func(*account.Account)) error {
	err := s.system(p, func(a *account.Account, _ int64) error {
		apply(a)
		return nil
	})
	if err != nil {
		return err
	}
	return p.Update([]string{"shipComp"}, p.Account.Snapshot(s.Store.Catalog, s.Store.Rules).Purchases)
}

func (s service) purchase(p *server.Player, r Purchase) error {
	if s.Store.Rules == nil {
		return errSystemRules
	}
	before := *p.Account
	var quantities map[model.ItemID]int64
	err := s.system(p, func(a *account.Account, now int64) error {
		if p.State != server.Lobby {
			return fmt.Errorf("%w: purchases are unavailable in battle", account.ErrProgression)
		}
		var err error
		quantities, _, err = a.PurchaseLocalProduct(s.Store.Rules, r.Product, r.Count, r.Kind, r.Reference, now)
		return err
	})
	if err != nil {
		if !errors.Is(err, account.ErrProgression) {
			p.Session.Logger.Error("local purchase failed", "product", r.Product, "error", err)
		}
		return p.Send("onPassThroughCheck", r.Product, r.Count, r.Kind, r.Reference, account.ShipErrorCode(err))
	}
	snapshot := p.Account.Snapshot(s.Store.Catalog, s.Store.Rules)
	if err = s.Sync(p, "itemComp", "heroComp", "taskComp", "activityComp"); err != nil {
		return err
	}
	for _, component := range []struct {
		name  string
		value any
	}{{"shipComp", snapshot.Purchases}, {"popupPackComp", snapshot.PopupPacks}, {"heroPurchaseComp", snapshot.HeroPacks}, {"battlepassComp", snapshot.BattlePass}, {"monthlyPassComp", snapshot.MonthlyPass}} {
		if err = p.Update([]string{component.name}, component.value); err != nil {
			return err
		}
	}
	if err = p.Send("onShipFinish", r.Product, r.Kind, r.Reference, 0, "LOCAL", float64(0)); err != nil {
		return err
	}
	if len(quantities) > 0 {
		return p.ShowRewards(s.ItemRewards(before, p.Account, quantities))
	}
	return nil
}
